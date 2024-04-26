package project

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/commands/sandbox"
	"fmt"
	"github.com/urfave/cli"
)

var Clean = cli.Command{
	Name:  "clean",
	Usage: "Clean current project",
	Flags: []cli.Flag{common.FORCE_FLAG},
	Action: func(c *cli.Context) error {
		if projectData, _ := ensureProjectDataExists(c, ".", "", "A sandbox is required to clean the project, "+
			"do you want to create one"); projectData != nil {
			var cleanMessage string
			if sandbox.Exists(projectData.Sandbox) {
				cleanMessage = fmt.Sprintf("Cleaning in sandbox '%s'...", projectData.Sandbox)
			} else {
				cleanMessage = "No sandbox found, cleaning without a sandbox..."
			}
			runGradleTask(projectData, cleanMessage, "clean")
		}

		return nil
	},
}
