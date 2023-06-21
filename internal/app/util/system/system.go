package system

import (
	"cli-enonic/internal/app/util"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func Run(command string, args, env []string) {
	cmd := exec.Command(command, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Env = env

	if err := cmd.Run(); err != nil {
		os.Stderr.WriteString(fmt.Sprintf("\n%s\n", err.Error()))
		if exitError, ok := err.(*exec.ExitError); ok {
			waitStatus := exitError.Sys().(syscall.WaitStatus)
			os.Exit(waitStatus.ExitStatus())
		} else {
			os.Exit(1)
		}
	}
}

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

func HasWriteAccess(path string) bool {
	return checkWriteAccess(path)
}
