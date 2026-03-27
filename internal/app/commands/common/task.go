package common

import (
	"cli-enonic/internal/app/util"
	"encoding/json"
	"fmt"
	"github.com/urfave/cli"
	"gopkg.in/cheggaaa/pb.v1"
	"net/http"
	"os"
	"strings"
	"time"
)

const TASK_FINISHED = "FINISHED"
const TASK_FAILED = "FAILED"
const TASK_WAITING = "WAITING"
const TASK_RUNNING = "RUNNING"

func RunTask(c *cli.Context, req *http.Request, msg string, target interface{}) *TaskStatus {
	return runTask(c, req, msg, target, doDisplayTaskProgress)
}

func RunTaskWithSpinner(c *cli.Context, req *http.Request, msg string, target interface{}) *TaskStatus {
	return runTask(c, req, msg, target, doDisplayTaskSpinner)
}

func runTask(c *cli.Context, req *http.Request, msg string, target interface{}, displayFn func(*cli.Context, string, string, chan<- *TaskStatus)) *TaskStatus {
	resp, err := SendRequestCustom(c, req, "", 3)
	util.Fatal(err, "Request error")

	var result TaskResponse
	ParseResponse(resp, &result)

	return waitForTask(c, result.TaskId, msg, target, displayFn)
}

func DisplayTaskProgress(c *cli.Context, taskId, msg string, target interface{}) *TaskStatus {
	return waitForTask(c, taskId, msg, target, doDisplayTaskProgress)
}

func waitForTask(c *cli.Context, taskId, msg string, target interface{}, displayFn func(*cli.Context, string, string, chan<- *TaskStatus)) *TaskStatus {
	doneCh := make(chan *TaskStatus)

	go displayFn(c, taskId, msg, doneCh)

	status := <-doneCh
	close(doneCh)

	if status != nil && status.State == TASK_FINISHED && status.Progress.Info != "" {
		decoder := json.NewDecoder(strings.NewReader(status.Progress.Info))
		if err := decoder.Decode(target); err != nil {
			fmt.Fprint(os.Stderr, "Error parsing response ", err)
			os.Exit(1)
		}
	}

	return status
}

func doDisplayTaskSpinner(c *cli.Context, taskId, msg string, doneCh chan<- *TaskStatus) {
	dotCount := 0
	fmt.Fprintf(os.Stderr, "\r%s", msg)
	var exitFlag bool
	for {
		time.Sleep(time.Second)
		status, statusOk := fetchTaskStatusWithRetry(c, taskId)
		if statusOk {
			switch status.State {
			case TASK_WAITING:
				if time.Now().Sub(status.StartTime).Seconds() > 120 {
					fmt.Fprintf(os.Stderr, "\n")
					fmt.Fprintf(os.Stderr, "Timeout waiting for a task\n")
					exitFlag = true
				}
			case TASK_FINISHED:
				fmt.Fprintf(os.Stderr, "\r%s...\n", msg)
				exitFlag = true
			case TASK_FAILED:
				fmt.Fprintf(os.Stderr, "\n")
				exitFlag = true
			case TASK_RUNNING:
				dotCount = (dotCount + 1) % 4
				dots := strings.Repeat(".", dotCount)
				padding := strings.Repeat(" ", 3-dotCount)
				fmt.Fprintf(os.Stderr, "\r%s%s%s", msg, dots, padding)
			}
		} else {
			fmt.Fprintf(os.Stderr, "\nLost connection to the server\n")
			exitFlag = true
		}

		if exitFlag {
			doneCh <- status
			break
		}
	}
}

func doDisplayTaskProgress(c *cli.Context, taskId, msg string, doneCh chan<- *TaskStatus) {
	bar := pb.New(100)
	bar.ShowSpeed = false
	bar.ShowCounters = false
	bar.ShowPercent = true
	bar.ShowTimeLeft = false
	bar.ShowElapsedTime = false
	bar.ShowFinalTime = false
	bar.Prefix(msg + " ").SetRefreshRate(time.Second).Start()
	var exitFlag bool
	for {
		time.Sleep(time.Second)
		status, statusOk := fetchTaskStatus(c, taskId)
		if statusOk {
			switch status.State {
			case TASK_WAITING:
				if time.Now().Sub(status.StartTime).Seconds() > 120 {
					fmt.Fprintf(os.Stderr, "Timeout waiting for a task\n")
					exitFlag = true
				}
			case TASK_FINISHED:
				bar.Set(100)
				exitFlag = true
			case TASK_FAILED:
				fmt.Fprintln(os.Stderr, "")
				exitFlag = true
			case TASK_RUNNING:
				if status.Progress.Total != 0 {
					percent := int64(float64(status.Progress.Current) / float64(status.Progress.Total) * 100)
					if percent != bar.Get() {
						bar.Set64(percent)
					}
				}
			}
		} else {
			exitFlag = true
		}

		if exitFlag {
			bar.Finish()
			doneCh <- status
			break
		}
	}
}

func fetchTaskStatus(c *cli.Context, taskId string) (*TaskStatus, bool) {
	req := CreateRequest(c, "GET", "/task/"+taskId, nil)
	resp := SendRequest(c, req, "")
	var taskStatus TaskStatus
	ParseResponse(resp, &taskStatus)
	return &taskStatus, resp.StatusCode == http.StatusOK
}

func fetchTaskStatusWithRetry(c *cli.Context, taskId string) (*TaskStatus, bool) {
	maxRetries := 10
	for attempt := 0; attempt < maxRetries; attempt++ {
		req := CreateRequest(c, "GET", "/task/"+taskId, nil)
		resp := SendRequest(c, req, "")
		if resp.StatusCode == http.StatusOK {
			var taskStatus TaskStatus
			ParseResponse(resp, &taskStatus)
			return &taskStatus, true
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			return nil, false
		}
		if attempt < maxRetries-1 {
			time.Sleep(time.Second)
		}
	}
	return nil, false
}

type TaskResponse struct {
	TaskId string `json:"taskId"`
}

type TaskStatus struct {
	Id          string    `json:"id"`
	Description string    `json:"description"`
	Name        string    `json:"name"`
	State       string    `json:"state"`
	Application string    `json:"application"`
	User        string    `json:"user"`
	StartTime   time.Time `json:"startTime"`
	Progress    struct {
		Current uint32 `json:"current"`
		Total   uint32 `json:"total"`
		Info    string `json:"info"`
	} `json:"progress"`
}
