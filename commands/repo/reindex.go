package repo

import (
	"github.com/urfave/cli"
	"enonic.com/xp-cli/commands/common"
)

var Reindex = cli.Command{
	Name:  "reindex",
	Usage: "Reindex content in search indices for the given repository and branches.",
	Flags: append([]cli.Flag{}, common.FLAGS...),
	Action: func(c *cli.Context) error {

		return nil
	},
}
