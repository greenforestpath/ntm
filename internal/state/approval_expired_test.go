package state

import (
	"testing"
	"time"
)

func TestListExpiredPendingApprovals_Empty(t *testing.T) {
	t.Parallel()

	store := testStore(t)

	approvals, err := store.ListExpiredPendingApprovals()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(approvals) != 0 {
		t.Errorf("expected 0 approvals, got %d", len(approvals))
	}
}

func TestListExpiredPendingApprovals_OnlyExpired(t *testing.T) {
	t.Parallel()

	store := testStore(t)

	now := time.Now().UTC()

	// Create expired pending approval
	expired := &Approval{
		ID:          "appr-expired",
		Action:      "kill",
		Resource:    "session/myproj",
		RequestedBy: "agent-1",
		CreatedAt:   now.Add(-2 * time.Hour),
		ExpiresAt:   now.Add(-1 * time.Hour), // expired 1 hour ago
		Status:      ApprovalPending,
	}
	if err := store.CreateApproval(expired); err != nil {
		t.Fatalf("create expired: %v", err)
	}

	// Create non-expired pending approval
	active := &Approval{
		ID:          "appr-active",
		Action:      "restart",
		Resource:    "session/myproj",
		RequestedBy: "agent-2",
		CreatedAt:   now.Add(-30 * time.Minute),
		ExpiresAt:   now.Add(1 * time.Hour), // still valid
		Status:      ApprovalPending,
	}
	if err := store.CreateApproval(active); err != nil {
		t.Fatalf("create active: %v", err)
	}

	approvals, err := store.ListExpiredPendingApprovals()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(approvals) != 1 {
		t.Fatalf("expected 1 expired approval, got %d", len(approvals))
	}

	if approvals[0].ID != "appr-expired" {
		t.Errorf("expected expired approval id, got %q", approvals[0].ID)
	}
}

func TestListExpiredPendingApprovals_IgnoresNonPending(t *testing.T) {
	t.Parallel()

	store := testStore(t)

	now := time.Now().UTC()

	// Create expired but already approved
	approved := &Approval{
		ID:          "appr-approved",
		Action:      "kill",
		Resource:    "session/test",
		RequestedBy: "agent-1",
		CreatedAt:   now.Add(-2 * time.Hour),
		ExpiresAt:   now.Add(-1 * time.Hour), // expired
		Status:      ApprovalApproved,         // not pending
	}
	if err := store.CreateApproval(approved); err != nil {
		t.Fatalf("create approved: %v", err)
	}

	// Create expired but denied
	denied := &Approval{
		ID:          "appr-denied",
		Action:      "restart",
		Resource:    "session/test",
		RequestedBy: "agent-2",
		CreatedAt:   now.Add(-2 * time.Hour),
		ExpiresAt:   now.Add(-1 * time.Hour), // expired
		Status:      ApprovalDenied,           // not pending
	}
	if err := store.CreateApproval(denied); err != nil {
		t.Fatalf("create denied: %v", err)
	}

	approvals, err := store.ListExpiredPendingApprovals()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(approvals) != 0 {
		t.Errorf("expected 0 expired pending (non-pending should be excluded), got %d", len(approvals))
	}
}

func TestListExpiredPendingApprovals_MultipleExpired(t *testing.T) {
	t.Parallel()

	store := testStore(t)

	now := time.Now().UTC()

	// Create 3 expired pending approvals
	for i, id := range []string{"exp-1", "exp-2", "exp-3"} {
		appr := &Approval{
			ID:          id,
			Action:      "kill",
			Resource:    "pane/" + id,
			RequestedBy: "agent-x",
			CreatedAt:   now.Add(-time.Duration(3-i) * time.Hour),
			ExpiresAt:   now.Add(-time.Duration(i+1) * time.Minute), // all expired
			Status:      ApprovalPending,
		}
		if err := store.CreateApproval(appr); err != nil {
			t.Fatalf("create %s: %v", id, err)
		}
	}

	approvals, err := store.ListExpiredPendingApprovals()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(approvals) != 3 {
		t.Errorf("expected 3 expired approvals, got %d", len(approvals))
	}

	// Verify ordered by created_at
	for i := 1; i < len(approvals); i++ {
		if approvals[i].CreatedAt.Before(approvals[i-1].CreatedAt) {
			t.Errorf("approvals not ordered by created_at: [%d]=%v before [%d]=%v",
				i, approvals[i].CreatedAt, i-1, approvals[i-1].CreatedAt)
		}
	}
}

func TestListExpiredPendingApprovals_FieldsPopulated(t *testing.T) {
	t.Parallel()

	store := testStore(t)

	now := time.Now().UTC()

	appr := &Approval{
		ID:            "full-fields",
		Action:        "force-restart",
		Resource:      "session/production",
		Reason:        "agent stuck",
		RequestedBy:   "orchestrator",
		CorrelationID: "corr-123",
		RequiresSLB:   true,
		CreatedAt:     now.Add(-2 * time.Hour),
		ExpiresAt:     now.Add(-30 * time.Minute),
		Status:        ApprovalPending,
	}
	if err := store.CreateApproval(appr); err != nil {
		t.Fatalf("create: %v", err)
	}

	approvals, err := store.ListExpiredPendingApprovals()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(approvals) != 1 {
		t.Fatalf("expected 1 approval, got %d", len(approvals))
	}

	a := approvals[0]
	if a.ID != "full-fields" {
		t.Errorf("ID: got %q", a.ID)
	}
	if a.Action != "force-restart" {
		t.Errorf("Action: got %q", a.Action)
	}
	if a.Resource != "session/production" {
		t.Errorf("Resource: got %q", a.Resource)
	}
	if a.Reason != "agent stuck" {
		t.Errorf("Reason: got %q", a.Reason)
	}
	if a.RequestedBy != "orchestrator" {
		t.Errorf("RequestedBy: got %q", a.RequestedBy)
	}
	if a.CorrelationID != "corr-123" {
		t.Errorf("CorrelationID: got %q", a.CorrelationID)
	}
	if !a.RequiresSLB {
		t.Error("RequiresSLB: expected true")
	}
	if a.Status != ApprovalPending {
		t.Errorf("Status: got %q", a.Status)
	}
}
