package snapshot

import (
	"github.com/urfave/cli"
	"net/http"
	"fmt"
	"bytes"
	"encoding/json"
	"os"
	"github.com/enonic/xp-cli/internal/app/commands/common"
	"github.com/AlecAivazis/survey"
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
	}, common.FLAGS...),
	Action: func(c *cli.Context) error {

		ensureSnapshotFlag(c)

		req := createRestoreRequest(c)

		fmt.Fprint(os.Stderr, "Restoring snapshot...")
		resp := common.SendRequest(req)

		var response RestoreResponse
		if common.ParseResponse(resp, &response); !response.Failed {
			fmt.Fprintln(os.Stderr, "Done")
		} else {
			fmt.Fprintln(os.Stderr, response.Message)
		}

		return nil
	},
}

func ensureSnapshotFlag(c *cli.Context) {
	if c.String("snapshot") == "" {

		snapshotList := listSnapshots(c)

		var name string
		prompt := &survey.Select{
			Message: "Select snapshot to restore",
			Options: getSnapshotNames(snapshotList),
		}
		survey.AskOne(prompt, &name, nil)

		c.Set("snapshot", name)
	}
}

func createRestoreRequest(c *cli.Context) *http.Request {
	body := new(bytes.Buffer)
	params := map[string]interface{}{
		"snapshotName": c.String("snapshot"),
	}
	if repo := c.String("repo"); repo != "" {
		params["repository"] = repo
	}
	json.NewEncoder(body).Encode(params)

	return common.CreateRequest(c, "POST", "api/repo/snapshot/restore", body)
}

func getSnapshotNames(list *SnapshotList) []string {
	var names []string
	for _, s := range list.Results {
		names = append(names, s.Name)
	}
	return names
}

type RestoreResponse struct {
	Message string   `json:message`
	Name    string   `json:name`
	Failed  bool     `json:failed`
	Indices []string `json:indices`
}
