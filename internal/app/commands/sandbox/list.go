package sandbox

import (
	"github.com/urfave/cli"
)

var List = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Usage:   "List all sandboxes",
	Action: func(c *cli.Context) error {
		//TODO: mention the running one if there is one
		return nil
	},
}
