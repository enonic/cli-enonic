package project

import (
	"github.com/urfave/cli"
)

var Deploy = cli.Command{
	Name:  "deploy",
	Usage: "Deploy current project to a sandbox",
	Action: func(c *cli.Context) error {

		return nil
	},
}
