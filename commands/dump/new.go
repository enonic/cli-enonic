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

var New = cli.Command{
	Name:  "new",
	Usage: "Export data from every repository.",
	Flags: append([]cli.Flag{
		cli.StringFlag{
			Name:  "d",
			Usage: "Dump name.",
		},
		cli.StringFlag{
			Name:  "skip-versions",
			Usage: "Don't dump version-history, only current versions included.",
		},
		cli.StringFlag{
			Name:  "max-version-age",
			Usage: "Max age of versions to include, in days, in addition to current version.",
		},
		cli.StringFlag{
			Name:  "max-versions",
			Usage: "Max number of versions to dump in addition to current version.",
		},
	}, common.FLAGS...),
	Action: func(c *cli.Context) error {

		ensureNameFlag(c)

		req := createNewRequest(c)

		fmt.Fprint(os.Stderr, "Creating dump (this may take few minutes)...")
		resp := common.SendRequest(req)

		var dump Dump
		common.ParseResponse(resp, &dump)
		fmt.Fprintf(os.Stderr, "Done %d repositories", len(dump.Repositories))

		return nil
	},
}

func createNewRequest(c *cli.Context) *http.Request {
	body := new(bytes.Buffer)
	params := map[string]interface{}{
		"name": c.String("d"),
	}

	if includeVersions := c.String("skip-versions"); includeVersions != "" {
		params["includeVersions"] = includeVersions
	}
	if maxAge := c.String("max-version-age"); maxAge != "" {
		params["maxAge"] = maxAge
	}
	if maxVersions := c.String("max-versions"); maxVersions != "" {
		params["maxVersions"] = maxVersions
	}
	json.NewEncoder(body).Encode(params)

	return common.CreateRequest(c, "POST", "api/system/dump", body)
}
