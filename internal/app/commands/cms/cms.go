package cms

import (
	"cli-enonic/internal/app/commands/cms/namespace"
	"cli-enonic/internal/app/commands/cms/schema"

	"github.com/urfave/cli"
)

var Namespace = cli.Command{
	Name:        "namespace",
	Usage:       "Namespace commands",
	Subcommands: namespace.All(),
}

var Schema = cli.Command{
	Name:        "schema",
	Usage:       "Schema commands",
	Subcommands: schema.All(),
}

func All() []cli.Command {
	return []cli.Command{
		Reprocess,
		Namespace,
		Schema,
	}
}
