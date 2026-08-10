package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/logger"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/queue"
	"github.com/dujiao-next/internal/repository"
	"github.com/dujiao-next/internal/upstream"
	"github.com/shopspring/decimal"
)

var (
	ErrReconciliationJobNotFound  = errors.New("reconciliation job not found")
	ErrReconciliationItemNotFound = errors.New("reconciliation item not found")
	ErrReconciliationJobRunning   = errors.New("reconciliation job is already running")
)

// ReconciliationService ????
type ReconciliationService struct {
	jobRepo     repository.ReconciliationJobRepository
	itemRepo    repository.ReconciliationItemRepository
	procRepo    repository.ProcurementOrderRepository
	connSvc     *SiteConnectionService
	queueClient *queue.Client
	notifySvc   *NotificationService
}

// NewReconciliationService ??????
func NewReconciliationService(
	jobRepo repository.ReconciliationJobRepository,
	itemRepo repository.ReconciliationItemRepository,
	procRepo repository.ProcurementOrderRepository,
	connSvc *SiteConnectionService,
	queueClient *queue.Client,
	notifySvc *NotificationService,
) *ReconciliationService {
	return &ReconciliationService{
		jobRepo:     jobRepo,
		itemRepo:    itemRepo,
		procRepo:    procRepo,
		connSvc:     connSvc,
		queueClient: queueClient,
		notifySvc:   notifySvc,
	}
}

// RunReconciliationInput ?????????
type RunReconciliationInput struct {
	ConnectionID   uint      `json:"connection_id" binding:"required"`
	Type           string    `json:"type" binding:"required"`
	TimeRangeStart time.Time `json:"time_range_start" binding:"required"`
	TimeRangeEnd   time.Time `json:"time_range_end" binding:"required"`
}

// CreateAndEnqueue ???????????
func (s *ReconciliationService) CreateAndEnqueue(input RunReconciliationInput) (*models.ReconciliationJob, error) {
	job := &models.ReconciliationJob{
		ConnectionID:   input.ConnectionID,
		Type:           input.Type,
		Status:         constants.ReconciliationJobStatusPending,
		TimeRangeStart: input.TimeRangeStart,
		TimeRangeEnd:   input.TimeRangeEnd,
	}
	if err := s.jobRepo.Create(job); err != nil {
		return nil, fmt.Errorf("create reconciliation job: %w", err)
	}

	if s.queueClient != nil {
		if err := s.queueClient.EnqueueReconciliationRun(queue.ReconciliationRunPayload{
			JobID: job.ID,
		}); err != nil {
			logger.Warnw("reconciliation_enqueue_failed", "job_id", job.ID, "error", err)
		}
	}

	return job, nil
}

// Execute ??????(? worker ??)
func (s *ReconciliationService) Execute(ctx context.Context, jobID uint) error {
	job, err := s.jobRepo.GetByID(jobID)
	if err != nil {
		return fmt.Errorf("get reconciliation job: %w", err)
	}
	if job.Status == constants.ReconciliationJobStatusRunning {
		return ErrReconciliationJobRunning
	}
	if job.Status == constants.ReconciliationJobStatusCompleted {
		return nil // ???,?????
	}

	now := time.Now()
	job.Status = constants.ReconciliationJobStatusRunning
	job.StartedAt = &now
	if err := s.jobRepo.Update(job); err != nil {
		return fmt.Errorf("update job status to running: %w", err)
	}

	if err := s.executeReconciliation(ctx, job); err != nil {
		finishedAt := time.Now()
		job.Status = constants.ReconciliationJobStatusFailed
		job.FinishedAt = &finishedAt
		resultJSON, _ := json.Marshal(map[string]string{"error": err.Error()})
		job.ResultJSON = string(resultJSON)
		_ = s.jobRepo.Update(job)
		return fmt.Errorf("execute reconciliation: %w", err)
	}

	finishedAt := time.Now()
	job.Status = constants.ReconciliationJobStatusCompleted
	job.FinishedAt = &finishedAt
	_ = s.jobRepo.Update(job)

	// ??????,????
	if job.MismatchedCount > 0 {
		s.sendMismatchNotification(job)
	}

	return nil
}

