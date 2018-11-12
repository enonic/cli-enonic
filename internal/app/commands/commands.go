package commands

import (
	"github.com/urfave/cli"
	"github.com/enonic/xp-cli/internal/app/commands/snapshot"
	"github.com/enonic/xp-cli/internal/app/commands/dump"
	"github.com/enonic/xp-cli/internal/app/commands/export"
	"github.com/enonic/xp-cli/internal/app/commands/app"
	"github.com/enonic/xp-cli/internal/app/commands/repo"
	"github.com/enonic/xp-cli/internal/app/commands/cms"
	"github.com/enonic/xp-cli/internal/app/commands/cluster"
	"github.com/enonic/xp-cli/internal/app/commands/vacuum"
	"github.com/enonic/xp-cli/internal/app/commands/sandbox"
)

func All() []cli.Command {
	return []cli.Command{
		{
			Name:        "snapshot",
			Usage:       "Snapshot commands",
			Subcommands: snapshot.All(),
		},
		{
			Name:        "dump",
			Usage:       "Dump commands",
			Subcommands: dump.All(),
		},
		{
			Name:        "export",
			Usage:       "Export commands",
			Subcommands: export.All(),
		},
		{
			Name:        "app",
			Usage:       "Application commands",
			Subcommands: app.All(),
		},
		{
			Name:        "repo",
			Usage:       "Repository commands",
			Subcommands: repo.All(),
		},
		{
			Name:        "cms",
			Usage:       "CMS commands",
			Subcommands: cms.All(),
		},
		{
			Name:        "cluster",
			Usage:       "Cluster commands",
			Subcommands: cluster.All(),
		},
		{
			Name:        "sandbox",
			Usage:       "Sandbox commands",
			Subcommands: sandbox.All(),
		},
		vacuum.Vacuum,
	}
}
