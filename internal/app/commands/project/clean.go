package project

import (
	"github.com/urfave/cli"
	"fmt"
)

var Clean = cli.Command{
	Name:  "clean",
	Usage: "Clean current project",
	Action: func(c *cli.Context) error {

		projectData := ensureProjectDataExists(c)
		runGradleTask(projectData, "clean", fmt.Sprintf("Cleaning using '%s'...", projectData.Sandbox))

		return nil
	},
}
