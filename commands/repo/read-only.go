package repo

import (
	"github.com/urfave/cli"
	"enonic.com/xp-cli/commands/common"
)

var ReadOnly = cli.Command{
	Name:  "readonly",
	Usage: "Toggle read-only mode for server or single repository",
	Flags: append([]cli.Flag{}, common.FLAGS...),
	Action: func(c *cli.Context) error {

		return nil
	},
}
