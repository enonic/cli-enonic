package project

import (
	"github.com/urfave/cli"
)

func All() []cli.Command {
	return []cli.Command{
		Create,
		Sandbox,
		Clean,
		Build,
		Deploy,
		Install,
	}
}
