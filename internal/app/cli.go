package main

import (
	"github.com/Masterminds/semver"
	"github.com/enonic/cli-enonic/internal/app/commands"
	"github.com/enonic/cli-enonic/internal/app/commands/common"
	"github.com/enonic/cli-enonic/internal/app/util"
	"github.com/urfave/cli"
	"log"
	"math"
	"os"
	"time"
)

// set by goreleaser
// https://goreleaser.com/environment/
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	app := cli.NewApp()
	app.Name = "Enonic CLI"
	app.Version = version
	app.Usage = "Manage XP instances, home folders and projects"
	app.Commands = commands.All()
	app.Metadata = populateMeta(app)

	util.SetupTemplates(app)

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func populateMeta(app *cli.App) map[string]interface{} {
	meta := make(map[string]interface{})

	rData := common.ReadRuntimeData()
	if rData.LatestCheck.IsZero() {
		// this is the first check so set it to now
		rData.LatestCheck = time.Now()
		rData.LatestVersion = app.Version
		common.WriteRuntimeData(rData)
	}

	daysSinceLastCheck := time.Since(rData.LatestCheck).Hours() / 24
	if daysSinceLastCheck > 30 {
		meta["LatestCheck"] = math.Round(daysSinceLastCheck)
	} else {
		latestVer := semver.MustParse(rData.LatestVersion)
		currentVer := semver.MustParse(app.Version)
		if currentVer.LessThan(latestVer) {
			meta["LatestVersion"] = rData.LatestVersion
		}
	}

	return meta
}
