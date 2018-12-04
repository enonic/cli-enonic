package sandbox

import (
	"github.com/urfave/cli"
	"fmt"
	"os"
	"os/signal"
	"github.com/enonic/xp-cli/internal/app/util"
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
		sandbox := EnsureSandboxNameExists(c, "Select sandbox to start:")
		ensureDistroPresent(sandbox.Distro)
		detach := c.Bool("detach")

		cmd := startDistro(sandbox.Distro, sandbox.Name, detach)
		writeRunningSandbox(sandbox.Name, cmd.Process.Pid)

		if !detach {
			listenForInterrupt(sandbox.Name)
			cmd.Wait()
		} else {
			fmt.Fprintf(os.Stderr, "Started sandbox '%s' in detached mode.", sandbox.Name)
		}
		return nil
	},
}

func ensurePortAvailable(port uint16) {
	if running := readRunningSandbox(); running != "" {
		fmt.Fprintf(os.Stderr, "Sandbox '%s' is currently running, stop it first!\n", running)
		os.Exit(1)
	}
	if !util.IsPortAvailable(port) {
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

func readRunningSandbox() string {
	return readSandboxesData().Running
}
