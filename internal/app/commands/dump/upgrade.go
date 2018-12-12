package dump

import (
	"github.com/urfave/cli"
	"github.com/enonic/xp-cli/internal/app/commands/common"
	"fmt"
	"os"
	"net/http"
	"bytes"
	"encoding/json"
	"time"
)

var Upgrade = cli.Command{
	Name:    "upgrade",
	Aliases: []string{"up"},
	Usage:   "Upgrade a dump.",
	Flags: append([]cli.Flag{
		cli.StringFlag{
			Name:  "d",
			Usage: "Dump name.",
		},
	}, common.FLAGS...),
	Action: func(c *cli.Context) error {

		ensureNameFlag(c)

		req := createUpgradeRequest(c)
		var result UpgradeResult
		status := common.RunTask(c, req, "Upgrading dump...", &result)

		switch status.State {
		case common.TASK_FINISHED:
			if result.InitialVersion != result.UpgradedVersion {
				fmt.Fprintf(os.Stderr, "Upgraded from version '%s' to '%s' in %v\n", result.InitialVersion, result.UpgradedVersion, time.Now().Sub(status.StartTime))
			} else {
				fmt.Fprintf(os.Stderr, "You already have the latest version '%s'\n", result.InitialVersion)
			}
		case common.TASK_FAILED:
			fmt.Fprintf(os.Stderr, "Failed to upgrade dump: %s", status.Progress.Info)
		}

		return nil
	},
}

func createUpgradeRequest(c *cli.Context) *http.Request {
	body := new(bytes.Buffer)
	params := map[string]string{
		"name": c.String("d"),
	}
	json.NewEncoder(body).Encode(params)

	return common.CreateRequest(c, "POST", "api/system/upgrade", body)
}

type UpgradeResult struct {
	InitialVersion  string `json:initialVersion`
	UpgradedVersion string `json:upgradedVersion`
}
