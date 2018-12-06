package remote

import (
	"github.com/urfave/cli"
)

func All() []cli.Command {
	return []cli.Command{
		Add,
		Remove,
		Set,
		List,
	}
}
