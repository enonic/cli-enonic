//+build !windows

package system

import (
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"syscall"
)

const detachedProcName = ""

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

func checkWriteAccess(path string) bool {
	if info, err := os.Stat(path); err == nil {
		mode := info.Mode()
		if mode&(1<<1) != 0 {
			// anyone has write access
			return true
		} else {
			stat := info.Sys().(*syscall.Stat_t)
			me, meErr := user.Current()
			if meErr != nil {
				return false
			}
			if mode&(1<<7) != 0 {
				// author has write access, check if current user is author
				authorId := strconv.FormatUint(uint64(stat.Uid), 10)
				if me.Uid == authorId {
					return true
				}
			} else if mode&(1<<4) != 0 {
				// members of file group has write access, check if current user belongs to it
				fileGroupId := int(stat.Gid)
				myGroupIds, groupsErr := me.GroupIds()
				if groupsErr != nil {
					return false
				}
				for myGroupId, _ := range myGroupIds {
					if myGroupId == fileGroupId {
						return true
					}
				}
			}
		}
	}
	return false
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
