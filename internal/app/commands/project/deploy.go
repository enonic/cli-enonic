package project

import (
	"fmt"
	"github.com/enonic/cli-enonic/internal/app/commands/common"
	"github.com/enonic/cli-enonic/internal/app/commands/sandbox"
	"github.com/enonic/cli-enonic/internal/app/util"
	"github.com/urfave/cli"
)

var Deploy = cli.Command{
	Name:  "deploy",
	Usage: "Deploy current project to a sandbox",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "dev",
			Usage: "Run enonic XP distribution in development mode",
		},
	},
	Action: func(c *cli.Context) error {

		devMode := c.Bool("dev")
		if projectData := ensureProjectDataExists(c, ".", "A sandbox is required to deploy the project, do you want to create one?"); projectData != nil {
			runGradleTask(projectData, fmt.Sprintf("Deploying to sandbox '%s'...", projectData.Sandbox), "deploy")

			rData := common.ReadRuntimeData()
			processRunning := common.VerifyRuntimeData(&rData)

			if !processRunning {
				if util.PromptBool(fmt.Sprintf("\nDo you want to start sandbox '%s'?", projectData.Sandbox), true) {
					sandbox.StartSandbox(sandbox.ReadSandboxData(projectData.Sandbox), false, devMode)
				}
			} else {
				if util.PromptBool(fmt.Sprintf("Do you want to stop running sandbox '%s' and start '%s' instead ?", rData.Running, projectData.Sandbox), true) {
					sandbox.StopSandbox(rData)
					sandbox.StartSandbox(sandbox.ReadSandboxData(projectData.Sandbox), false, devMode)
				}
			}
		}

		return nil
	},
}