// executeReconciliation ???????????????????
func (s *ReconciliationService) executeReconciliation(ctx context.Context, job *models.ReconciliationJob) error {
	// ??????????
	conn, err := s.connSvc.GetByID(job.ConnectionID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}
	adapter, err := s.connSvc.GetAdapter(conn)
	if err != nil {
		return fmt.Errorf("get adapter: %w", err)
	}

	// ???????????
	procOrders, err := s.procRepo.ListByConnectionAndTimeRange(
		job.ConnectionID, job.TimeRangeStart, job.TimeRangeEnd,
	)
	if err != nil {
		return fmt.Errorf("list procurement orders: %w", err)
	}

	var mismatchItems []models.ReconciliationItem
	var skippedCount int
	var errorCount int

	for _, po := range procOrders {
		if po.UpstreamOrderID == 0 {
			skippedCount++
			continue
		}

		// ????????
		upstreamDetail, err := adapter.GetOrder(ctx, po.UpstreamOrderID)
		if err != nil {
			logger.Warnw("reconciliation_get_upstream_order_failed",
				"job_id", job.ID, "procurement_id", po.ID,
				"upstream_order_id", po.UpstreamOrderID, "error", err)
			errorCount++
			continue
		}

		item := s.compareOrder(job, &po, upstreamDetail)
		if item != nil {
			mismatchItems = append(mismatchItems, *item)
		}
	}

	// ???????
	if len(mismatchItems) > 0 {
		if err := s.itemRepo.BatchCreate(mismatchItems); err != nil {
			return fmt.Errorf("batch create reconciliation items: %w", err)
		}
	}

	// TotalCount ?????????????(?????????????)
	comparedCount := len(procOrders) - skippedCount - errorCount
	job.TotalCount = comparedCount
	job.MismatchedCount = len(mismatchItems)
	job.MatchedCount = comparedCount - job.MismatchedCount

	resultJSON, _ := json.Marshal(map[string]any{
		"total":      job.TotalCount,
		"matched":    job.MatchedCount,
		"mismatched": job.MismatchedCount,
		"skipped":    skippedCount,
		"errors":     errorCount,
	})
	job.ResultJSON = string(resultJSON)

	return nil
}

// compareOrder ????????????,?????(????? nil)?
func (s *ReconciliationService) compareOrder(job *models.ReconciliationJob, po *models.ProcurementOrder, detail *upstream.UpstreamOrderDetail) *models.ReconciliationItem {
	checkStatus := job.Type == constants.ReconciliationTypeStatus || job.Type == constants.ReconciliationTypeFull
	checkAmount := job.Type == constants.ReconciliationTypeAmount || job.Type == constants.ReconciliationTypeFull

	statusMismatch := false
	if checkStatus {
		statusMismatch = !isStatusConsistent(po.Status, detail.Status)
	}

	// ????:???????? vs ?????????
	// ??:??? LocalSellAmount(????)? UpstreamAmount(???)?,??????????
	amountMismatch := false
	var upstreamActualAmount models.Money
	if checkAmount && detail.Amount != "" {
		upstreamDecimal, parseErr := decimal.NewFromString(detail.Amount)
		if parseErr == nil && upstreamDecimal.IsPositive() && po.UpstreamAmount.IsPositive() {
			upstreamActualAmount = models.NewMoneyFromDecimal(upstreamDecimal)
			amountMismatch = !po.UpstreamAmount.Equal(upstreamDecimal)
		}
	}

	var mismatchType string
	if statusMismatch && amountMismatch {
		mismatchType = constants.MismatchTypeBoth
	} else if statusMismatch {
		mismatchType = constants.MismatchTypeStatus
	} else if amountMismatch {
		mismatchType = constants.MismatchTypeAmount
	}

	if mismatchType == "" {
		return nil
	}

	return &models.ReconciliationItem{
		JobID:              job.ID,
		ProcurementOrderID: po.ID,
		LocalOrderNo:       po.LocalOrderNo,
		UpstreamOrderNo:    po.UpstreamOrderNo,
		LocalStatus:        po.Status,
		UpstreamStatus:     detail.Status,
		LocalAmount:        po.UpstreamAmount,
		UpstreamAmount:     upstreamActualAmount,
		MismatchType:       mismatchType,
	}
}

