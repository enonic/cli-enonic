package project

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/commands/sandbox"
	"fmt"
	"github.com/urfave/cli"
)

var Test = cli.Command{
	Name:  "test",
	Usage: "Run tests in the current project",
	Flags: []cli.Flag{common.FORCE_FLAG},
	Action: func(c *cli.Context) error {
		if projectData, _ := ensureProjectDataExists(c, ".", "", "A sandbox is required to test the project, "+
			"do you want to create one"); projectData != nil {
			var cleanMessage string
			if sandbox.Exists(projectData.Sandbox) {
				cleanMessage = fmt.Sprintf("Testing in sandbox '%s'...", projectData.Sandbox)
			} else {
				cleanMessage = "No sandbox found, testing without a sandbox..."
			}
			runGradleTask(projectData, cleanMessage, "test")
		}

		return nil
	},
}
