package commands

import (
	"github.com/urfave/cli"
	"enonic.com/xp-cli/commands/snapshot"
	"enonic.com/xp-cli/commands/dump"
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
	}
}
