package sandbox

import (
	"cli-enonic/internal/app/commands/common"
	"fmt"
	"github.com/urfave/cli"
	"os"
)

var Upgrade = cli.Command{
	Name:      "upgrade",
	Aliases:   []string{"up"},
	Usage:     "Upgrades the distribution version.",
	ArgsUsage: "<name>",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "version, v",
			Usage: "Distro version to upgrade to.",
		},
		cli.BoolFlag{
			Name:  "all, a",
			Usage: "List all distro versions.",
		},
		common.FORCE_FLAG,
	},
	Action: func(c *cli.Context) error {

		var sandboxName string
		if c.NArg() > 0 {
			sandboxName = c.Args().First()
		} else if common.HasProjectData(".") {
			prjData := common.ReadProjectData(".")
			sandboxName = prjData.Sandbox
		}
		sandbox, _ := EnsureSandboxExists(c, EnsureSandboxOptions{
			Name:               sandboxName,
			SelectBoxMessage:   "Select sandbox to upgrade",
			ShowSuccessMessage: true,
		})
		if sandbox == nil {
			os.Exit(1)
		}
		minDistroVer := parseDistroVersion(sandbox.Distro, false)
		version, total := ensureVersionCorrect(c, c.String("version"), minDistroVer, false, c.Bool("all"), common.IsForceMode(c))
		if total == 0 {
			fmt.Fprintf(os.Stdout, "Sandbox '%s' is using the latest release of Enonic XP\n", sandbox.Name)
			os.Exit(0)
		}

		sandbox.Distro = formatDistroVersion(version)
		writeSandboxData(sandbox)
		fmt.Fprintf(os.Stdout, "Sandbox '%s' distro upgraded to '%s'.\n", sandbox.Name, sandbox.Distro)

		return nil
	},
}
