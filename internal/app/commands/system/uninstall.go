package system

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"cli-enonic/internal/app/util/system"
	"github.com/urfave/cli"
	"os"
	"strings"
)

var Uninstall = cli.Command{
	Name:  "uninstall",
	Usage: "Uninstall Enonic CLI",
	Flags: []cli.Flag{common.AUTH_FLAG, common.FORCE_FLAG},
	Action: func(c *cli.Context) error {

		if answer := common.IsForceMode(c) ||
			util.PromptBool("Do you want to remove Enonic CLI from your system", false); answer {

			isNPM := common.IsInstalledViaNPM()
			uninstallCommand := common.GetOSUninstallCommand(isNPM)
			uninstallArgs := strings.Split(uninstallCommand, " ")

			system.Run(uninstallArgs[0], uninstallArgs[1:], os.Environ())
		}

		return nil
	},
}
