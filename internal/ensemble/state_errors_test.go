package ensemble

import "testing"

func TestStateStore_NilAndValidationErrors(t *testing.T) {
	var store *StateStore

	if err := store.Save(nil); err == nil {
		t.Fatal("expected error for nil store Save")
	}
	if _, err := store.Load(""); err == nil {
		t.Fatal("expected error for nil store Load")
	}
	if err := store.UpdateStatus("", EnsembleActive); err == nil {
		t.Fatal("expected error for nil store UpdateStatus")
	}
	if err := store.UpdateAssignmentStatus("", "", AssignmentActive); err == nil {
		t.Fatal("expected error for nil store UpdateAssignmentStatus")
	}
	if _, err := store.List(); err == nil {
		t.Fatal("expected error for nil store List")
	}
	if err := store.Delete(""); err == nil {
		t.Fatal("expected error for nil store Delete")
	}
}

func TestSessionStore_Errors(t *testing.T) {
	if _, err := LoadSession(""); err == nil {
		t.Fatal("expected error for empty session name")
	}
	if err := SaveSession("", nil); err == nil {
		t.Fatal("expected error for nil session")
	}
}
