package sandbox

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"fmt"
	"github.com/urfave/cli"
	"os"
)

var List = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Flags:   []cli.Flag{common.FORCE_FLAG},
	Usage:   "List all sandboxes",
	Action: func(c *cli.Context) error {
		rData := common.ReadRuntimeData()
		myOs := util.GetCurrentOs()

		for _, box := range listSandboxes("") {
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
