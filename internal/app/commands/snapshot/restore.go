package snapshot

import (
	"bytes"
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"encoding/json"
	"fmt"
	"github.com/urfave/cli"
	"net/http"
	"os"
	"strings"
)

var Restore = cli.Command{
	Name:  "restore",
	Usage: "Restores a snapshot of a previous state of the repository.",
	Flags: append([]cli.Flag{
		cli.StringFlag{
			Name:  "repo, r",
			Usage: "The name of the repository to restore",
		},
		cli.StringFlag{
			Name:  "snapshot, snap",
			Usage: "The name of the snapshot to restore",
		},
		cli.BoolFlag{
			Name:  "latest",
			Usage: "Flag to use latest snapshot, takes precedence over name flag",
		},
		cli.BoolFlag{
			Name:  "clean",
			Usage: "Delete indices before restoring",
		},
		common.FORCE_FLAG,
	}, append(common.AUTH_AND_TLS_FLAGS, common.COMPAT_FLAG)...),
	Action: func(c *cli.Context) error {

		req := createRestoreRequest(c)
		var result RestoreResult

		if common.IsCompatMode(c) {
			resp, err := common.SendRequestCustom(c, req, "Restoring snapshot", 5)
			util.Fatal(err, "Request error")

			if common.ParseResponse(resp, &result); !result.Failed {
				fmt.Fprintln(os.Stderr, "Done")
				fmt.Fprintln(os.Stderr, common.RESTART_ALL_RUNNING_INSTANCES_MSG)
			} else {
				fmt.Fprintln(os.Stderr, result.Message)
			}
			fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(result))
			return nil
		}

		status := common.RunTaskWithSpinner(c, req, "Restoring snapshot", &result)
		if status == nil {
			return nil
		}

		switch status.State {
		case common.TASK_FINISHED:
			if !result.Failed {
				fmt.Fprintln(os.Stderr, "Done")
				fmt.Fprintln(os.Stderr, common.RESTART_ALL_RUNNING_INSTANCES_MSG)
			} else {
				fmt.Fprintln(os.Stderr, result.Message)
			}
			fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(result))
		case common.TASK_FAILED:
			fmt.Fprintf(os.Stderr, "Failed to restore snapshot: %s\n", status.Progress.Info)
		}

		return nil
	},
}

func ensureSnapshotFlag(c *cli.Context) string {
	return ensureSnapshotFlagWithMessage(c, "Select snapshot to restore")
}

func ensureSnapshotFlagWithMessage(c *cli.Context, message string) string {
	snapName := c.String("snapshot")
	if strings.TrimSpace(snapName) != "" {
		return snapName
	}

	if common.IsForceMode(c) {
		fmt.Fprintln(os.Stderr, "Snapshot name can not be empty in non-interactive mode.")
		os.Exit(1)
	}

	snapshotList := listSnapshots(c)
	if len(snapshotList.Results) == 0 {
		fmt.Fprintln(os.Stderr, "No existing snapshots found")
		os.Exit(1)
	}

	_, pos, err := util.PromptSelect(&util.SelectOptions{
		Message: message,
		Options: getSnapshotDisplayNames(snapshotList),
	})
	util.Fatal(err, "Could not select snapshot: ")

	return snapshotList.Results[pos].Name
}

func createRestoreRequest(c *cli.Context) *http.Request {
	body := new(bytes.Buffer)
	params := map[string]interface{}{}

	if c.Bool("latest") {
		params["latest"] = true
	} else {
		params["snapshotName"] = ensureSnapshotFlag(c)
	}

	if c.Bool("clean") {
		params["force"] = true
	}

	if repo := c.String("repo"); repo != "" {
		params["repository"] = repo
	}
	json.NewEncoder(body).Encode(params)

	return common.CreateRequest(c, "POST", "repo/snapshot/restore", body)
}

func getSnapshotDisplayNames(list *SnapshotList) []string {
	var names []string
	for _, s := range list.Results {
		if s.Timestamp.IsZero() {
			names = append(names, s.Name)
		} else {
			names = append(names, fmt.Sprintf("%s (%s)", s.Name, s.Timestamp.Local().Format("2006-01-02 15:04:05")))
		}
	}
	return names
}

type RestoreResult struct {
	Message string   `json:"message"`
	Name    string   `json:"name"`
	Failed  bool     `json:"failed"`
	Indices []string `json:"indices"`
}
