package sandbox

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
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
			Name:  "dev",
			Usage: "Run enonic XP distribution in development mode",
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

		StartSandbox(c, sandbox, c.Bool("detach"), c.Bool("dev"), c.Bool("debug"), uint16(c.Uint("http.port")))

		return nil
	},
}

func ReadSandboxFromProjectOrAsk(c *cli.Context, useArguments bool) *Sandbox {
	var sandbox *Sandbox
	var minDistroVersion string
	// use configured sandbox if we're in a project folder
	if c.NArg() == 0 && common.HasProjectData(".") {
		pData := common.ReadProjectData(".")
		minDistroVersion = common.ReadProjectDistroVersion(".")
		sandbox = ReadSandboxData(pData.Sandbox)
	}
	if sandbox == nil {
		var sandboxName string
		if useArguments && c.NArg() > 0 {
			sandboxName = c.Args().First()
		}
		sandbox, _ = EnsureSandboxExists(c, minDistroVersion, sandboxName, "No sandboxes found, create one", "Select sandbox to start", true, true)
		if sandbox == nil {
			os.Exit(1)
		}
	}
	return sandbox
}

func StartSandbox(c *cli.Context, sandbox *Sandbox, detach, devMode, debug bool, httpPort uint16) {
	force := common.IsForceMode(c)
	rData := common.ReadRuntimeData()
	isSandboxRunning := common.VerifyRuntimeData(&rData)

	if isSandboxRunning {
		if rData.Running == sandbox.Name {
			fmt.Fprintf(os.Stderr, "Sandbox '%s' is already running", rData.Running)
			os.Exit(1)
		} else {
			AskToStopSandbox(rData, force)
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
			fmt.Fprintf(os.Stderr, "Port(s) %v are not available, stop the app(s) using them first!\n", unavailablePorts)
			os.Exit(1)
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
	writeRunningSandbox(sandbox.Name, pid)

	if !detach {
		util.ListenForInterrupt(func() {
			fmt.Fprintln(os.Stderr)
			fmt.Fprintf(os.Stderr, "Got interrupt signal, stopping sandbox '%s'\n", sandbox.Name)
			fmt.Fprintln(os.Stderr)
			writeRunningSandbox("", 0)
		})
		cmd.Wait()
	} else {
		fmt.Fprintf(os.Stdout, "Started sandbox '%s' in detached mode.\n", sandbox.Name)
	}
}

func AskToStopSandbox(rData common.RuntimeData, force bool) {
	if force || util.PromptBool(fmt.Sprintf("Sandbox '%s' is running, do you want to stop it", rData.Running), true) {
		StopSandbox(rData)
	} else {
		os.Exit(1)
	}
}

func writeRunningSandbox(name string, pid int) {
	data := common.ReadRuntimeData()
	data.Running = name
	data.PID = pid
	common.WriteRuntimeData(data)
}
