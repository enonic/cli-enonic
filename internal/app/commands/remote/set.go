package remote

import (
	"github.com/urfave/cli"
)

var Set = cli.Command{
	Name:  "set",
	Usage: "Set a remote as default to be used in all remote api queries.",
	Action: func(c *cli.Context) error {

		return nil
	},
}
