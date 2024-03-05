package sandbox

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"fmt"
	"github.com/urfave/cli"
	"os"
)

var Delete = cli.Command{
	Name:      "delete",
	Usage:     "Delete a sandbox",
	ArgsUsage: "<name>",
	Aliases:   []string{"del", "rm"},
	Flags:     []cli.Flag{common.FORCE_FLAG},
	Action: func(c *cli.Context) error {

		var sandboxName string
		if c.NArg() > 0 {
			sandboxName = c.Args().First()
		}
		sandbox, _ := EnsureSandboxExists(c, EnsureSandboxOptions{
			Name:               sandboxName,
			SelectBoxMessage:   "Select sandbox to delete",
			ShowSuccessMessage: true,
		})
		force := common.IsForceMode(c)
		if sandbox == nil || !acceptToDeleteSandbox(sandbox.Name, force) {
			os.Exit(1)
		}

		if rData := common.ReadRuntimeData(); rData.Running == sandbox.Name {
			if !AskToStopSandbox(rData, force) {
				os.Exit(1)
			}
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
	return force || util.PromptBool(fmt.Sprintf("WARNING: This can not be undone ! Do you still want to delete sandbox '%s'", name), false)
}

func acceptToDeleteDistro(name string, force bool) bool {
	return force || util.PromptBool(fmt.Sprintf("Distro '%s' is not used any more. Do you want to delete it", name), true)
}
