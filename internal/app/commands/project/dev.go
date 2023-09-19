package project

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/commands/sandbox"
	"cli-enonic/internal/app/util"
	"fmt"
	"github.com/urfave/cli"
	"os"
)

var Dev = cli.Command{
	Name:  "dev",
	Usage: "Start current project in dev mode",
	Flags: []cli.Flag{common.FORCE_FLAG},
	Action: func(c *cli.Context) error {

		if projectData := ensureProjectDataExists(c, ".", "", "A sandbox is required to run the project in dev mode, do you want to create one"); projectData != nil {

			sbox := sandbox.ReadSandboxFromProjectOrAsk(c, false)

			sandbox.StartSandbox(c, sbox, true, true, true, common.HTTP_PORT)

			devMessage := fmt.Sprintln("\nRunning project in dev mode...")
			util.ListenForInterrupt(func() {
				fmt.Fprintln(os.Stderr)
				fmt.Fprintln(os.Stderr, "Stopping project dev mode...")
				fmt.Fprintln(os.Stderr)
				sandbox.StopSandbox(common.ReadRuntimeData())
			})

			runGradleTask(projectData, devMessage, "dev")
		}

		return nil
	},
}
