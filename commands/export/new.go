package export

import (
	"github.com/urfave/cli"
	"enonic.com/xp-cli/commands/common"
	"fmt"
	"os"
	"net/http"
	"bytes"
	"encoding/json"
	"enonic.com/xp-cli/util"
	"strings"
)

var New = cli.Command{
	Name:  "new",
	Usage: "Export data from every repository.",
	Flags: append([]cli.Flag{
		cli.StringFlag{
			Name:  "t",
			Usage: "Target name to save export.",
		},
		cli.StringFlag{
			Name:  "path",
			Usage: "Path of data to export. Format: <repo-name>:<branch-name>:<node-path>.",
		},
		cli.BoolFlag{
			Name:  "skip-ids",
			Usage: "Flag that skips ids in data when exporting.",
		},
		cli.BoolFlag{
			Name:  "skip-versions",
			Usage: "Flag that skips versions in data when exporting.",
		},
		cli.BoolFlag{
			Name:  "dry",
			Usage: "Don't modify anything when exporting.",
		},
	}, common.FLAGS...),
	Action: func(c *cli.Context) error {

		ensureNameFlag(c)
		ensurePathFlag(c)

		req := createNewRequest(c)

		fmt.Fprint(os.Stderr, "Exporting data (this may take few minutes)...")
		resp := common.SendRequest(req)

		var result NewExportResponse
		common.ParseResponse(resp, &result)
		fmt.Fprintf(os.Stderr, "Exported %d nodes and %d binaries with %d errors\n", len(result.ExportedNodes), len(result.ExportedBinaries), len(result.Errors))

		return nil
	},
}

func createNewRequest(c *cli.Context) *http.Request {
	body := new(bytes.Buffer)
	params := map[string]interface{}{
		"exportName": c.String("t"),
	}

	if path := c.String("path"); path != "" {
		params["sourceRepoPath"] = path
	}
	if skipIds := c.Bool("skip-ids"); skipIds {
		params["exportWithIds"] = !skipIds
	}
	if skipVersions := c.Bool("skip-versions"); skipVersions {
		params["includeVersions"] = !skipVersions
	}
	if dry := c.Bool("dry"); dry {
		params["dryRun"] = dry
	}
	json.NewEncoder(body).Encode(params)

	return common.CreateRequest(c, "POST", "api/repo/export", body)
}

func ensurePathFlag(c *cli.Context) {
	var name = c.String("path")

	name = util.PromptUntilTrue(name, func(val string, ind byte) string {
		if len(strings.TrimSpace(val)) == 0 {
			switch ind {
			case 0:
				return "Enter source repo path (<repo-name>:<branch-name>:<node-path>): "
			default:
				return "Source repo path can not be empty (<repo-name>:<branch-name>:<node-path>): "
			}
		} else {
			splitPathLen := len(strings.Split(val, ":"))
			if splitPathLen != 3 {
				return "Source repo path must have the following format <repo-name>:<branch-name>:<node-path>: "
			} else {
				return ""
			}
		}
	})

	c.Set("path", name)
}

type NewExportResponse struct {
	DryRun           bool     `json:dryRun`
	ExportedBinaries []string `json:exportedBinaries`
	ExportedNodes    []string `json:exportedNodes`
	Errors []struct {
		message string `json:message`
	} `json:exportErrors`
}
