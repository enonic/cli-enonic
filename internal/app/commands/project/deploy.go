package project

import (
	"fmt"
	"github.com/enonic/enonic-cli/internal/app/commands/sandbox"
	"github.com/enonic/enonic-cli/internal/app/util"
	"github.com/urfave/cli"
)

var Deploy = cli.Command{
	Name:  "deploy",
	Usage: "Deploy current project to a sandbox",
	Action: func(c *cli.Context) error {

		if projectData := ensureProjectDataExists(c, ".", "A sandbox is required to deploy the project, do you want to create one?"); projectData != nil {
			runGradleTask(projectData, "deploy", fmt.Sprintf("Deploying to sandbox '%s'...", projectData.Sandbox))

			sData := sandbox.ReadSandboxesData();
			if sData.PID == 0 {
				if util.PromptBool(fmt.Sprintf("\nDo you want to start sandbox '%s'?", projectData.Sandbox), true) {
					sandbox.StartSandbox(sandbox.ReadSandboxData(projectData.Sandbox), false)
				}
			} else if sData.Running != projectData.Sandbox {
				if util.PromptBool(fmt.Sprintf("Do you want to stop running sandbox '%s' and start '%s' instead ?", sData.Running, projectData.Sandbox), true) {
					sandbox.StopSandbox(sData)
					sandbox.StartSandbox(sandbox.ReadSandboxData(projectData.Sandbox), false)
				}
			}
		}

		return nil
	},
}
