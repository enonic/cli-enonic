package common

import (
	"cli-enonic/internal/app/util"
	"encoding/json"
	"fmt"
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

func RunTask(req *http.Request, msg string, target interface{}) *TaskStatus {
	resp, err := SendRequestCustom(req, "", 3)
	util.Fatal(err, "Request error")

	var result TaskResponse
	ParseResponse(resp, &result)

	return DisplayTaskProgress(result.TaskId, msg, target)
}

func DisplayTaskProgress(taskId, msg string, target interface{}) *TaskStatus {
	doneCh := make(chan *TaskStatus)

	go doDisplayTaskProgress(taskId, msg, doneCh)

	status := <-doneCh
	close(doneCh)

	if status.State == TASK_FINISHED && status.Progress.Info != "" {
		decoder := json.NewDecoder(strings.NewReader(status.Progress.Info))
		if err := decoder.Decode(target); err != nil {
			fmt.Fprint(os.Stderr, "Error parsing response ", err)
			os.Exit(1)
		}
	}

	return status
}

func doDisplayTaskProgress(taskId, msg string, doneCh chan<- *TaskStatus) {
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
		status, statusOk := fetchTaskStatus(taskId)
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
				var percent int64
				if status.Progress.Total != 0 {
					percent = int64(float64(status.Progress.Current) / float64(status.Progress.Total) * 100)
				}
				if percent != bar.Get() {
					bar.Set64(percent)
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

func fetchTaskStatus(taskId string) (*TaskStatus, bool) {
	req := doCreateRequest("GET", "/task/"+taskId, "", "", nil, false)
	resp := SendRequest(req, "")
	var taskStatus TaskStatus
	ParseResponse(resp, &taskStatus)
	return &taskStatus, resp.StatusCode == http.StatusOK
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
	Progress    struct {
		Current uint32
		Total   uint32
		Info    string
	}
}
