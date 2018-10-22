package dump

import (
	"github.com/urfave/cli"
	"enonic.com/xp-cli/commands/common"
)

var Load = cli.Command{
	Name:  "load",
	Usage: "Import data from a dump.",
	Flags: append([]cli.Flag{}, common.FLAGS...),
	Action: func(c *cli.Context) error {

		return nil
	},
}
