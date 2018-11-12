package sandbox

import (
	"github.com/urfave/cli"
)

var New = cli.Command{
	Name:  "new",
	Usage: "Create a new sandbox.",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "version, v",
			Usage: "Use specific distro version.",
		},
	},
	Action: func(c *cli.Context) error {
		//TODO: Download XP distro if necessary
		return nil
	},
}
