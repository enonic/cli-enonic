package commands

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/commands/project"
	"github.com/urfave/cli"
)

var Dev = cli.Command{
	Name:  "dev",
	Usage: "Start current project in dev mode",
	Flags: []cli.Flag{common.FORCE_FLAG},
	Action: func(c *cli.Context) error {

		project.StartDevMode(c)

		return nil
	},
}
