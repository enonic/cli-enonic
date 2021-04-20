package project

import (
	"cli-enonic/internal/app/commands/common"
	"fmt"
	"github.com/urfave/cli"
)

var Clean = cli.Command{
	Name:  "clean",
	Usage: "Clean current project",
	Flags: []cli.Flag{common.FORCE_FLAG},
	Action: func(c *cli.Context) error {

		if projectData := ensureProjectDataExists(c, ".", "A sandbox is required to clean the project, do you want to create one?", true); projectData != nil {
			runGradleTask(projectData, fmt.Sprintf("Cleaning in sandbox '%s'...", projectData.Sandbox), "clean")
		}

		return nil
	},
}
