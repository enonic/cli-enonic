package cms

import (
	"github.com/urfave/cli"
)

func All() []cli.Command {
	return []cli.Command{
		Reprocess,
	}
}
