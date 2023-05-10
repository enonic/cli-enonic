package create

import (
	"cli-enonic/internal/app/commands/project"
	"github.com/urfave/cli"
)

var Create = cli.Command{
	Name:      "create",
	Usage:     "Create a new Enonic project",
	ArgsUsage: "<project name>",
	Action: func(c *cli.Context) error {

		project.ProjectCreateWizard(c)

		return nil
	},
}
