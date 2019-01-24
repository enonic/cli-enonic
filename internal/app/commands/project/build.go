package project

import (
	"github.com/urfave/cli"
	"fmt"
)

var Build = cli.Command{
	Name:  "build",
	Usage: "Build current project",
	Action: func(c *cli.Context) error {

		buildProject(c)

		return nil
	},
}

func buildProject(c *cli.Context) {
	projectData := ensureProjectDataExists(c, "A sandbox is required to build the project, do you want to create one?")
	runGradleTask(projectData, "build", fmt.Sprintf("Building using sandbox '%s'...", projectData.Sandbox))
}
