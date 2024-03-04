package sandbox

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"errors"
	"fmt"
	"github.com/urfave/cli"
	"os"
)

var Start = cli.Command{
	Name: "start",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "detach,d",
			Usage: "Run in the background even after console is closed",
		},
		cli.BoolFlag{
			Name:  "prod",
			Usage: "Run Enonic XP distribution in non-development mode",
		},
		cli.BoolFlag{
			Name:  "debug",
			Usage: "Run enonic XP server with debug enabled on port 5005",
		},
		cli.UintFlag{
			Name:  "http.port",
			Usage: "Set to the http port used by Enonic XP to check availability on startup",
			Value: common.HTTP_PORT,
		},
		common.FORCE_FLAG,
	},
	Usage:     "Start the sandbox.",
	ArgsUsage: "<name>",
	Action: func(c *cli.Context) error {

		sandbox := ReadSandboxFromProjectOrAsk(c, true)

		err, _ := StartSandbox(c, sandbox, c.Bool("detach"), !c.Bool("prod"), c.Bool("debug"), uint16(c.Uint("http.port")))
		util.Fatal(err, "")

		return nil
	},
}

func ReadSandboxFromProjectOrAsk(c *cli.Context, useArguments bool) *Sandbox {
	var sandbox *Sandbox
	var minDistroVersion string
	// use configured sandbox if we're in a project folder
	if !useArguments || (c.NArg() == 0 && common.HasProjectData(".")) {
		pData := common.ReadProjectData(".")
		minDistroVersion = common.ReadProjectDistroVersion(".")
		sandbox = ReadSandboxData(pData.Sandbox)
	}
	if sandbox == nil {
		var sandboxName string
		if useArguments && c.NArg() > 0 {
			sandboxName = c.Args().First()
		}
		sandbox, _ = EnsureSandboxExists(c, EnsureSandboxOptions{
			MinDistroVersion:   minDistroVersion,
			Name:               sandboxName,
			NoBoxMessage:       "No sandboxes found, do you want to create one",
			SelectBoxMessage:   "Select sandbox to start",
			ShowSuccessMessage: true,
			ShowCreateOption:   true,
		})
		if sandbox == nil {
			os.Exit(1)
		}
	}
	return sandbox
}

func StartSandbox(c *cli.Context, sandbox *Sandbox, detach, devMode, debug bool, httpPort uint16) (error, bool) {
	force := common.IsForceMode(c)
	rData := common.ReadRuntimeData()
	isSandboxRunning := common.VerifyRuntimeData(&rData)

	if sandbox.Distro == "" || sandbox.Name == "" {
		return errors.New("Sandbox distro and name must be set!"), false
	}

	if isSandboxRunning {
		if rData.Running == sandbox.Name && ((rData.Mode == common.MODE_DEV) == devMode) {
			return nil, true
		} else if !AskToStopSandbox(rData, force) {
			return errors.New(fmt.Sprintf("Sandbox '%s' is already running in %s mode", rData.Running, rData.Mode)), true
		}
	} else {
		ports := []uint16{httpPort, common.MGMT_PORT, common.INFO_PORT}
		var unavailablePorts []uint16
		for _, port := range ports {
			if !util.IsPortAvailable(port) {
				unavailablePorts = append(unavailablePorts, port)
			}
		}
		if len(unavailablePorts) > 0 {
			return errors.New(fmt.Sprintf("Port(s) %v are not available, stop the app(s) using them first!\n", unavailablePorts)), false
		}
	}

	EnsureDistroExists(sandbox.Distro)

	cmd := startDistro(sandbox.Distro, sandbox.Name, detach, devMode, debug)

	var pid int
	if !detach {
		// current process' PID
		pid = os.Getpid()
	} else {
		// current process will finish so use detached process' PID
		pid = cmd.Process.Pid
	}
	writeRunningSandbox(sandbox.Name, pid, devMode)

	if !detach {
		util.ListenForInterrupt(func() {
			fmt.Fprintln(os.Stderr)
			fmt.Fprintf(os.Stderr, "Got interrupt signal, stopping sandbox '%s'\n", sandbox.Name)
			fmt.Fprintln(os.Stderr)
			writeRunningSandbox("", 0, false)
		})
		cmd.Wait()
	} else {
		fmt.Fprintf(os.Stdout, "Started sandbox '%s' in detached mode.\n", sandbox.Name)
	}
	return nil, false
}

func writeRunningSandbox(name string, pid int, dev bool) {
	data := common.ReadRuntimeData()
	data.Running = name
	data.PID = pid
	if dev {
		data.Mode = common.MODE_DEV
	} else {
		data.Mode = common.MODE_DEFAULT
	}
	common.WriteRuntimeData(data)
}
