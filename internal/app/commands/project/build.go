package project

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/commands/sandbox"
	"fmt"
	"github.com/urfave/cli"
)

var Build = cli.Command{
	Name:  "build",
	Usage: "Build current project",
	Flags: []cli.Flag{common.FORCE_FLAG},
	Action: func(c *cli.Context) error {

		buildProject(c)

		return nil
	},
}

func buildProject(c *cli.Context) {
	if projectData, _ := ensureProjectDataExists(c, ".", "", "A sandbox is required for your project, create one"); projectData != nil {
		var buildMessage string
		if sandbox.Exists(projectData.Sandbox) {
			buildMessage = fmt.Sprintf("Building in sandbox '%s'...", projectData.Sandbox)
		} else {
			buildMessage = "No sandbox found, building without a sandbox..."
		}
		runGradleTask(projectData, buildMessage, "build")
	}
}
