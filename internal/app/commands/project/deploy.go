package project

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/commands/sandbox"
	"cli-enonic/internal/app/util"
	"fmt"
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
		common.FORCE_FLAG,
	},
	Action: func(c *cli.Context) error {
		force := common.IsForceMode(c)
		continuous := c.Bool("continuous")
		if projectData := ensureProjectDataExists(c, ".", "A sandbox is required to deploy the project, do you want to create one?", true); projectData != nil {
			sandboxExists := sandbox.Exists(projectData.Sandbox)
			tasks := []string{"deploy"}
			if continuous {
				tasks = append(tasks, "--continuous")
				// ask to run sandbox in detached mode before gradle deploy because it has continuous flag
				if sandboxExists {
					askToRunSandbox(c, projectData)
					fmt.Fprintln(os.Stderr, "")
				}
			}

			var deployMessage string
			if sandboxExists {
				deployMessage = fmt.Sprintf("Deploying to sandbox '%s'...", projectData.Sandbox)
			} else {
				deployMessage = "No sandbox found, deploying without a sandbox..."
			}
			runGradleTask(projectData, deployMessage, tasks...)
			fmt.Fprintln(os.Stderr, "")

			if sandboxExists {
				if !continuous {
					askToRunSandbox(c, projectData)
				} else if rData := common.ReadRuntimeData(); rData.Running != "" {
					// ask to stop sandbox running in detached mode
					sandbox.AskToStopSandbox(rData, force)
				}
			}
		}

		return nil
	},
}

func askToRunSandbox(c *cli.Context, projectData *common.ProjectData) {
	rData := common.ReadRuntimeData()
	processRunning := common.VerifyRuntimeData(&rData)
	force := common.IsForceMode(c)
	devMode := c.Bool("dev")
	debug := c.Bool("debug")
	continuous := c.Bool("continuous")

	if !processRunning {
		if force || util.PromptBool(fmt.Sprintf("Do you want to start sandbox '%s'?", projectData.Sandbox), true) {
			// detach in continuous mode to release terminal window
			sandbox.StartSandbox(c, sandbox.ReadSandboxData(projectData.Sandbox), continuous, devMode, debug, common.HTTP_PORT)
		}
	} else if rData.Running != projectData.Sandbox {
		// Ask to stop running box if it differs from project selected only
		if force || util.PromptBool(fmt.Sprintf("Do you want to stop running sandbox '%s' and start '%s' instead ?", rData.Running, projectData.Sandbox), true) {
			sandbox.StopSandbox(rData)
			// detach in continuous mode to release terminal window
			sandbox.StartSandbox(c, sandbox.ReadSandboxData(projectData.Sandbox), continuous, devMode, debug, common.HTTP_PORT)
		}
	} else {
		// Desired sandbox is already running, just give a heads up about  --dev and --debug params
		color.New(color.FgCyan).Fprintf(os.Stderr, "Sandbox '%s' is already running. --dev and --debug parameters ignored\n\n", projectData.Sandbox)
	}
}
