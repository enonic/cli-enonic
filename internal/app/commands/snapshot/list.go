package snapshot

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"fmt"
	"github.com/urfave/cli"
	"os"
)

var List = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Usage:   "Returns a list of existing snapshots with name and status.",
	Flags:   []cli.Flag{common.AUTH_FLAG, common.CRED_FILE_FLAG, common.FORCE_FLAG, common.CLIENT_KEY_FLAG, common.CLIENT_CERT_FLAG},
	Action: func(c *cli.Context) error {

		snapshots := listSnapshots(c)
		fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(snapshots))

		return nil
	},
}

func listSnapshots(c *cli.Context) *SnapshotList {
	req := common.CreateRequest(c, "GET", "repo/snapshot/list", nil)

	resp := common.SendRequest(c, req, "Loading snapshots")

	var list SnapshotList
	common.ParseResponse(resp, &list)

	fmt.Fprintln(os.Stderr, "Done")

	return &list
}

type SnapshotList struct {
	Results []Snapshot `json:"results"`
}
