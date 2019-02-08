package commands

import (
	"github.com/urfave/cli"
	"github.com/enonic/enonic-cli/internal/app/commands/sandbox"
	"github.com/enonic/enonic-cli/internal/app/commands/project"
	"github.com/enonic/enonic-cli/internal/app/commands/snapshot"
	"github.com/enonic/enonic-cli/internal/app/commands/dump"
	"github.com/enonic/enonic-cli/internal/app/commands/export"
	"github.com/enonic/enonic-cli/internal/app/commands/app"
	"github.com/enonic/enonic-cli/internal/app/commands/repo"
	"github.com/enonic/enonic-cli/internal/app/commands/cms"
	"github.com/enonic/enonic-cli/internal/app/commands/cluster"
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
		{
			Name:        "export",
			Usage:       "Export and load repository structures",
			HelpName:    "Export",
			Subcommands: export.All(),
		},
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
			Name:        "cluster",
			Usage:       "Cluster commands",
			HelpName:    "Cluster",
			Subcommands: cluster.All(),
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
		/*
				{
					Name:        "remote",
					Usage:       "Remote commands",
					Subcommands: remote.All(),
				},
				vacuum.Vacuum,
		*/
	}
}
