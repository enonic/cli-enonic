package remote

import (
	"github.com/urfave/cli"
)

var List = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Usage:   "List all known remotes.",
	Action: func(c *cli.Context) error {

		return nil
	},
}
