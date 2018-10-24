package dump

import (
	"github.com/urfave/cli"
	"enonic.com/xp-cli/commands/common"
	"fmt"
	"os"
	"net/http"
	"bytes"
	"encoding/json"
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

		fmt.Fprint(os.Stderr, "Upgrading dump...")
		resp := common.SendRequest(req)

		var result UpgradeResult
		common.ParseResponse(resp, &result)
		if result.InitialVersion != result.UpgradedVersion {
			fmt.Fprintf(os.Stderr, "Upgraded from version '%s' to '%s'\n", result.InitialVersion, result.UpgradedVersion)
		} else {
			fmt.Fprintf(os.Stderr, "You already have the latest version '%s'\n", result.InitialVersion)
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
