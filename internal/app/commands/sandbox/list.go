package sandbox

import (
	"fmt"
	"github.com/enonic/cli-enonic/internal/app/commands/common"
	"github.com/enonic/cli-enonic/internal/app/util"
	"github.com/urfave/cli"
	"os"
)

var List = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Usage:   "List all sandboxes",
	Action: func(c *cli.Context) error {
		rData := common.ReadRuntimeData()
		myOs := util.GetCurrentOs()

		for _, box := range listSandboxes() {
			version := parseDistroVersion(box.Distro, false)
			boxVersion := formatSandboxListItemName(box.Name, version, myOs)
			if rData.Running == box.Name {
				fmt.Fprintf(os.Stdout, "* %s\n", boxVersion)
			} else {
				fmt.Fprintf(os.Stdout, "  %s\n", boxVersion)
			}
		}
		return nil
	},
}
