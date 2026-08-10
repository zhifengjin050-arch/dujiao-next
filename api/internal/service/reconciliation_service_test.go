package service

import (
	"testing"

	"github.com/dujiao-next/internal/constants"
)

func TestIsStatusConsistentCanceledWithRefundedStatuses(t *testing.T) {
	if !isStatusConsistent(constants.ProcurementStatusCanceled, "refunded") {
		t.Fatalf("expected canceled/refunded to be consistent")
	}
	if !isStatusConsistent(constants.ProcurementStatusCanceled, "partially_refunded") {
		t.Fatalf("expected canceled/partially_refunded to be consistent")
	}
	if !isStatusConsistent(constants.ProcurementStatusCanceled, "  CaNcElLeD  ") {
		t.Fatalf("expected canceled/cancelled(case-insensitive) to be consistent")
	}
}

func TestIsStatusConsistentFulfilledWithRefundedStatuses(t *testing.T) {
	if !isStatusConsistent(constants.ProcurementStatusFulfilled, "partially_refunded") {
		t.Fatalf("expected fulfilled/partially_refunded to be consistent")
	}
	if !isStatusConsistent(constants.ProcurementStatusCompleted, "refunded") {
		t.Fatalf("expected completed/refunded to be consistent")
	}
}

func TestIsStatusConsistentAcceptedWithRefundedStatuses(t *testing.T) {
	if isStatusConsistent(constants.ProcurementStatusAccepted, "partially_refunded") {
		t.Fatalf("expected accepted/partially_refunded to be mismatched")
	}
	if isStatusConsistent(constants.ProcurementStatusSubmitted, "refunded") {
		t.Fatalf("expected submitted/refunded to be mismatched")
	}
}
