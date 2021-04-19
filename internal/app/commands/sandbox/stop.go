package sandbox

import (
	"fmt"
	"github.com/enonic/cli-enonic/internal/app/commands/common"
	"github.com/urfave/cli"
	"os"
)

var Stop = cli.Command{
	Name:  "stop",
	Usage: "Stop the sandbox started in detached mode.",
	Flags: []cli.Flag{common.FORCE_FLAG},
	Action: func(c *cli.Context) error {

		rData := common.ReadRuntimeData()
		if !common.VerifyRuntimeData(&rData) {
			fmt.Fprintln(os.Stderr, "No sandbox is currently running.")
			os.Exit(1)
		}
		StopSandbox(rData)

		return nil
	},
}

func StopSandbox(rData common.RuntimeData) {
	stopDistro(rData.PID)
	writeRunningSandbox("", 0)

	fmt.Fprintf(os.Stdout, "Sandbox '%s' stopped\n", rData.Running)
}
