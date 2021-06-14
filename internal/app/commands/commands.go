package commands

import (
	"cli-enonic/internal/app/commands/app"
	"cli-enonic/internal/app/commands/cloud"
	"cli-enonic/internal/app/commands/cms"
	"cli-enonic/internal/app/commands/dump"
	"cli-enonic/internal/app/commands/export"
	"cli-enonic/internal/app/commands/project"
	"cli-enonic/internal/app/commands/repo"
	"cli-enonic/internal/app/commands/sandbox"
	"cli-enonic/internal/app/commands/snapshot"
	"cli-enonic/internal/app/commands/system"
	"cli-enonic/internal/app/commands/vacuum"
	"github.com/urfave/cli"
)

func All() []cli.Command {
	return []cli.Command{
		{
			Name:        "snapshot",
			Usage:       "Create and restore snapshots",
			HelpName:    "Snapshot",
			Subcommands: snapshot.All(),
		},
		{
			Name:        "dump",
			Usage:       "Dump and load complete repositories",
			HelpName:    "Dump",
			Subcommands: dump.All(),
		},
		export.Export,
		export.Import,
		{
			Name:        "app",
			Usage:       "Install, stop and start applications",
			HelpName:    "Application",
			Subcommands: app.All(),
		},
		{
			Name:        "repo",
			Usage:       "Tune and manage repositories",
			HelpName:    "Repo",
			Subcommands: repo.All(),
		},
		{
			Name:        "cms",
			Usage:       "CMS commands",
			HelpName:    "CMS",
			Subcommands: cms.All(),
		},
		{
			Name:        "system",
			Usage:       "System commands",
			Subcommands: system.All(),
			HelpName:    "System",
		},
		{
			Name:        "sandbox",
			Usage:       "Manage XP development instances",
			Subcommands: sandbox.All(),
			HelpName:    "Sandbox",
			Category:    "PROJECT COMMANDS",
		},
		{
			Name:        "project",
			Usage:       "Manage XP development projects",
			Subcommands: project.All(),
			HelpName:    "Project",
			Category:    "PROJECT COMMANDS",
		},
		{
			Name:        "cloud",
			Usage:       "Manage Enonic cloud",
			Subcommands: cloud.All(),
			HelpName:    "Cloud",
			Category:    "CLOUD COMMANDS",
		},
		system.Latest,
		vacuum.Vacuum,
		/*{
			Name:        "remote",
			Usage:       "Remote commands",
			Subcommands: remote.All(),
		},*/
	}
}
