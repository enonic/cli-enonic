package sandbox

import (
	"github.com/urfave/cli"
)

var Delete = cli.Command{
	Name:    "delete",
	Usage:   "Delete a sandbox",
	Aliases: []string{"del"},
	Action: func(c *cli.Context) error {
		//TODO: Delete XP distro if not used any more
		return nil
	},
}
