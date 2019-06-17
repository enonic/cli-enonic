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
		fmt.Fprintf(os.Stdout, "\nLatest version: %s\n", result.Version)
		fmt.Fprintf(os.Stdout, "Local version: %s\n", c.App.Version)

		currentVer := semver.MustParse(rData.LatestVersion)
		latestVer := semver.MustParse(result.Version)
		if latestVer.GreaterThan(currentVer) {
			fmt.Fprintln(os.Stdout, common.FormatLatestVersionMessage(result.Version))
			rData.LatestVersion = result.Version
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
