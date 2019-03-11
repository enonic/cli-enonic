//+build !windows

package system

import (
	"os/exec"
	"syscall"
)

func setStartAttachedParams(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		//Setpgid: true,
	}
}

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
		syscall.Kill(int(pid), syscall.SIGQUIT)
	}
}

func KillAll(pid int) error {
	return syscall.Kill(pid-(pid*2), syscall.SIGQUIT)
}
