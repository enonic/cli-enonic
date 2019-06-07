package sandbox

import (
	"fmt"
	"github.com/enonic/cli-enonic/internal/app/commands/common"
	"github.com/urfave/cli"
	"os"
)

var List = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Usage:   "List all sandboxes",
	Action: func(c *cli.Context) error {
		rData := common.ReadRuntimeData()

		for _, box := range listSandboxes() {
			if rData.Running == box.Name {
				fmt.Fprintf(os.Stdout, "* %s ( %s )\n", box.Name, box.Distro)
			} else {
				fmt.Fprintf(os.Stdout, "  %s ( %s )\n", box.Name, box.Distro)
			}
		}
		return nil
	},
}
