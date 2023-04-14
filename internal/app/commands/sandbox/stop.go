package sandbox

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"fmt"
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
	pId := rData.PID
	stopDistro(pId)
	writeRunningSandbox("", 0)

	common.StartSpinner(fmt.Sprintf("Stopping sandbox '%s'", rData.Running))
	if err := util.WaitUntilProcessStopped(pId, 30); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
	common.StopSpinner()
	fmt.Fprintln(os.Stderr, "Done")
}
