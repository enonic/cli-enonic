package project

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/commands/sandbox"
	"fmt"
	"github.com/urfave/cli"
	"os"
)

var Sandbox = cli.Command{
	Name:      "sandbox",
	Aliases:   []string{"sbox", "sb"},
	Usage:     "Set the default sandbox associated with the current project",
	ArgsUsage: "<sandbox name>",
	Flags:     []cli.Flag{common.FORCE_FLAG},
	Action: func(c *cli.Context) error {

		ensureValidProjectFolder(".")

		var sandboxName string
		if c.NArg() > 0 {
			sandboxName = c.Args().First()
		}
		minDistroVersion := common.ReadProjectDistroVersion(".")
		sandbox, _ := sandbox.EnsureSandboxExists(c, minDistroVersion, sandboxName, "No sandboxes found, do you want to create one", "Select sandbox to use as default for this project", true, true)
		if sandbox == nil {
			os.Exit(1)
		}
		common.WriteProjectData(&common.ProjectData{sandbox.Name}, ".")

		fmt.Fprintf(os.Stdout, "\nSandbox '%s' set as default.\n", sandbox.Name)

		return nil
	},
}
