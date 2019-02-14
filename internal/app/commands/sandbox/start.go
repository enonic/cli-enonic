package sandbox

import (
	"github.com/urfave/cli"
	"fmt"
	"os"
	"os/signal"
	"github.com/enonic/enonic-cli/internal/app/util"
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

		ensurePortAvailable(8080)
		sandbox, _ := EnsureSandboxExists(c, "No sandboxes found, create one?", "Select sandbox to start:", true)
		if sandbox == nil {
			os.Exit(0)
		}
		EnsureDistroExists(sandbox.Distro)

		StartSandbox(sandbox, c.Bool("detach"))

		return nil
	},
}

func StartSandbox(sandbox *Sandbox, detach bool) {
	cmd := startDistro(sandbox.Distro, sandbox.Name, detach)

	writeRunningSandbox(sandbox.Name, cmd.Process.Pid)
	listenForInterrupt(sandbox.Name)

	if !detach {
		listenForInterrupt(sandbox.Name)
		cmd.Wait()
	} else {
		fmt.Fprintf(os.Stderr, "Started sandbox '%s' in detached mode.", sandbox.Name)
	}
}

func ensurePortAvailable(port uint16) {
	sData := readSandboxesData()
	if sData.Running != "" && sData.PID != 0 {
		if util.YesNoPrompt(fmt.Sprintf("Sandbox '%s' is running, do you want to stop it?", sData.Running)) {
			StopSandbox(sData)
		} else {
			os.Exit(0)
		}
	} else if !util.IsPortAvailable(port) {
		fmt.Fprintln(os.Stderr, "Port 8080 is not available, stop the app using it first!")
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
	}()
}

func writeRunningSandbox(name string, pid int) {
	data := readSandboxesData()
	data.Running = name
	data.PID = pid
	writeSandboxesData(data)
}
