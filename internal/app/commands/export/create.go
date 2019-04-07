package export

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/enonic/enonic-cli/internal/app/commands/common"
	"github.com/enonic/enonic-cli/internal/app/util"
	"github.com/urfave/cli"
	"net/http"
	"os"
)

var Create = cli.Command{
	Name:  "create",
	Usage: "Export data from a given repository, branch and content path.",
	Flags: append([]cli.Flag{
		cli.StringFlag{
			Name:  "t",
			Usage: "Target name to save export.",
		},
		cli.StringFlag{
			Name:  "path",
			Usage: "Path of data to export. Format: <repo-name>:<branch-name>:<node-path> e.g. 'cms-repo:draft:/'",
		},
		cli.BoolFlag{
			Name:  "skip-ids",
			Usage: "Flag to skip ids in data when exporting.",
		},
		cli.BoolFlag{
			Name:  "skip-versions",
			Usage: "Flag to skip versions in data when exporting.",
		},
		cli.BoolFlag{
			Name:  "dry",
			Usage: "Show the result without making actual changes.",
		},
	}, common.FLAGS...),
	Action: func(c *cli.Context) error {

		ensureNameFlag(c)
		ensurePathFlag(c)

		req := createNewRequest(c)
		var result NewExportResponse
		status := common.RunTask(req, "Exporting data", &result)

		switch status.State {
		case common.TASK_FINISHED:
			fmt.Fprintf(os.Stderr, "Exported %d nodes and %d binaries with %d errors in %s\n", len(result.ExportedNodes), len(result.ExportedBinaries), len(result.Errors), util.TimeFromNow(status.StartTime))
		case common.TASK_FAILED:
			fmt.Fprintf(os.Stderr, "Export failed: %s\n", status.Progress.Info)
		}
		fmt.Fprintln(os.Stderr, util.PrettyPrintJSON(result))

		return nil
	},
}

func createNewRequest(c *cli.Context) *http.Request {
	body := new(bytes.Buffer)
	params := map[string]interface{}{
		"exportName":     c.String("t"),
		"sourceRepoPath": c.String("path"),
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

	return common.CreateRequest(c, "POST", "repo/export", body)
}

type NewExportResponse struct {
	DryRun           bool     `json:dryRun`
	ExportedBinaries []string `json:exportedBinaries`
	ExportedNodes    []string `json:exportedNodes`
	Errors []struct {
		message string `json:message`
	} `json:exportErrors`
}
