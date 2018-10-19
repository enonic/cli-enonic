package snapshot

import (
	"github.com/urfave/cli"
	"net/http"
	"fmt"
	"bytes"
	"encoding/json"
	"github.com/manifoldco/promptui"
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
	}, SNAPSHOT_FLAGS...),
	Action: func(c *cli.Context) error {

		ensureSnapshotFlag(c)

		req := createRestoreRequest(c)

		fmt.Fprint(os.Stderr, "Restoring snapshot...")
		resp := sendRequest(req)

		var response RestoreResponse
		if parseResponse(resp, &response); !response.Failed {
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

		prompt := promptui.Select{
			Label: "Select snapshot to restore",
			Items: getSnapshotNames(snapshotList),
			Size:  10,
		}

		_, promptResult, err := prompt.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Prompt failed %v\n", err)
			os.Exit(1)
		}

		c.Set("snapshot", promptResult)
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

	return createRequest(c, "POST", "api/repo/snapshot/restore", body)
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
