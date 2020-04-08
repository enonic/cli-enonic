package cloud

import (
	project "github.com/enonic/cli-enonic/internal/app/commands/cloud/project"
	"github.com/urfave/cli"
)

var Project = cli.Command{
	Name:    "project",
	Usage:   "Manage projects in Enonic Cloud",
	Aliases: []string{},
	Subcommands: []cli.Command{
		project.ProjectDeploy,
	},
}

func All() []cli.Command {
	return []cli.Command{
		Login,
		Logout,
		Project,
	}
}
