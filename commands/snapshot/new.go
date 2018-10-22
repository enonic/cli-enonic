package snapshot

import (
	"github.com/urfave/cli"
	"net/http"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"enonic.com/xp-cli/commands/common"
)

var New = cli.Command{
	Name:  "new",
	Usage: "Stores a snapshot of the current state of the repository.",
	Flags: append([]cli.Flag{
		cli.StringFlag{
			Name:  "repo, r",
			Usage: "The name of the repository to snapshot",
		},
	}, common.FLAGS...),
	Action: func(c *cli.Context) error {

		req := createNewRequest(c)

		fmt.Fprint(os.Stderr, "Creating snapshot...")
		resp := common.SendRequest(req)

		var snap Snapshot
		if common.ParseResponse(resp, &snap); snap.State == "SUCCESS" {
			fmt.Fprintln(os.Stderr, "Done")
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

	return common.CreateRequest(c, "POST", "api/repo/snapshot", body)
}
