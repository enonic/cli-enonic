package common

import "testing"

func TestRestartAllRunningInstancesMessage(t *testing.T) {
	expected := "Please restart all running instances of XP."
	if RESTART_ALL_RUNNING_INSTANCES_MSG != expected {
		t.Fatalf("expected %q, got %q", expected, RESTART_ALL_RUNNING_INSTANCES_MSG)
	}
}
