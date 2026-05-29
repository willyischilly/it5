package services

import "testing"

func TestValidateNoDuplicateWorkIDs(t *testing.T) {
	if err := validateNoDuplicateWorkIDs([]AddTaskItem{{WorkID: 1}, {WorkID: 2}}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := validateNoDuplicateWorkIDs([]AddTaskItem{{WorkID: 1}, {WorkID: 1}}); err == nil {
		t.Fatal("expected error for duplicate work_id")
	}
}
