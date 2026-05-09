package common

import "testing"

func TestRestartAllRunningInstancesMessage(t *testing.T) {
	expected := "Please restart XP instance/cluster."
	if RESTART_ALL_RUNNING_INSTANCES_MSG != expected {
		t.Fatalf("expected %q, got %q", expected, RESTART_ALL_RUNNING_INSTANCES_MSG)
	}
}
