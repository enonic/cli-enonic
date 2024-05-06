//go:build windows
// +build windows

package system

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"
)

const detachedProcName = "cmd"

func prepareCmd(app string, args []string) *exec.Cmd {
	cmd := exec.Command(detachedProcName)
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	return cmd
}

/*
Description of the '@' trick to prevent quote stripping by cmd.exe
https://github.com/Microsoft/WSL/issues/2835#issuecomment-364475668
*/
func setCommandLineParams(cmd *exec.Cmd, app string, args []string) {
	cmd.SysProcAttr.CmdLine = fmt.Sprintf(` /c @ "%v" %v`, app, strings.Join(args, " "))
}

/*
https://docs.microsoft.com/en-us/windows/desktop/procthread/process-creation-flags
CREATE_NEW_PROCESS_GROUP = 0x00000200
*/
func setStartAttachedParams(cmd *exec.Cmd) {
	//cmd.SysProcAttr.CreationFlags = 0x00000200
}

/*
https://docs.microsoft.com/en-us/windows/desktop/procthread/process-creation-flags
CREATE_NEW_PROCESS_GROUP = 0x00000200
CREATE_NO_WINDOW = 0x08000000
*/
func setStartDetachedParams(cmd *exec.Cmd) {
	cmd.SysProcAttr.CreationFlags = 0x08000200
	cmd.SysProcAttr.HideWindow = true
}

func checkWriteAccess(path string) bool {
	return true
}

/*
 *	Taken from https://blog.csdn.net/fyxichen/article/details/51857864
 */

const (
	MAX_PATH           = 260
	TH32CS_SNAPPROCESS = 0x00000002
)

type ProcessInfo struct {
	Name string
	Pid  uint32
	PPid uint32
}

type PROCESSENTRY32 struct {
	DwSize              uint32
	CntUsage            uint32
	Th32ProcessID       uint32
	Th32DefaultHeapID   uintptr
	Th32ModuleID        uint32
	CntThreads          uint32
	Th32ParentProcessID uint32
	PcPriClassBase      int32
	DwFlags             uint32
	SzExeFile           [MAX_PATH]uint16
}

type HANDLE uintptr

var (
	modkernel32                  = syscall.NewLazyDLL("kernel32.dll")
	procCreateToolhelp32Snapshot = modkernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32First           = modkernel32.NewProc("Process32FirstW")
	procProcess32Next            = modkernel32.NewProc("Process32NextW")
	procCloseHandle              = modkernel32.NewProc("CloseHandle")
)

func Setpgid(pid, pgid int) error {
	return nil
}

func KillAll(pid int) error {
	pids := getppids(pid)
	Kill(pids)
	return nil
}

func Kill(pids []int) {
	for _, pid := range pids {
		pro, err := os.FindProcess(pid)
		if err != nil {
			continue
		}
		pro.Kill()
	}
}

func getppids(pid int) []int {
	infos, err := GetProcs()
	if err != nil {
		return []int{pid}
	}
	var pids = make([]int, 0, len(infos))
	var index int = 0
	pids = append(pids, pid)

	var length int = len(pids)
	for index < length {
		for _, info := range infos {
			if int(info.PPid) == pids[index] {
				pids = append(pids, int(info.Pid))
			}
		}
		index += 1
		length = len(pids)
	}
	return pids
}

func GetProcs() (procs []ProcessInfo, err error) {
	snap := createToolhelp32Snapshot(TH32CS_SNAPPROCESS, uint32(0))
	if snap == 0 {
		err = syscall.GetLastError()
		return
	}
	defer closeHandle(snap)
	var pe32 PROCESSENTRY32
	pe32.DwSize = uint32(unsafe.Sizeof(pe32))
	if process32First(snap, &pe32) == false {
		err = syscall.GetLastError()
		return
	}
	procs = append(procs, ProcessInfo{syscall.UTF16ToString(pe32.SzExeFile[:260]), pe32.Th32ProcessID, pe32.Th32ParentProcessID})
	for process32Next(snap, &pe32) {
		procs = append(procs, ProcessInfo{syscall.UTF16ToString(pe32.SzExeFile[:260]), pe32.Th32ProcessID, pe32.Th32ParentProcessID})
	}
	return
}

func createToolhelp32Snapshot(flags, processId uint32) HANDLE {
	ret, _, _ := procCreateToolhelp32Snapshot.Call(
		uintptr(flags),
		uintptr(processId))
	if ret <= 0 {
		return HANDLE(0)
	}
	return HANDLE(ret)
}

func process32First(snapshot HANDLE, pe *PROCESSENTRY32) bool {
	ret, _, _ := procProcess32First.Call(
		uintptr(snapshot),
		uintptr(unsafe.Pointer(pe)))
	return ret != 0
}

func process32Next(snapshot HANDLE, pe *PROCESSENTRY32) bool {
	ret, _, _ := procProcess32Next.Call(
		uintptr(snapshot),
		uintptr(unsafe.Pointer(pe)))
	return ret != 0
}

func closeHandle(object HANDLE) bool {
	ret, _, _ := procCloseHandle.Call(
		uintptr(object))
	return ret != 0
}
