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

		sandbox := sandbox.EnsureSandboxNameExists(c, "Select sandbox to attach to:")
		data := readProjectData()
		data.Sandbox = sandbox.Name
		writeProjectData(data)

		fmt.Fprintf(os.Stderr, "Attached current project to sandbox '%s'", sandbox.Name)

		return nil
	},
}
