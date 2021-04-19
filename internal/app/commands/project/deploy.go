package project

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/commands/sandbox"
	"cli-enonic/internal/app/util"
	"fmt"
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
		cli.BoolFlag{
			Name:  "debug",
			Usage: "Run enonic XP server with debug enabled on port 5005",
		},
	},
	Action: func(c *cli.Context) error {

		devMode := c.Bool("dev")
		debug := c.Bool("debug")
		if projectData := ensureProjectDataExists(c, ".", "A sandbox is required to deploy the project, do you want to create one?"); projectData != nil {
			runGradleTask(projectData, fmt.Sprintf("Deploying to sandbox '%s'...", projectData.Sandbox), "deploy")

			rData := common.ReadRuntimeData()
			processRunning := common.VerifyRuntimeData(&rData)

			if !processRunning {
				if util.PromptBool(fmt.Sprintf("\nDo you want to start sandbox '%s'?", projectData.Sandbox), true) {
					sandbox.StartSandbox(sandbox.ReadSandboxData(projectData.Sandbox), false, devMode, debug)
				}
			} else if rData.Running != projectData.Sandbox {
				// Ask to stop running box if it differs from project selected only
				if util.PromptBool(fmt.Sprintf("Do you want to stop running sandbox '%s' and start '%s' instead ?", rData.Running, projectData.Sandbox), true) {
					sandbox.StopSandbox(rData)
					sandbox.StartSandbox(sandbox.ReadSandboxData(projectData.Sandbox), false, devMode, debug)
				}
			}
		}

		return nil
	},
}
