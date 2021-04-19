package snapshot

import (
	"bytes"
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"encoding/json"
	"fmt"
	"github.com/urfave/cli"
	"gopkg.in/AlecAivazis/survey.v1"
	"net/http"
	"os"
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
	}, common.FLAGS...),
	Action: func(c *cli.Context) error {

		req := createRestoreRequest(c)

		resp, err := common.SendRequestCustom(req, "Restoring snapshot", 5)
		util.Fatal(err, "Request error")

		var result RestoreResult
		if common.ParseResponse(resp, &result); !result.Failed {
			fmt.Fprintln(os.Stderr, "Done")
		} else {
			fmt.Fprintln(os.Stderr, result.Message)
		}
		fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(result))

		return nil
	},
}

func ensureSnapshotFlag(c *cli.Context) string {
	snapName := c.String("snapshot")
	if snapName == "" {

		snapshotList := listSnapshots(c)
		if len(snapshotList.Results) == 0 {
			fmt.Fprintln(os.Stderr, "No existing snapshots found")
			os.Exit(1)
		}

		var name string
		prompt := &survey.Select{
			Message: "Select snapshot to restore",
			Options: getSnapshotNames(snapshotList),
		}
		survey.AskOne(prompt, &name, nil)

		return name
	}
	return snapName
}

func createRestoreRequest(c *cli.Context) *http.Request {
	body := new(bytes.Buffer)
	params := map[string]interface{}{}

	if c.Bool("latest") {
		params["latest"] = true
	} else {
		params["snapshotName"] = ensureSnapshotFlag(c)
	}

	if repo := c.String("repo"); repo != "" {
		params["repository"] = repo
	}
	json.NewEncoder(body).Encode(params)

	return common.CreateRequest(c, "POST", "repo/snapshot/restore", body)
}

func getSnapshotNames(list *SnapshotList) []string {
	var names []string
	for _, s := range list.Results {
		names = append(names, s.Name)
	}
	return names
}

type RestoreResult struct {
	Message string   `json:message`
	Name    string   `json:name`
	Failed  bool     `json:failed`
	Indices []string `json:indices`
}
