//+build !windows

package system

import (
	"syscall"
	"os/exec"
)

func setStartDetachedParams(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Noctty:  true,
		Setpgid: true,
	}
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
