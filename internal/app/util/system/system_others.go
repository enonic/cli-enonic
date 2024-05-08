//go:build !windows

package system

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"
)

const detachedProcName = "java"

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

func Kill(pids []int) {
	for _, pid := range pids {
		if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
			fmt.Printf("Failed to kill process %d: %s\n", pid, err)
			// Continue trying to kill other processes
		}
	}
}

func KillAll(pid int) error {
	pids, err := findChildPIDs(pid)
	if err != nil {
		return err
	}
	// Append the parent PID at the end to ensure it gets killed last
	pids = append(pids, pid)

	Kill(pids)

	return nil
}

func findChildPIDs(parentPID int) ([]int, error) {
	var pids []int
	var out bytes.Buffer
	// Using 'ps' to get child processes of a given PID
	cmd := exec.Command("pgrep", "-P", strconv.Itoa(parentPID))
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		errCode := (err.(*exec.ExitError)).ExitCode()
		if errCode == 1 {
			// No child processes found
			return pids, nil
		} else if errCode == 2 {
			// Syntax error
			return nil, err
		}
	}
	// Parse the output to get the child PIDs
	for _, pidStr := range strings.Split(out.String(), "\n") {
		pidStr = strings.TrimSpace(pidStr)
		if pidStr != "" {
			pid, err := strconv.Atoi(pidStr)
			if err != nil {
				continue // skip individual errors
			}
			pids = append(pids, pid)
			// Recursively find children of this child
			childPIDs, err := findChildPIDs(pid)
			if err == nil {
				pids = append(pids, childPIDs...)
			}
		}
	}
	return pids, nil
}
