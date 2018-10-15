package commands

import (
	"github.com/urfave/cli"
	"enonic.com/xp-cli/commands/snapshot"
)

func All() []cli.Command {
	return []cli.Command{
		{
			Name:    "snapshot",
			Usage:   "Snapshot commands",
			Subcommands: snapshot.All(),
		},
	}
}
