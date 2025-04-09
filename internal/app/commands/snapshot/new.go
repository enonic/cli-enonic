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
	}, common.AUTH_FLAG, common.CRED_FILE_FLAG, common.FORCE_FLAG, common.CLIENT_KEY_FLAG, common.CLIENT_CERT_FLAG),
	Action: func(c *cli.Context) error {

		req := createNewRequest(c)

		resp := common.SendRequest(c, req, "Creating snapshot")

		var snap Snapshot
		if common.ParseResponse(resp, &snap); snap.State == "SUCCESS" {
			fmt.Fprintln(os.Stderr, "Done")
			fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(snap))
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
