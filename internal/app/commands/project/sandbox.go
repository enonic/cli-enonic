package project

import (
	"github.com/urfave/cli"
	"github.com/enonic/xp-cli/internal/app/commands/sandbox"
	"fmt"
	"os"
)

var Sandbox = cli.Command{
	Name:    "sandbox",
	Aliases: []string{"sbox", "sb"},
	Usage:   "Set the default sandbox associated with the current project",
	Action: func(c *cli.Context) error {

		sandbox := sandbox.EnsureSandboxNameExists(c, "Select sandbox to use as default for this project:")
		writeProjectData(ProjectData{sandbox.Name})

		fmt.Fprintf(os.Stderr, "Sandbox '%s' set as default", sandbox.Name)

		return nil
	},
}
