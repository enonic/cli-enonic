package dump

import (
	"github.com/urfave/cli"
	"enonic.com/xp-cli/commands/common"
)

var Upgrade = cli.Command{
	Name:    "upgrade",
	Aliases: []string{"up"},
	Usage:   "Upgrade a dump.",
	Flags:   append([]cli.Flag{}, common.FLAGS...),
	Action: func(c *cli.Context) error {

		return nil
	},
}
