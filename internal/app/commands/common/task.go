package common

import (
	"net/http"
	"time"
	"fmt"
	"os"
	"github.com/urfave/cli"
	"encoding/json"
	"strings"
)

const TASK_FINISHED = "FINISHED"
const TASK_FAILED = "FAILED"
const TASK_WAITING = "WAITING"
const TASK_RUNNING = "RUNNING"

func RunTask(c *cli.Context, req *http.Request, msg string, target interface{}) *TaskStatus {
	resp := SendRequest(req)

	var result TaskResponse
	ParseResponse(resp, &result)

	doneCh := make(chan *TaskStatus)

	user, pass, _ := req.BasicAuth()
	go displayTaskProgress(c, result.TaskId, msg, user, pass, doneCh)

	status := <-doneCh
	close(doneCh)

	if status.State == TASK_FINISHED {
		decoder := json.NewDecoder(strings.NewReader(status.Progress.Info))
		if err := decoder.Decode(target); err != nil {
			fmt.Fprint(os.Stderr, "Error parsing response ", err)
			os.Exit(1)
		}
	}

	return status
}

func displayTaskProgress(c *cli.Context, taskId, msg, user, pass string, doneCh chan<- *TaskStatus) {
	var exitFlag bool
	fmt.Fprint(os.Stderr, msg)
	for {
		status := fetchTaskStatus(taskId, user, pass)
		switch status.State {
		case TASK_WAITING:
			if time.Now().Sub(status.StartTime).Minutes() > 5 {
				fmt.Fprintf(os.Stderr, "Timeout waiting for a task\n")
				exitFlag = true
			}
		case TASK_FINISHED:
			fmt.Fprintf(os.Stderr, "\r%s%d %%\n", msg, 100)
			exitFlag = true
		case TASK_FAILED:
			fmt.Fprintln(os.Stderr, "")
			exitFlag = true
		case TASK_RUNNING:
			var percent float64
			if status.Progress.Total != 0 {
				percent = float64(status.Progress.Current) / float64(status.Progress.Total) * 100
			}
			fmt.Fprintf(os.Stderr, "\r%s%.0f %%", msg, percent)
		}

		if !exitFlag {
			time.Sleep(time.Second)
		} else {
			doneCh <- status
			break
		}
	}
}

func fetchTaskStatus(taskId, user, pass string) *TaskStatus {
	req := doCreateRequest("GET", "admin/rest/tasks/"+taskId, user, pass, nil)
	resp := SendRequest(req)
	var taskStatus TaskStatus
	ParseResponse(resp, &taskStatus)
	return &taskStatus
}

type TaskResponse struct {
	TaskId string
}

type TaskStatus struct {
	Id          string
	Description string
	Name        string
	State       string
	Application string
	User        string
	StartTime   time.Time
	Progress struct {
		Current uint32
		Total   uint32
		Info    string
	}
}
