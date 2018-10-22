package dump

import (
	"github.com/urfave/cli"
	"enonic.com/xp-cli/commands/common"
)

var New = cli.Command{
	Name:  "new",
	Usage: "Export data from every repository.",
	Flags: append([]cli.Flag{}, common.FLAGS...),
	Action: func(c *cli.Context) error {

		return nil
	},
}
