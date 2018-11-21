package project

import (
	"github.com/urfave/cli"
)

var Install = cli.Command{
	Name:  "install",
	Usage: "Build current project and install it to Enonic XP",
	Action: func(c *cli.Context) error {

		return nil
	},
}
