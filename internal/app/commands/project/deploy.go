package project

import (
	"fmt"
	"github.com/enonic/cli-enonic/internal/app/commands/common"
	"github.com/enonic/cli-enonic/internal/app/commands/sandbox"
	"github.com/enonic/cli-enonic/internal/app/util"
	"github.com/fatih/color"
	"github.com/urfave/cli"
	"os"
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
		cli.BoolFlag{
			Name:  "continuous, c",
			Usage: "Watch changes and deploy project continuously",
		},
	},
	Action: func(c *cli.Context) error {
		continuous := c.Bool("continuous")
		devMode := c.Bool("dev")
		debug := c.Bool("debug")
		if projectData := ensureProjectDataExists(c, ".", "A sandbox is required to deploy the project, do you want to create one?"); projectData != nil {
			tasks := []string{"deploy"}
			if continuous {
				tasks = append(tasks, "--continuous")
				// ask to run sandbox in detached mode before gradle deploy because it has continuous flag
				askToRunSandbox(projectData, devMode, debug, continuous)
				fmt.Fprintln(os.Stderr, "")
			}

			runGradleTask(projectData, fmt.Sprintf("Deploying to sandbox '%s'...", projectData.Sandbox), tasks...)
			fmt.Fprintln(os.Stderr, "")

			if !continuous {
				askToRunSandbox(projectData, devMode, debug, continuous)
			} else {
				// ask to stop sandbox running in detached mode
				sandbox.AskToStopSandbox(common.ReadRuntimeData())
			}
		}

		return nil
	},
}

func askToRunSandbox(projectData *common.ProjectData, devMode, debug, continuous bool) {
	rData := common.ReadRuntimeData()
	processRunning := common.VerifyRuntimeData(&rData)

	if !processRunning {
		if util.PromptBool(fmt.Sprintf("Do you want to start sandbox '%s'?", projectData.Sandbox), true) {
			sandbox.StartSandbox(sandbox.ReadSandboxData(projectData.Sandbox), continuous, devMode, debug)
		}
	} else if rData.Running != projectData.Sandbox {
		// Ask to stop running box if it differs from project selected only
		if util.PromptBool(fmt.Sprintf("Do you want to stop running sandbox '%s' and start '%s' instead ?", rData.Running, projectData.Sandbox), true) {
			sandbox.StopSandbox(rData)
			sandbox.StartSandbox(sandbox.ReadSandboxData(projectData.Sandbox), continuous, devMode, debug)
		}
	} else {
		// Desired sandbox is already running, just give a heads up about  --dev and --debug params
		color.New(color.FgCyan).Fprintf(os.Stderr, "Sandbox '%s' is already running. --dev and --debug parameters ignored\n\n", projectData.Sandbox)
	}
}
