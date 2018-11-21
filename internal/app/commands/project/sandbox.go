package project

import (
	"github.com/urfave/cli"
)

var Sandbox = cli.Command{
	Name:    "sandbox",
	Aliases: []string{"sbox"},
	Usage:   "Set the default sandbox associated with the current project",
	Action: func(c *cli.Context) error {

		return nil
	},
}
