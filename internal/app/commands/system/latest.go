package system

import (
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/enonic/cli-enonic/internal/app/commands/common"
	"github.com/urfave/cli"
	"os"
	"time"
)

var Latest = cli.Command{
	Name:  "latest",
	Usage: "Check for latest version",
	Flags: common.FLAGS,
	Action: func(c *cli.Context) error {
		fmt.Fprintln(os.Stderr, "")
		req := common.CreateRequest(c, "GET", common.SCOOP_MANIFEST_URL, nil)
		res := common.SendRequest(req, "Loading")

		var result ScoopManifest
		common.ParseResponse(res, &result)

		rData := common.ReadRuntimeData()
		rData.LatestCheck = time.Now()

		currentVer := semver.MustParse(c.App.Version)
		latestVer := semver.MustParse(result.Version)
		rData.LatestVersion = result.Version

		if latestVer.Equal(currentVer) || latestVer.LessThan(currentVer) {
			fmt.Fprintf(os.Stdout, "\nYou are using the latest version of Enonic CLI: %s.\n", c.App.Version)
		} else if latestVer.GreaterThan(currentVer) {
			fmt.Fprintf(os.Stdout, "\nLocal version: %s.\n", c.App.Version)
			fmt.Fprintln(os.Stdout, common.FormatLatestVersionMessage(result.Version))
		}
		common.WriteRuntimeData(rData)

		return nil
	},
}

type ScoopManifest struct {
	Version     string
	Homepage    string
	License     string
	Description string
}
