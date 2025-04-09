package dump

import (
	"bytes"
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/urfave/cli"
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
	}, common.AUTH_FLAG, common.CRED_FILE_FLAG, common.FORCE_FLAG, common.CLIENT_KEY_FLAG, common.CLIENT_CERT_FLAG),
	Action: func(c *cli.Context) error {

		name := ensureNameFlag(c.String("d"), false, common.IsForceMode(c))

		req := createUpgradeRequest(c, name)
		var result UpgradeResult
		status := common.RunTask(c, req, "Upgrading dump", &result)

		switch status.State {
		case common.TASK_FINISHED:
			if result.InitialVersion != result.UpgradedVersion {
				fmt.Fprintf(os.Stderr, "Upgraded from version '%s' to '%s' in %s\n", result.InitialVersion, result.UpgradedVersion, util.TimeFromNow(status.StartTime))
			} else {
				fmt.Fprintf(os.Stderr, "You already have the latest version '%s'\n", result.InitialVersion)
			}
		case common.TASK_FAILED:
			fmt.Fprintf(os.Stderr, "Failed to upgrade dump: %s\n", status.Progress.Info)
		}
		fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(result))

		return nil
	},
}

func createUpgradeRequest(c *cli.Context, name string) *http.Request {
	body := new(bytes.Buffer)
	params := map[string]string{
		"name": name,
	}
	json.NewEncoder(body).Encode(params)

	return common.CreateRequest(c, "POST", "system/upgrade", body)
}

type UpgradeResult struct {
	InitialVersion  string `json:"initialVersion"`
	UpgradedVersion string `json:"upgradedVersion"`
}
