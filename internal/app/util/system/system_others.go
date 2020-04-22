//+build !windows

package system

import (
	"os/exec"
	"syscall"
)

func prepareCmd(app string, args []string) *exec.Cmd {
	cmd := exec.Command(app, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	return cmd
}

func setCommandLineParams(cmd *exec.Cmd, app string, args []string) {
	// they have already been set in the prepareCmd
}

func setStartAttachedParams(cmd *exec.Cmd) {
	// cmd.SysProcAttr.Setpgid = true
}

func setStartDetachedParams(cmd *exec.Cmd) {
	cmd.SysProcAttr.Noctty = false
	cmd.SysProcAttr.Setpgid = true
}

/*
 *	Taken from https://blog.csdn.net/fyxichen/article/details/51857864
 */

func Setpgid(pid, pgid int) error {
	return syscall.Setpgid(pid, pgid)
}

func Getppids(pid int) ([]uint32, error) {
	return []uint32{}, nil
}

func Kill(pids []uint32) {
	for _, pid := range pids {
		syscall.Kill(int(pid), syscall.SIGTERM)
	}
}

func KillAll(pid int) error {
	return syscall.Kill(pid-(pid*2), syscall.SIGTERM)
}
