package sandbox

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"fmt"
	"github.com/urfave/cli"
	"os"
	"os/signal"
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
			Value: 8080,
		},
	},
	Usage: "Start the sandbox.",
	Action: func(c *cli.Context) error {

		rData := common.ReadRuntimeData()
		isSandboxRunning := common.VerifyRuntimeData(&rData)

		port := uint16(c.Uint("http.port"))
		if isSandboxRunning {
			if rData.Running == c.Args().First() {
				fmt.Fprintf(os.Stderr, "Sandbox '%s' is already running", rData.Running)
				os.Exit(1)
			} else {
				AskToStopSandbox(rData)
			}
		} else if !util.IsPortAvailable(port) {
			fmt.Fprintf(os.Stderr, "Port %d is not available, stop the app using it first!\n", port)
			os.Exit(1)
		}

		var sandbox *Sandbox
		var minDistroVersion string
		// use configured sandbox if we're in a project folder
		if c.NArg() == 0 && common.HasProjectData(".") {
			pData := common.ReadProjectData(".")
			minDistroVersion = common.ReadProjectDistroVersion(".")
			sandbox = ReadSandboxData(pData.Sandbox)
		}
		if sandbox == nil {
			sandbox, _ = EnsureSandboxExists(c, minDistroVersion, "No sandboxes found, create one?", "Select sandbox to start:", true, true)
			if sandbox == nil {
				os.Exit(1)
			}
		}

		StartSandbox(sandbox, c.Bool("detach"), c.Bool("dev"), c.Bool("debug"))

		return nil
	},
}

func StartSandbox(sandbox *Sandbox, detach, devMode, debug bool) {
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
		listenForInterrupt(sandbox.Name)
		cmd.Wait()
	} else {
		fmt.Fprintf(os.Stdout, "Started sandbox '%s' in detached mode.", sandbox.Name)
	}
}

func AskToStopSandbox(rData common.RuntimeData) {
	if util.PromptBool(fmt.Sprintf("Sandbox '%s' is running, do you want to stop it?", rData.Running), true) {
		StopSandbox(rData)
	} else {
		os.Exit(1)
	}
}

func listenForInterrupt(name string) {
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt)

	go func() {
		<-interruptChan
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "Got interrupt signal, stopping sandbox '%s'\n", name)
		fmt.Fprintln(os.Stderr)
		writeRunningSandbox("", 0)
		signal.Stop(interruptChan)
	}()
}

func writeRunningSandbox(name string, pid int) {
	data := common.ReadRuntimeData()
	data.Running = name
	data.PID = pid
	common.WriteRuntimeData(data)
}
