package system

import (
	"fmt"
	"github.com/enonic/cli-enonic/internal/app/util"
	"os"
	"os/exec"
)

func Start(app string, args []string, detach bool) *exec.Cmd {

	cmd := prepareCmd(app, args)
	setCommandLineParams(cmd, app, args)

	if !detach {
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		cmd.Stdin = os.Stdin
		setStartAttachedParams(cmd)
	} else {
		setStartDetachedParams(cmd)
	}
	err := cmd.Start()

	util.Fatal(err, fmt.Sprintf("Could not start process: %s", app))
	return cmd
}

func GetDetachedProcName() string {
	return detachedProcName
}
