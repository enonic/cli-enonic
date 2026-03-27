package common

import (
	"testing"

	"github.com/urfave/cli"
)

func TestWaitForTask_NilStatus(t *testing.T) {
	// Simulates fetchTaskStatusWithRetry returning nil after exhausting retries (e.g. repeated 401s).
	// waitForTask must not panic when the display function sends nil on the channel.
	displayFn := func(c *cli.Context, taskId, msg string, doneCh chan<- *TaskStatus) {
		doneCh <- nil
	}

	var target map[string]interface{}
	status := waitForTask(nil, "fake-task-id", "Loading dump", &target, displayFn)

	if status != nil {
		t.Errorf("expected nil status, got %+v", status)
	}
}

func TestWaitForTask_FinishedStatus(t *testing.T) {
	// Verifies that waitForTask correctly decodes Progress.Info into the target when task finishes.
	displayFn := func(c *cli.Context, taskId, msg string, doneCh chan<- *TaskStatus) {
		status := &TaskStatus{
			State: TASK_FINISHED,
		}
		status.Progress.Info = `{"name":"test"}`
		doneCh <- status
	}

	var target map[string]interface{}
	status := waitForTask(nil, "fake-task-id", "Loading dump", &target, displayFn)

	if status == nil {
		t.Fatal("expected non-nil status")
	}
	if status.State != TASK_FINISHED {
		t.Errorf("expected state FINISHED, got %s", status.State)
	}
	if target["name"] != "test" {
		t.Errorf("expected target name 'test', got %v", target["name"])
	}
}

func TestWaitForTask_FinishedNoInfo(t *testing.T) {
	// When task finishes with empty Info, target should not be decoded.
	displayFn := func(c *cli.Context, taskId, msg string, doneCh chan<- *TaskStatus) {
		status := &TaskStatus{
			State: TASK_FINISHED,
		}
		doneCh <- status
	}

	var target map[string]interface{}
	status := waitForTask(nil, "fake-task-id", "Loading dump", &target, displayFn)

	if status == nil {
		t.Fatal("expected non-nil status")
	}
	if target != nil {
		t.Errorf("expected nil target, got %v", target)
	}
}
