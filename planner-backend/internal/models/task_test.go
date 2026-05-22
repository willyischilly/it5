package models

import "testing"

func TestValidTaskTransition(t *testing.T) {
	cases := []struct {
		old, new string
		ok       bool
	}{
		{TaskStatusPending, TaskStatusInProgress, true},
		{TaskStatusInProgress, TaskStatusCompleted, true},
		{TaskStatusPending, TaskStatusCompleted, false},
		{TaskStatusCompleted, TaskStatusInProgress, false},
		{TaskStatusPending, TaskStatusPending, false},
	}
	for _, c := range cases {
		if ValidTaskTransition(c.old, c.new) != c.ok {
			t.Fatalf("%s -> %s expected %v", c.old, c.new, c.ok)
		}
	}
}
