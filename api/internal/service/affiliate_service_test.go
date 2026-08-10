package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestResolveOrderAffiliateSnapshotPreferLatestVisitorClick(t *testing.T) {
	svc, db := setupAffiliateServiceTest(t)

	promoterA := createAffiliateTestUser(t, db, "affiliate-a@example.com")
	promoterB := createAffiliateTestUser(t, db, "affiliate-b@example.com")
	profileA := createAffiliateTestProfile(t, db, promoterA.ID, "AFFA0001", constants.AffiliateProfileStatusActive)
	profileB := createAffiliateTestProfile(t, db, promoterB.ID, "AFFB0002", constants.AffiliateProfileStatusActive)

	visitorKey := "visitor-key-priority"
	now := time.Now()
	createAffiliateTestClick(t, db, profileA.ID, visitorKey, now.Add(-2*time.Hour))
	createAffiliateTestClick(t, db, profileB.ID, visitorKey, now.Add(-1*time.Hour))

	profileID, code, err := svc.ResolveOrderAffiliateSnapshot(0, profileA.AffiliateCode, visitorKey)
	if err != nil {
		t.Fatalf("resolve snapshot failed: %v", err)
	}
	if profileID == nil || *profileID != profileB.ID {
		t.Fatalf("expected latest clicked profile %d, got %+v", profileB.ID, profileID)
	}
	if code != profileB.AffiliateCode {
		t.Fatalf("expected latest clicked code %s, got %s", profileB.AffiliateCode, code)
	}
}

func TestResolveOrderAffiliateSnapshotFallbackToCodeWhenNoVisitorClick(t *testing.T) {
	svc, db := setupAffiliateServiceTest(t)

	promoter := createAffiliateTestUser(t, db, "affiliate-fallback@example.com")
	profile := createAffiliateTestProfile(t, db, promoter.ID, "AFFF0003", constants.AffiliateProfileStatusActive)

	profileID, code, err := svc.ResolveOrderAffiliateSnapshot(0, profile.AffiliateCode, "visitor-key-not-found")
	if err != nil {
		t.Fatalf("resolve snapshot failed: %v", err)
	}
	if profileID == nil || *profileID != profile.ID {
		t.Fatalf("expected fallback profile %d, got %+v", profile.ID, profileID)
	}
	if code != profile.AffiliateCode {
		t.Fatalf("expected fallback code %s, got %s", profile.AffiliateCode, code)
	}
}

func TestResolveOrderAffiliateSnapshotRejectSelfByVisitorClick(t *testing.T) {
	svc, db := setupAffiliateServiceTest(t)

	promoter := createAffiliateTestUser(t, db, "affiliate-self@example.com")
	profile := createAffiliateTestProfile(t, db, promoter.ID, "AFFS0004", constants.AffiliateProfileStatusActive)
	createAffiliateTestClick(t, db, profile.ID, "visitor-key-self", time.Now().Add(-10*time.Minute))

	profileID, code, err := svc.ResolveOrderAffiliateSnapshot(promoter.ID, "AFFF9999", "visitor-key-self")
	if err != nil {
		t.Fatalf("resolve snapshot failed: %v", err)
	}
	if profileID != nil || code != "" {
		t.Fatalf("expected self-order attribution ignored, got profile=%+v code=%q", profileID, code)
	}
}

func TestUpdateAffiliateProfileStatus(t *testing.T) {
	svc, db := setupAffiliateServiceTest(t)

	user := createAffiliateTestUser(t, db, "affiliate-status@example.com")
	profile := createAffiliateTestProfile(t, db, user.ID, "AFFST001", constants.AffiliateProfileStatusActive)

	disabled, err := svc.UpdateAffiliateProfileStatus(profile.ID, constants.AffiliateProfileStatusDisabled)
	if err != nil {
		t.Fatalf("disable profile failed: %v", err)
	}
	if disabled == nil || disabled.Status != constants.AffiliateProfileStatusDisabled {
		t.Fatalf("expected disabled status, got %+v", disabled)
	}

	enabled, err := svc.UpdateAffiliateProfileStatus(profile.ID, constants.AffiliateProfileStatusActive)
	if err != nil {
		t.Fatalf("enable profile failed: %v", err)
	}
	if enabled == nil || enabled.Status != constants.AffiliateProfileStatusActive {
		t.Fatalf("expected active status, got %+v", enabled)
	}
}

func TestBatchUpdateAffiliateProfileStatus(t *testing.T) {
	svc, db := setupAffiliateServiceTest(t)

	userA := createAffiliateTestUser(t, db, "affiliate-batch-a@example.com")
	userB := createAffiliateTestUser(t, db, "affiliate-batch-b@example.com")
	profileA := createAffiliateTestProfile(t, db, userA.ID, "AFFBT001", constants.AffiliateProfileStatusActive)
	profileB := createAffiliateTestProfile(t, db, userB.ID, "AFFBT002", constants.AffiliateProfileStatusActive)

	updated, err := svc.BatchUpdateAffiliateProfileStatus([]uint{profileA.ID, profileB.ID}, constants.AffiliateProfileStatusDisabled)
	if err != nil {
		t.Fatalf("batch disable failed: %v", err)
	}
	if updated != 2 {
		t.Fatalf("expected updated 2, got %d", updated)
	}

	reloadedA, err := svc.repo.GetProfileByID(profileA.ID)
	if err != nil || reloadedA == nil {
		t.Fatalf("reload profileA failed: %v", err)
	}
	reloadedB, err := svc.repo.GetProfileByID(profileB.ID)
	if err != nil || reloadedB == nil {
		t.Fatalf("reload profileB failed: %v", err)
	}
	if reloadedA.Status != constants.AffiliateProfileStatusDisabled || reloadedB.Status != constants.AffiliateProfileStatusDisabled {
		t.Fatalf("unexpected statuses after batch disable: %s, %s", reloadedA.Status, reloadedB.Status)
	}
}

func setupAffiliateServiceTest(t *testing.T) (*AffiliateService, *gorm.DB) {
	t.Helper()

	dsn := fmt.Sprintf("file:affiliate_service_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.AffiliateProfile{}, &models.AffiliateClick{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	settingRepo := newMockSettingRepo()
	settingSvc := NewSettingService(settingRepo)
	if _, err := settingSvc.UpdateAffiliateSetting(AffiliateSetting{
		Enabled:        true,
		CommissionRate: 20,
	}); err != nil {
		t.Fatalf("init affiliate setting failed: %v", err)
	}

	affiliateRepo := repository.NewAffiliateRepository(db)
	return NewAffiliateService(affiliateRepo, repository.NewUserRepository(db), nil, nil, settingSvc), db
}

func createAffiliateTestUser(t *testing.T, db *gorm.DB, email string) models.User {
	t.Helper()

	row := models.User{
		Email:        email,
		PasswordHash: "hash",
		DisplayName:  "tester",
		Status:       constants.UserStatusActive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	return row
}

func createAffiliateTestProfile(t *testing.T, db *gorm.DB, userID uint, code, status string) models.AffiliateProfile {
	t.Helper()

	row := models.AffiliateProfile{
		UserID:        userID,
		AffiliateCode: code,
		Status:        status,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("create affiliate profile failed: %v", err)
	}
	return row
}

func createAffiliateTestClick(t *testing.T, db *gorm.DB, profileID uint, visitorKey string, createdAt time.Time) {
	t.Helper()

	row := models.AffiliateClick{
		AffiliateProfileID: profileID,
		VisitorKey:         visitorKey,
		LandingPath:        "/",
		CreatedAt:          createdAt,
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("create affiliate click failed: %v", err)
	}
}
