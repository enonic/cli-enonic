//+build !windows

package system

import (
	"syscall"
	"os/exec"
	"os"
	"github.com/enonic/xp-cli/internal/app/util"
	"fmt"
)

func Start(app string, args []string, detach bool) *exec.Cmd {
	var err error
	cmd := exec.Command(app, args...)

	if !detach {
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		cmd.Stdin = os.Stdin
		err = cmd.Run()
	} else {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Noctty:  true,
			Setpgid: true,
		}
		err = cmd.Start()
	}

	util.Fatal(err, fmt.Sprintf("Could not start process: %s", app))
	return cmd
}

/*
 *	Taken from https://blog.csdn.net/fyxichen/article/details/51857864
 */

func SetPgid(pid, pgid int) error {
	return syscall.Setpgid(pid, pgid)
}

func GetPPids(pid int) ([]int, error) {
	return []int{}, nil
}

func Kill(pids []uint32) {
	for _, pid := range pids {
		syscall.Kill(int(pid), syscall.SIGKILL)
	}
}

func KillAll(pid int) error {
	return syscall.Kill(pid-(pid*2), syscall.SIGKILL)
}
