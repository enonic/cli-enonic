package sandbox

import (
	"github.com/urfave/cli"
)

var Version = cli.Command{
	Name:    "version ",
	Usage:   "Updates the distribution version.",
	Aliases: []string{"ver"},
	Action: func(c *cli.Context) error {

		return nil
	},
}
