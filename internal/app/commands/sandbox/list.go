package sandbox

import (
	"fmt"
	"github.com/urfave/cli"
	"os"
)

var List = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Usage:   "List all sandboxes",
	Action: func(c *cli.Context) error {
		data := ReadSandboxesData()

		for _, box := range listSandboxes() {
			if data.Running == box.Name {
				fmt.Fprintf(os.Stderr, "* %s ( %s )\n", box.Name, box.Distro)
			} else {
				fmt.Fprintf(os.Stderr, "  %s ( %s )\n", box.Name, box.Distro)
			}
		}
		return nil
	},
}
