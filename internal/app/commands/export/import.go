package export

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
			Usage: "Show the result without making actual changes. Only effective in compat mode (XP 7).",
		},
		common.FORCE_FLAG,
	}, append(common.AUTH_AND_TLS_FLAGS, common.COMPAT_FLAG)...),
	Action: func(c *cli.Context) error {

		util.Fatal(common.ValidateCompatFlag(c), "Invalid argument")

		ensureNameFlag(c)
		ensurePathFlag(c)

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

func createLoadRequest(c *cli.Context) *http.Request {
	body := new(bytes.Buffer)
	params := map[string]interface{}{
		"exportName":     c.String("t"),
		"targetRepoPath": c.String("path"),
	}

	params["importWithIds"] = !c.Bool("skip-ids")

	params["importWithPermissions"] = !c.Bool("skip-permissions")

	if common.IsCompatMode(c) {
		params["dryRun"] = c.Bool("dry")
	}

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
