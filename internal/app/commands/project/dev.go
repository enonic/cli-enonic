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

		StartDevMode(c)

		return nil
	},
}

func StartDevMode(c *cli.Context) {
	if projectData := ensureProjectDataExists(c, ".", "", "A sandbox is required to run the project in dev mode, do you want to create one"); projectData != nil {

		sbox := sandbox.ReadSandboxFromProjectOrAsk(c, false)

		err := sandbox.StartSandbox(c, sbox, true, true, true, common.HTTP_PORT)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Restart sandbox '%s' in dev mode or stop it before running dev command\n", sbox.Name)
			os.Exit(1)
		}

		devMessage := fmt.Sprintln("\nRunning project in dev mode...")
		util.ListenForInterrupt(func() {
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, "Stopping project dev mode...")
			fmt.Fprintln(os.Stderr)
			sandbox.StopSandbox(common.ReadRuntimeData())
		})

		runGradleTask(projectData, devMessage, "dev")
	}
}