// isStatusConsistent ???????????????????
//
// ??????(????? -> ????):
// 1) completed / fulfilled
//   - ???? delivered / completed / fulfilled(???)?
//   - ?????? refunded / partially_refunded(??????????????)?
//
// 2) canceled
//   - ???? canceled / cancelled(??)?
//   - ?????? refunded / partially_refunded(???????????????)?
//
// 3) pending
//   - ???? pending / paid(?????????)?
//
// 4) submitted / accepted
//   - ???? paid / processing / accepted(?????????)?
//
// 5) failed / rejected
//   - ???? failed / rejected(?????)?
//
// 6) ?????? "fulfilling"
//   - ???? fulfilling / processing / paid(?????)?
//
// ??:
// - ????????????????
// - ????????,??????????????
func isStatusConsistent(localStatus, upstreamStatus string) bool {
	localStatus = strings.ToLower(strings.TrimSpace(localStatus))
	upstreamStatus = strings.ToLower(strings.TrimSpace(upstreamStatus))

	switch localStatus {
	case constants.ProcurementStatusCompleted, constants.ProcurementStatusFulfilled:
		return upstreamStatus == "completed" ||
			upstreamStatus == "delivered" ||
			upstreamStatus == "fulfilled" ||
			upstreamStatus == "refunded" ||
			upstreamStatus == "partially_refunded"
	case constants.ProcurementStatusCanceled:
		return upstreamStatus == "canceled" || upstreamStatus == "cancelled" || upstreamStatus == "refunded" || upstreamStatus == "partially_refunded"
	case constants.ProcurementStatusPending:
		return upstreamStatus == "pending" || upstreamStatus == "paid"
	case constants.ProcurementStatusSubmitted, constants.ProcurementStatusAccepted:
		return upstreamStatus == "paid" || upstreamStatus == "processing" || upstreamStatus == "accepted"
	case constants.ProcurementStatusFailed, constants.ProcurementStatusRejected:
		return upstreamStatus == "failed" || upstreamStatus == "rejected"
	case "fulfilling":
		return upstreamStatus == "fulfilling" || upstreamStatus == "processing" || upstreamStatus == "paid"
	default:
		return localStatus == upstreamStatus
	}
}

// sendMismatchNotification ??????????????
func (s *ReconciliationService) sendMismatchNotification(job *models.ReconciliationJob) {
	if s.notifySvc == nil {
		return
	}
	_ = s.notifySvc.Enqueue(NotificationEnqueueInput{
		EventType: constants.NotificationEventExceptionAlert,
		BizType:   constants.NotificationBizTypeReconciliation,
		BizID:     job.ID,
		Data: map[string]any{
			"message":          fmt.Sprintf("???? #%d ??,?? %d ???", job.ID, job.MismatchedCount),
			"job_id":           job.ID,
			"connection_id":    job.ConnectionID,
			"total_count":      job.TotalCount,
			"mismatched_count": job.MismatchedCount,
		},
	})
}

// GetJob ??????
func (s *ReconciliationService) GetJob(id uint) (*models.ReconciliationJob, error) {
	return s.jobRepo.GetByID(id)
}

// ListJobs ??????
func (s *ReconciliationService) ListJobs(filter repository.ReconciliationJobListFilter) ([]models.ReconciliationJob, int64, error) {
	return s.jobRepo.List(filter)
}

// GetJobItems ????????
func (s *ReconciliationService) GetJobItems(jobID uint, page, pageSize int) ([]models.ReconciliationItem, int64, error) {
	return s.itemRepo.ListByJobID(jobID, page, pageSize)
}

// ResolveItem ???????
func (s *ReconciliationService) ResolveItem(itemID uint, adminID uint, remark string) error {
	item, err := s.itemRepo.GetByID(itemID)
	if err != nil {
		return ErrReconciliationItemNotFound
	}
	now := time.Now()
	item.Resolved = true
	item.ResolvedBy = &adminID
	item.ResolvedAt = &now
	item.Remark = remark
	return s.itemRepo.Update(item)
}
