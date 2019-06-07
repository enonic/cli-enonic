package sandbox

import (
	"fmt"
	"github.com/enonic/cli-enonic/internal/app/commands/common"
	"github.com/enonic/cli-enonic/internal/app/util"
	"github.com/mitchellh/go-ps"
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
	},
	Usage: "Start the sandbox.",
	Action: func(c *cli.Context) error {

		rData := common.ReadRuntimeData()
		isSandboxRunning := false

		if rData.Running != "" && rData.PID != 0 {
			proc, _ := ps.FindProcess(rData.PID)

			// make sure that process is still alive and has the same name
			if proc != nil && proc.Executable() == "enonic" {
				isSandboxRunning = true
			} else {
				writeRunningSandbox("", 0)
			}
		}

		if isSandboxRunning {
			if rData.Running == c.Args().First() {
				fmt.Fprintf(os.Stderr, "Sandbox '%s' is already running", rData.Running)
				os.Exit(0)
			} else {
				AskToStopSandbox(rData)
			}
		} else if !util.IsPortAvailable(8080) {
			fmt.Fprintln(os.Stderr, "Port 8080 is not available, stop the app using it first!")
			os.Exit(0)
		}

		var sandbox *Sandbox
		// use configured sandbox if we're in a project folder
		if c.NArg() == 0 && common.HasProjectData(".") {
			pData := common.ReadProjectData(".")
			sandbox = ReadSandboxData(pData.Sandbox)
		}
		if sandbox == nil {
			sandbox, _ = EnsureSandboxExists(c, "No sandboxes found, create one?", "Select sandbox to start:", true, true)
			if sandbox == nil {
				os.Exit(0)
			}
		}

		StartSandbox(sandbox, c.Bool("detach"))

		return nil
	},
}

func StartSandbox(sandbox *Sandbox, detach bool) {
	EnsureDistroExists(sandbox.Distro)

	cmd := startDistro(sandbox.Distro, sandbox.Name, detach)

	pid := os.Getpid()
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
		os.Exit(0)
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
