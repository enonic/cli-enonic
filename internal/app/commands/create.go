package commands

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/commands/project"
	"github.com/urfave/cli"
)

var Create = cli.Command{
	Name:      "create",
	Usage:     "Create a new Enonic project",
	ArgsUsage: "<project name>",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "repository, repo, r",
			Usage: "Repository path. Format: <enonic repo> or <organisation>/<repo> or <full repo url>",
		},
		cli.StringFlag{
			Name:  "sandbox, sb, s",
			Usage: "Sandbox name",
		},
		common.FORCE_FLAG,
	},
	Action: func(c *cli.Context) error {

		project.ProjectCreateWizard(c, true)

		return nil
	},
}
