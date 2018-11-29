package project

import (
	"github.com/urfave/cli"
	"fmt"
)

var Deploy = cli.Command{
	Name:  "deploy",
	Usage: "Deploy current project to a sandbox",
	Action: func(c *cli.Context) error {

		projectData := ensureProjectDataExists(c)
		runGradleTask(projectData, "deploy", fmt.Sprintf("Deploying to %s...", projectData.Sandbox))

		return nil
	},
}
