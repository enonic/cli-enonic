package commands

import (
	"github.com/urfave/cli"
	"github.com/enonic/xp-cli/internal/app/commands/sandbox"
	"github.com/enonic/xp-cli/internal/app/commands/project"
	"github.com/enonic/xp-cli/internal/app/commands/snapshot"
	"github.com/enonic/xp-cli/internal/app/commands/dump"
	"github.com/enonic/xp-cli/internal/app/commands/export"
	"github.com/enonic/xp-cli/internal/app/commands/app"
	"github.com/enonic/xp-cli/internal/app/commands/repo"
	"github.com/enonic/xp-cli/internal/app/commands/cms"
	"github.com/enonic/xp-cli/internal/app/commands/cluster"
)

func All() []cli.Command {
	return []cli.Command{
		{
			Name:        "snapshot",
			Usage:       "Snapshot commands",
			HelpName:    "Snapshot",
			Subcommands: snapshot.All(),
		},
		{
			Name:        "dump",
			Usage:       "Dump commands",
			HelpName:    "Dump",
			Subcommands: dump.All(),
		},
		{
			Name:        "export",
			Usage:       "Export commands",
			HelpName:    "Export",
			Subcommands: export.All(),
		},
		{
			Name:        "app",
			Usage:       "Application commands",
			HelpName:    "Application",
			Subcommands: app.All(),
		},
		{
			Name:        "repo",
			Usage:       "Repository commands",
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
			Usage:       "Sandbox commands",
			Subcommands: sandbox.All(),
			HelpName:    "Sandbox",
			Category:    "PROJECT COMMANDS",
		},
		{
			Name:        "project",
			Usage:       "Project commands",
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
