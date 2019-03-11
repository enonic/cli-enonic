package sandbox

import (
	"fmt"
	"github.com/urfave/cli"
	"os"
)

var Stop = cli.Command{
	Name:  "stop",
	Usage: "Stop the sandbox started in detached mode.",
	Action: func(c *cli.Context) error {

		sData := ReadSandboxesData()
		if sData.Running == "" || sData.PID == 0 {
			fmt.Fprintln(os.Stderr, "No sandbox is currently running.")
			os.Exit(0)
		}
		StopSandbox(sData)

		return nil
	},
}

func StopSandbox(sData SandboxesData) {
	stopDistro(sData.PID)
	writeRunningSandbox("", 0)

	fmt.Fprintf(os.Stderr, "Sandbox '%s' stopped\n", sData.Running)
}
