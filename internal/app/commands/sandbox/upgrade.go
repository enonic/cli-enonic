package sandbox

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/urfave/cli"
	"os"
)

var Upgrade = cli.Command{
	Name:    "upgrade",
	Aliases: []string{"up"},
	Usage:   "Upgrades the distribution version.",
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

		sandbox, _ := EnsureSandboxExists(c, "", "No sandboxes found, do you want to create one?", "Select sandbox:", true, false, true)
		if sandbox == nil {
			os.Exit(1)
		}
		version := ensureVersionCorrect(c.String("version"), "", c.Bool("all"), common.IsForceMode(c))
		preventVersionDowngrade(sandbox, version)

		sandbox.Distro = formatDistroVersion(version)
		writeSandboxData(sandbox)
		fmt.Fprintf(os.Stdout, "Sandbox '%s' distro upgraded to '%s'.\n", sandbox.Name, sandbox.Distro)

		return nil
	},
}

func preventVersionDowngrade(sandbox *Sandbox, newString string) {
	distroVer := parseDistroVersion(sandbox.Distro, false)
	oldVer, err := semver.NewVersion(distroVer)
	util.Fatal(err, fmt.Sprintf("Could not parse sandbox distro version from: %s", sandbox.Distro))
	newVer, err2 := semver.NewVersion(newString)
	util.Fatal(err2, fmt.Sprintf("Could not parse new distro version from: %s", newString))

	if oldVer.Compare(newVer) > 0 {
		fmt.Fprintf(os.Stdout, "Sandbox '%s' already has newer distro version: %s\n", sandbox.Name, sandbox.Distro)
		os.Exit(0)
	}
}
