package cloud

import (
	"github.com/urfave/cli"
)

func All() []cli.Command {
	return []cli.Command{
		Login,
		Logout,
		App,
	}
}
