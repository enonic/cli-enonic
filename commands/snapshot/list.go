package snapshot

import (
	"fmt"
	"github.com/urfave/cli"
	"enonic.com/xp-cli/util"
	"os"
	"enonic.com/xp-cli/commands/common"
)

var List = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Usage:   "Returns a list of existing snapshots with name and status.",
	Flags:   common.FLAGS,
	Action: func(c *cli.Context) error {

		snapshots := listSnapshots(c)
		if js, err := util.PrettyPrintJSON(snapshots); err == nil {
			fmt.Println(js)
		}

		return nil
	},
}

func listSnapshots(c *cli.Context) *SnapshotList {
	req := common.CreateRequest(c, "GET", "api/repo/snapshot/list", nil)

	fmt.Fprint(os.Stderr, "Loading snapshots...")
	resp := common.SendRequest(req)

	var list SnapshotList
	common.ParseResponse(resp, &list)

	fmt.Fprintln(os.Stderr, "Done")

	return &list
}

type SnapshotList struct {
	Results []Snapshot `json:results`
}
