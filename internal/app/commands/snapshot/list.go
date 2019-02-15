package snapshot

import (
	"fmt"
	"github.com/urfave/cli"
	"github.com/enonic/enonic-cli/internal/app/util"
	"os"
	"github.com/enonic/enonic-cli/internal/app/commands/common"
)

var List = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Usage:   "Returns a list of existing snapshots with name and status.",
	Flags:   common.FLAGS,
	Action: func(c *cli.Context) error {

		snapshots := listSnapshots(c)
		fmt.Println(util.PrettyPrintJSON(snapshots))

		return nil
	},
}

func listSnapshots(c *cli.Context) *SnapshotList {
	req := common.CreateRequest(c, "GET", "repo/snapshot/list", nil)

	resp := common.SendRequest(req, "Loading snapshots")

	var list SnapshotList
	common.ParseResponse(resp, &list)

	fmt.Fprintln(os.Stderr, "Done")

	return &list
}

type SnapshotList struct {
	Results []Snapshot `json:results`
}
