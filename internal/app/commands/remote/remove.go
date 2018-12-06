package remote

import (
	"github.com/urfave/cli"
)

var Remove = cli.Command{
	Name:    "remove",
	Aliases: []string{"rm"},
	Usage:   "Remove a remote from list.",
	Action: func(c *cli.Context) error {

		return nil
	},
}
