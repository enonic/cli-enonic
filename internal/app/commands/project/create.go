package project

import (
	"github.com/urfave/cli"
)

var Create = cli.Command{
	Name:  "create",
	Usage: "Create new project",
	Action: func(c *cli.Context) error {

		return nil
	},
}
