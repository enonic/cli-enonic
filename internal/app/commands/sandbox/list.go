package sandbox

import (
	"github.com/urfave/cli"
	"fmt"
	"os"
)

var List = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Usage:   "List all sandboxes",
	Action: func(c *cli.Context) error {
		data := readSandboxesData()

		for _, b := range listSandboxes() {
			if data.Running == b {
				fmt.Fprintf(os.Stderr, "* %s\n", b)
			} else {
				fmt.Fprintf(os.Stderr, "  %s\n", b)
			}
		}
		return nil
	},
}
