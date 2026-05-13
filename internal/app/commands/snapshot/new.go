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
)

var Create = cli.Command{
	Name:  "create",
	Usage: "Stores a snapshot of the current state of the repository.",
	Flags: append([]cli.Flag{
		cli.StringFlag{
			Name:  "repo, r",
			Usage: "The name of the repository to snapshot",
		},
		common.FORCE_FLAG,
	}, append(common.AUTH_AND_TLS_FLAGS, common.COMPAT_FLAG)...),
	Action: func(c *cli.Context) error {

		util.Fatal(common.ValidateCompatFlag(c), "Invalid argument")

		req := createNewRequest(c)
		var snap Snapshot

		if common.IsCompatMode(c) {
			resp := common.SendRequest(c, req, "Creating snapshot")
			if common.ParseResponse(resp, &snap); snap.State == "SUCCESS" {
				fmt.Fprintln(os.Stderr, "Done")
				fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(snap))
			}
			return nil
		}

		status := common.RunTaskWithSpinner(c, req, "Creating snapshot", &snap)
		if status == nil {
			return nil
		}

		switch status.State {
		case common.TASK_FINISHED:
			if snap.State == "SUCCESS" {
				fmt.Fprintln(os.Stderr, "Done")
			} else {
				fmt.Fprintf(os.Stderr, "Snapshot finished with state: %s\n", snap.State)
			}
			fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(snap))
		case common.TASK_FAILED:
			fmt.Fprintf(os.Stderr, "Failed to create snapshot: %s\n", status.Progress.Info)
		}

		return nil
	},
}

func createNewRequest(c *cli.Context) *http.Request {
	body := new(bytes.Buffer)
	params := map[string]interface{}{}
	if repo := c.String("repo"); repo != "" {
		params["repositoryId"] = repo
	}
	json.NewEncoder(body).Encode(params)

	return common.CreateRequest(c, "POST", "repo/snapshot", body)
}
