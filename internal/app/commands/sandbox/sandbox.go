package sandbox

import (
	"github.com/urfave/cli"
)

func All() []cli.Command {
	return []cli.Command{
		List,
		Start,
		New,
		Delete,
		Version,
	}
}
