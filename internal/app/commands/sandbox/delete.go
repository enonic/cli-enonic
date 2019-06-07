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
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "f, force",
			Usage: "assume “Yes” as answer to all prompts and run non-interactively",
		},
	},
	Action: func(c *cli.Context) error {
		sandbox, _ := EnsureSandboxExists(c, "No sandboxes found, do you want to create one?", "Select sandbox to delete:", true, false)
		if sandbox == nil || !(c.Bool("f") || acceptToDeleteSandbox(sandbox.Name)) {
			os.Exit(0)
		}

		if rData := common.ReadRuntimeData(); rData.Running == sandbox.Name {
			AskToStopSandbox(rData)
		}

		boxes := getSandboxesUsingDistro(sandbox.Distro)
		if len(boxes) == 1 && boxes[0].Name == sandbox.Name && acceptToDeleteDistro(sandbox.Distro) {
			deleteDistro(sandbox.Distro)
		}

		deleteSandbox(sandbox.Name)
		fmt.Fprintf(os.Stdout, "Sandbox '%s' deleted.\n", sandbox.Name)

		return nil
	},
}

func acceptToDeleteSandbox(name string) bool {
	return util.PromptBool(fmt.Sprintf("WARNING: This can not be undone ! Do you still want to delete sandbox '%s' ?", name), false)
}

func acceptToDeleteDistro(name string) bool {
	return util.PromptBool(fmt.Sprintf("Distro '%s' is not used any more. Do you want to delete it ?", name), true)
}
