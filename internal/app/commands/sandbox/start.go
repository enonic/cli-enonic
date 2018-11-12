package sandbox

import (
	"github.com/urfave/cli"
)

var Start = cli.Command{
	Name:  "start",
	Usage: "Start the sandbox.",
	Action: func(c *cli.Context) error {
		//TODO: Download XP distro if necessary
		return nil
	},
}
