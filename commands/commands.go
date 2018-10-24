package commands

import (
	"github.com/urfave/cli"
	"enonic.com/xp-cli/commands/snapshot"
	"enonic.com/xp-cli/commands/dump"
	"enonic.com/xp-cli/commands/export"
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
	}
}
