package sandbox

import (
	"github.com/urfave/cli"
	"fmt"
	"github.com/enonic/enonic-cli/internal/app/util"
	"os"
)

var Delete = cli.Command{
	Name:    "delete",
	Usage:   "Delete a sandbox",
	Aliases: []string{"del", "rm"},
	Action: func(c *cli.Context) error {
		sandbox, _ := EnsureSandboxExists(c, "No sandboxes found, do you want to create one?", "Select sandbox to delete:", true)
		if sandbox == nil || !acceptToDeleteSandbox(sandbox.Name) {
			os.Exit(0)
		}

		if boxesData := readSandboxesData(); boxesData.Running == sandbox.Name {
			AskToStopSandbox(boxesData)
		}

		boxes := getSandboxesUsingDistro(sandbox.Distro)
		if len(boxes) == 1 && boxes[0].Name == sandbox.Name && acceptToDeleteDistro(sandbox.Distro) {
			deleteDistro(sandbox.Distro)
		}

		deleteSandbox(sandbox.Name)
		fmt.Fprintf(os.Stderr, "Sandbox '%s' deleted.\n", sandbox.Name)

		return nil
	},
}

func acceptToDeleteSandbox(name string) bool {
	return util.PromptBool(fmt.Sprintf("WARNING: This can not be undone ! Do you still want to delete sandbox '%s' ?", name), false)
}

func acceptToDeleteDistro(name string) bool {
	return util.PromptBool(fmt.Sprintf("Distro '%s' is not used any more. Do you want to delete it ?", name), true)
}
