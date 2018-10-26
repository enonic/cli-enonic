package repo

import (
	"github.com/urfave/cli"
)

func All() []cli.Command {
	return []cli.Command{
		Reindex,
		ReadOnly,
	}
}
