package sandbox

import (
	"github.com/urfave/cli"
	"github.com/enonic/xp-cli/internal/app/util"
	"strings"
	"fmt"
	"os"
	"os/signal"
)

var Start = cli.Command{
	Name:  "start",
	Usage: "Start the sandbox.",
	Action: func(c *cli.Context) error {
		if running := readRunningSandbox(); running != "" {
			fmt.Fprintf(os.Stderr, "Sandbox '%s' is currently running, stop it first!\n", running)
			os.Exit(1)
		}
		if !util.IsPortAvailable(8080) {
			fmt.Fprintln(os.Stderr, "Port 8080 is not available, stop the app using it first!")
			os.Exit(1)
		}

		name := ensureStartNameArg(c)
		data := readSandboxData(name)
		ensureDistroPresent(data.Distro)

		cmd := startDistro(data.Distro, name)
		writeRunningSandbox(name)
		listenForInterrupt(name)

		cmd.Wait()
		return nil
	},
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

func ensureStartNameArg(c *cli.Context) string {
	var name string
	if c.NArg() > 0 {
		name = c.Args().First()
	}
	existingBoxes := ListSandboxes()
	return util.PromptUntilTrue(name, func(val string, i byte) string {
		if len(strings.TrimSpace(val)) == 0 {
			if i == 0 {
				return "Enter the name of the sandbox: "
			} else {
				return "Name of the sandbox can not be empty: "
			}
		} else {
			for _, existingBox := range existingBoxes {
				if existingBox == val {
					return ""
				}
			}
			return fmt.Sprintf("Sandbox with the name '%s' not found: ", val)
		}
	})
}
