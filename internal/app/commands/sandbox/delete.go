package sandbox

import (
	"fmt"
	"github.com/enonic/cli-enonic/internal/app/commands/common"
	"github.com/enonic/cli-enonic/internal/app/util"
	"github.com/urfave/cli"
	"os"
)

var Delete = cli.Command{
	Name:    "delete",
	Usage:   "Delete a sandbox",
	Aliases: []string{"del", "rm"},
	Flags:   []cli.Flag{common.FORCE_FLAG},
	Action: func(c *cli.Context) error {
		sandbox, _ := EnsureSandboxExists(c, "", "No sandboxes found, do you want to create one?", "Select sandbox to delete:", true, false, true)
		force := common.IsForceMode(c)
		if sandbox == nil || !acceptToDeleteSandbox(sandbox.Name, force) {
			os.Exit(1)
		}

		if rData := common.ReadRuntimeData(); rData.Running == sandbox.Name {
			AskToStopSandbox(rData, force)
		}

		boxes := getSandboxesUsingDistro(sandbox.Distro)
		if len(boxes) == 1 && boxes[0].Name == sandbox.Name && acceptToDeleteDistro(sandbox.Distro, force) {
			deleteDistro(sandbox.Distro)
		}

		deleteSandbox(sandbox.Name)
		fmt.Fprintf(os.Stdout, "Sandbox '%s' deleted.\n", sandbox.Name)

		return nil
	},
}

func acceptToDeleteSandbox(name string, force bool) bool {
	return force || util.PromptBool(fmt.Sprintf("WARNING: This can not be undone ! Do you still want to delete sandbox '%s' ?", name), false)
}

func acceptToDeleteDistro(name string, force bool) bool {
	return force || util.PromptBool(fmt.Sprintf("Distro '%s' is not used any more. Do you want to delete it ?", name), true)
}
