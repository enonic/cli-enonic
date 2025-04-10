package export

import (
	"bytes"
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var xslParams map[string]string

var Import = cli.Command{
	Name:  "import",
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
			Usage: "Flag to skips ids when importing",
		},
		cli.BoolFlag{
			Name:  "skip-permissions",
			Usage: "Flag to skips permissions when importing",
		},
		cli.BoolFlag{
			Name:  "dry",
			Usage: "Show the result without making actual changes.",
		},
		common.FORCE_FLAG,
	}, common.AUTH_AND_TLS_FLAGS...),
	Action: func(c *cli.Context) error {

		ensureNameFlag(c)
		ensurePathFlag(c)
		ensureXSLParamsFlagFormat(c)

		req := createLoadRequest(c)

		var result LoadDumpResponse
		status := common.RunTask(c, req, "Importing data", &result)

		switch status.State {
		case common.TASK_FINISHED:
			fmt.Fprintf(os.Stderr, "Added %d nodes, updated %d nodes, imported %d binaries with %d errors in %s\n", len(result.AddedNodes), len(result.UpdateNodes), len(result.ImportedBinaries), len(result.ImportErrors), util.TimeFromNow(status.StartTime))
		case common.TASK_FAILED:
			fmt.Fprintf(os.Stderr, "Import failed: %s\n", status.Progress.Info)
		}
		fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(result))

		return nil
	},
}

func ensureXSLParamsFlagFormat(c *cli.Context) {
	params := c.StringSlice("xsl-param")
	force := common.IsForceMode(c)
	xslParams = make(map[string]string)

	var splitParam []string
	validator := func(val interface{}) error {
		str := val.(string)
		splitParam = strings.Split(str, "=")
		if len(strings.TrimSpace(str)) != 0 && len(splitParam) != 2 {
			if force {
				fmt.Fprintf(os.Stderr, "Xsl parameter '%s' must have the following format <parameter-name>=<parameter-value>\n", str)
				os.Exit(1)
			}
			return errors.Errorf("Xsl parameter '%s' must have the following format <parameter-name>=<parameter-value>: ", str)
		}
		return nil
	}

	for _, param := range params {
		param = util.PromptString(fmt.Sprintf("Xsl parameter '%s'", param), param, "", validator)
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
	params["importWithIds"] = !c.Bool("skip-ids")

	params["importWithPermissions"] = !c.Bool("skip-permissions")

	params["dryRun"] = c.Bool("dry")

	json.NewEncoder(body).Encode(params)

	return common.CreateRequest(c, "POST", "repo/import", body)
}

type LoadDumpResponse struct {
	AddedNodes       []string `json:"addedNodes"`
	UpdateNodes      []string `json:"updateNodes"`
	ImportedBinaries []string `json:"importedBinaries"`
	ImportErrors     []string `json:"importErrors"`
	DryRun           bool     `json:"dryRun"`
}
