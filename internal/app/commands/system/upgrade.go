package system

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util/system"
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/urfave/cli"
	"os"
	"strings"
)

var Upgrade = cli.Command{
	Name:  "upgrade",
	Usage: "Upgrade to the latest version",
	Action: func(c *cli.Context) error {
		fmt.Fprintln(os.Stderr, "")

		latestVer := FetchLatestVersion(c)
		currentVer := semver.MustParse(c.App.Version)

		if !latestVer.GreaterThan(currentVer) {
			fmt.Fprintf(os.Stdout, "\nYou are using the latest version of Enonic CLI: %s.\n", c.App.Version)
			return nil
		}

		isNPM := common.IsInstalledViaNPM()
		upgradeCommand := common.GetOSUpdateCommand(isNPM)
		if upgradeCommand != "" {
			upgradeArgs := strings.Split(upgradeCommand, " ")

			system.Run(upgradeArgs[0], upgradeArgs[1:], os.Environ())
		} else {
			// not installed with package manager
			fmt.Fprintln(os.Stderr, "Could not upgrade. If you installed enonic CLI manually, you need to upgrade manually too.")
		}
		return nil
	},
}
