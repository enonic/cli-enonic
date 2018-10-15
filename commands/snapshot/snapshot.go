package snapshot

import (
	"github.com/urfave/cli"
)

func All() []cli.Command{
	return []cli.Command{
		List,
		New,
		Restore,
		Delete,
	}
}
