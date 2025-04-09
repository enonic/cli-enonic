package system

import (
	"cli-enonic/internal/app/commands/common"
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/urfave/cli"
	"os"
	"time"
)

var Latest = cli.Command{
	Name:  "latest",
	Usage: "Check for latest version",
	Flags: []cli.Flag{common.AUTH_FLAG, common.CRED_FILE_FLAG, common.FORCE_FLAG, common.CLIENT_KEY_FLAG, common.CLIENT_CERT_FLAG},
	Action: func(c *cli.Context) error {
		fmt.Fprintln(os.Stderr, "")

		latestVer := FetchLatestVersion(c)
		currentVer := semver.MustParse(c.App.Version)

		if !latestVer.GreaterThan(currentVer) {
			fmt.Fprintf(os.Stdout, "\nYou are using the latest version of Enonic CLI: %s.\n", c.App.Version)
		} else {
			fmt.Fprintf(os.Stdout, "\nLocal version: %s.\n", c.App.Version)
			fmt.Fprintln(os.Stdout, common.FormatLatestVersionMessage(latestVer.String()))
		}

		return nil
	},
}

func FetchLatestVersion(c *cli.Context) *semver.Version {
	rData := common.ReadRuntimeData()
	rData.LatestCheck = time.Now()

	var latestVer *semver.Version
	common.StartSpinner("Loading")
	isNPM := common.IsInstalledViaNPM()
	if isNPM {
		latestVer = semver.MustParse(common.GetLatestNPMVersion())
		common.StopSpinner()

	} else {
		req := common.CreateRequest(c, "GET", common.SCOOP_MANIFEST_URL, nil)
		res := common.SendRequest(c, req, "Loading")

		var result ScoopManifest
		common.ParseResponse(res, &result)

		latestVer = semver.MustParse(result.Version)
	}

	rData.LatestVersion = latestVer.String()
	common.WriteRuntimeData(rData)

	return latestVer
}

type ScoopManifest struct {
	Version     string
	Homepage    string
	License     string
	Description string
}
