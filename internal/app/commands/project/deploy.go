package project

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/commands/sandbox"
	"fmt"
	"github.com/urfave/cli"
	"os"
)

var Deploy = cli.Command{
	Name:      "deploy",
	Usage:     "Deploy current project to a sandbox",
	ArgsUsage: "<sandbox name>",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:   "dev",
			Usage:  "Run Enonic XP distribution in development mode",
			Hidden: true,
		},
		cli.BoolFlag{
			Name:  "prod",
			Usage: "Run Enonic XP distribution in non-development mode",
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
		var sandboxName string
		if c.NArg() > 0 {
			sandboxName = c.Args().First()
		}
		if projectData, _ := ensureProjectDataExists(c, ".", sandboxName, "A sandbox is required to deploy the project, "+
			"do you want to create one"); projectData != nil {
			sandboxExists := sandbox.Exists(projectData.Sandbox)
			tasks := []string{"deploy"}
			if continuous {
				tasks = append(tasks, "--continuous")
				// ask to run sandbox in detached mode before gradle deploy because it has continuous flag
				if sandboxExists {
					sandbox.AskToStartSandbox(c, projectData.Sandbox)
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
					sandbox.AskToStartSandbox(c, projectData.Sandbox)
				} else if rData := common.ReadRuntimeData(); rData.Running != "" {
					// ask to stop sandbox running in detached mode
					if !sandbox.AskToStopSandbox(rData, force) {
						os.Exit(1)
					}
				}
			}
		}

		return nil
	},
}
