package remote

import (
	"github.com/urfave/cli"
)

var Add = cli.Command{
	Name:  "add",
	Usage: "Add a new remote to list. Format:[name] [user:password]@[scheme]://[host]:[port]",
	Action: func(c *cli.Context) error {

		return nil
	},
}
