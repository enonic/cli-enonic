package project

import (
	"github.com/urfave/cli"
	"fmt"
	"os"
	"github.com/enonic/xp-cli/internal/app/commands/sandbox"
)

var Sandbox = cli.Command{
	Name:    "sandbox",
	Aliases: []string{"sbox", "sb"},
	Usage:   "Set the default sandbox associated with the current project",
	Action: func(c *cli.Context) error {

		sandbox := sandbox.EnsureSandboxNameExists(c, "No sandboxes found, do you want to create one?", "Select sandbox to use as default for this project:")
		writeProjectData(ProjectData{sandbox.Name})

		fmt.Fprintf(os.Stderr, "Sandbox '%s' set as default", sandbox.Name)

		return nil
	},
}
