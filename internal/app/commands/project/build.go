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
	projectData := ensureProjectDataExists(c)
	runGradleTask(projectData, "build", fmt.Sprintf("Building using '%s'...", projectData.Sandbox))
}
