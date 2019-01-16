package sandbox

import (
	"github.com/urfave/cli"
	"fmt"
	"os"
	"os/signal"
	"github.com/enonic/xp-cli/internal/app/util"
)

var Start = cli.Command{
	Name:  "start",
	Usage: "Start the sandbox.",
	Action: func(c *cli.Context) error {

		ensurePortAvailable(8080)
		sandbox, _ := EnsureSandboxExists(c, "No sandboxes found, do you want to create one?", "Select sandbox to start:")
		EnsureDistroExists(sandbox.Distro)

		cmd := startDistro(sandbox.Distro, sandbox.Name)
		writeRunningSandbox(sandbox.Name)
		listenForInterrupt(sandbox.Name)

		cmd.Wait()
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
		writeRunningSandbox("")
	}()
}

func writeRunningSandbox(name string) {
	data := readSandboxesData()
	data.Running = name
	writeSandboxesData(data)
}

func readRunningSandbox() string {
	return readSandboxesData().Running
}
