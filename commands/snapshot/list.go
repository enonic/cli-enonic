package snapshot

import (
	"fmt"
	"github.com/urfave/cli"
)

var List = cli.Command{
	Name:  "ls",
	Usage: "Returns a list of existing snapshots with name and status.",
	Flags: SNAPSHOT_FLAGS,
	Action: func(c *cli.Context) error {

		req := createRequest(c, "GET", "api/repo/snapshot/list", nil)

		resp := sendRequest(req)

		fmt.Println(parseResponse(resp))

		return nil
	},
}
