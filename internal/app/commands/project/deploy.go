package project

import (
	"github.com/urfave/cli"
	"fmt"
	"github.com/enonic/enonic-cli/internal/app/util"
	"github.com/enonic/enonic-cli/internal/app/commands/sandbox"
)

var Deploy = cli.Command{
	Name:  "deploy",
	Usage: "Deploy current project to a sandbox",
	Action: func(c *cli.Context) error {

		if projectData := ensureProjectDataExists(c, ".", "A sandbox is required to deploy the project, do you want to create one?"); projectData != nil {
			runGradleTask(projectData, "deploy", fmt.Sprintf("Deploying to sandbox '%s'...", projectData.Sandbox))

			if util.IsPortAvailable(8080) && util.YesNoPrompt(fmt.Sprintf("\nDo you want to start sandbox '%s'?", projectData.Sandbox)) {
				sandbox.StartSandbox(sandbox.ReadSandboxData(projectData.Sandbox))
			}
		}

		return nil
	},
}
