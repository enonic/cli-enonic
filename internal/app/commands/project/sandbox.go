package project

import (
	"github.com/urfave/cli"
	"fmt"
	"os"
	"github.com/enonic/enonic-cli/internal/app/commands/sandbox"
)

var Sandbox = cli.Command{
	Name:    "sandbox",
	Aliases: []string{"sbox", "sb"},
	Usage:   "Set the default sandbox associated with the current project",
	Action: func(c *cli.Context) error {

		ensureValidProjectFolder(".")

		sandbox, _ := sandbox.EnsureSandboxExists(c, "No sandboxes found, do you want to create one?", "Select sandbox to use as default for this project:", true)
		if sandbox == nil {
			os.Exit(0)
		}
		writeProjectData(&ProjectData{sandbox.Name}, ".")

		fmt.Fprintf(os.Stderr, "\nSandbox '%s' set as default.\n", sandbox.Name)

		return nil
	},
}
