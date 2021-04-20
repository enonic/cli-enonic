package project

import (
	"cli-enonic/internal/app/commands/common"
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
	if projectData := ensureProjectDataExists(c, ".", "A sandbox is required for your project, create one?", true); projectData != nil {
		runGradleTask(projectData, fmt.Sprintf("Building in sandbox '%s'...", projectData.Sandbox), "build")
	}
}
