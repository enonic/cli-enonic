package project

import (
	"github.com/urfave/cli"
	"fmt"
)

var Clean = cli.Command{
	Name:  "clean",
	Usage: "Clean current project",
	Action: func(c *cli.Context) error {

		if projectData := ensureProjectDataExists(c, ".", "A sandbox is required to clean the project, do you want to create one?"); projectData != nil {
			runGradleTask(projectData, "clean", fmt.Sprintf("Cleaning using sandbox '%s'...", projectData.Sandbox))
		}

		return nil
	},
}
