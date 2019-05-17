package project

import (
	"fmt"
	"github.com/enonic/cli-enonic/internal/app/commands/sandbox"
	"github.com/urfave/cli"
	"os"
)

var Sandbox = cli.Command{
	Name:    "sandbox",
	Aliases: []string{"sbox", "sb"},
	Usage:   "Set the default sandbox associated with the current project",
	Action: func(c *cli.Context) error {

		ensureValidProjectFolder(".")

		sandbox, _ := sandbox.EnsureSandboxExists(c, "No sandboxes found, do you want to create one?", "Select sandbox to use as default for this project:", true, true)
		if sandbox == nil {
			os.Exit(0)
		}
		writeProjectData(&ProjectData{sandbox.Name}, ".")

		fmt.Fprintf(os.Stderr, "\nSandbox '%s' set as default.\n", sandbox.Name)

		return nil
	},
}
