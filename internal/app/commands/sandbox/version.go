package sandbox

import (
	"github.com/urfave/cli"
	"fmt"
	"os"
)

var Version = cli.Command{
	Name:    "version",
	Aliases: []string{"ver"},
	Usage:   "Updates the distribution version.",
	Action: func(c *cli.Context) error {

		sandbox := ensureSandboxNameExists(c, "Select sandbox:")
		version := ensureVersionArg(c)

		ensureDistroPresent(version)
		writeSandboxData(sandbox.Name, SandboxData{version})
		fmt.Fprintf(os.Stderr, "Sandbox '%s' distro set to: %s", sandbox.Name, version)

		return nil
	},
}

func ensureVersionArg(c *cli.Context) string {
	var version string
	if c.NArg() > 1 {
		version = c.Args().Get(1) // Get the second, as the first is the name
	}
	return ensureVersionCorrect(version)
}
