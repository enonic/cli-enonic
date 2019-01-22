package export

import (
	"github.com/urfave/cli"
	"github.com/enonic/xp-cli/internal/app/commands/common"
	"os"
	"fmt"
	"net/http"
	"bytes"
	"encoding/json"
	"github.com/enonic/xp-cli/internal/app/util"
	"strings"
)

var xslParams map[string]string

var Load = cli.Command{
	Name:  "load",
	Usage: "Import data from a named export.",
	Flags: append([]cli.Flag{
		cli.StringFlag{
			Name:  "t",
			Usage: "A named export to import.",
		},
		cli.StringFlag{
			Name:  "path",
			Usage: "Target path for import. Format: <repo-name>:<branch-name>:<node-path> e.g. 'cms-repo:draft:/'",
		},
		cli.StringFlag{
			Name:  "xsl-source",
			Usage: "Path to xsl file (relative to <XP_HOME>/data/export) for applying transformations to node.xml before importing.",
		},
		cli.StringSliceFlag{
			Name:  "xsl-param",
			Usage: "Parameters to pass to the XSL transformations before importing nodes. Format: <parameter-name>=<parameter-value> e.g. 'applicationId=com.enonic.myapp'",
		},
		cli.BoolFlag{
			Name:  "skip-ids",
			Usage: "Flag that skips ids.",
		},
		cli.BoolFlag{
			Name:  "skip-permissions",
			Usage: "Flag that skips permissions.",
		},
		cli.BoolFlag{
			Name:  "dry",
			Usage: "Show the result without making actual changes.",
		},
	}, common.FLAGS...),
	Action: func(c *cli.Context) error {

		ensureNameFlag(c)
		ensurePathFlag(c)
		ensureXSLParamsFlagFormat(c)

		req := createLoadRequest(c)

		var result LoadDumpResponse
		status := common.RunTask(req, "Importing data", &result)

		switch status.State {
		case common.TASK_FINISHED:
			fmt.Fprintf(os.Stderr, "Added %d nodes, updated %d nodes, imported %d binaries with %d errors in %s\n", len(result.AddedNodes), len(result.UpdateNodes), len(result.ImportedBinaries), len(result.ImportErrors), util.TimeFromNow(status.StartTime))
		case common.TASK_FAILED:
			fmt.Fprintf(os.Stderr, "Import failed: %s\n", status.Progress.Info)
		}
		fmt.Fprintln(os.Stderr, util.PrettyPrintJSON(result))

		return nil
	},
}

func ensureXSLParamsFlagFormat(c *cli.Context) {
	params := c.StringSlice("xsl-param")
	xslParams = make(map[string]string)

	for _, param := range params {
		var splitParam []string
		param = util.PromptUntilTrue(param, func(val *string, ind byte) string {
			splitParam = strings.Split(*val, "=")
			if len(strings.TrimSpace(*val)) == 0 || len(splitParam) == 2 {
				return ""
			} else {
				return fmt.Sprintf("Xsl parameter '%s' must have the following format <parameter-name>=<parameter-value>: ", param)
			}
		})
		xslParams[splitParam[0]] = splitParam[1]
	}
}

func createLoadRequest(c *cli.Context) *http.Request {
	body := new(bytes.Buffer)
	params := map[string]interface{}{
		"exportName":     c.String("t"),
		"targetRepoPath": c.String("path"),
	}

	if xslSource := c.String("xsl-source"); xslSource != "" {
		params["xslSource"] = xslSource
	}
	if len(xslParams) > 0 {
		params["xslParams"] = xslParams
	}
	if skipIds := c.Bool("skip-ids"); skipIds {
		params["importWithIds"] = !skipIds
	}
	if skipPermissions := c.Bool("skip-permissions"); skipPermissions {
		params["importWithPermissions"] = !skipPermissions
	}
	if dry := c.Bool("dry"); dry {
		params["dryRun"] = dry
	}
	json.NewEncoder(body).Encode(params)

	return common.CreateRequest(c, "POST", "repo/import", body)
}

type LoadDumpResponse struct {
	AddedNodes       []string `json:addedNodes`
	UpdateNodes      []string `json:updateNodes`
	ImportedBinaries []string `json:importedBinaries`
	ImportErrors     []string `json:importErrors`
	DryRun           bool     `dryRun`
}
