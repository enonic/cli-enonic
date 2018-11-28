package project

import (
	"github.com/urfave/cli"
)

var Build = cli.Command{
	Name:  "build",
	Usage: "Build current project",
	Action: func(c *cli.Context) error {

		runGradleTask("build", "Building...")

		return nil
	},
}
