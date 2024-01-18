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

var Export = cli.Command{
	Name:  "export",
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
	}, common.AUTH_FLAG, common.FORCE_FLAG),
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
		fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(result))

		return nil
	},
}

func createNewRequest(c *cli.Context) *http.Request {
	body := new(bytes.Buffer)
	params := map[string]interface{}{
		"exportName":     c.String("t"),
		"sourceRepoPath": c.String("path"),
	}

	params["exportWithIds"] = !c.Bool("skip-ids")

	params["includeVersions"] = !c.Bool("skip-versions")

	params["dryRun"] = c.Bool("dry")

	json.NewEncoder(body).Encode(params)

	return common.CreateRequest(c, "POST", "repo/export", body)
}

type NewExportResponse struct {
	DryRun           bool     `json:"dryRun"`
	ExportedBinaries []string `json:"exportedBinaries"`
	ExportedNodes    []string `json:"exportedNodes"`
	Errors           []struct {
		Message string `json:"message"`
	} `json:"exportErrors"`
}

func ensureNameFlag(c *cli.Context) {
	target := c.String("t")

	targetValidator := func(val interface{}) error {
		str := val.(string)
		if len(strings.TrimSpace(str)) == 0 {
			if common.IsForceMode(c) {
				fmt.Fprintln(os.Stderr, "Target name can not be empty in non-interactive mode.")
				os.Exit(1)
			}
			return errors.New("Target name can not be empty: ")
		} else {
			return nil
		}
	}
	target = util.PromptString("Enter target name", target, "", targetValidator)

	c.Set("t", target)
}

func ensurePathFlag(c *cli.Context) {
	path := c.String("path")
	force := common.IsForceMode(c)

	pathValidator := func(val interface{}) error {
		str := val.(string)
		if len(strings.TrimSpace(str)) == 0 {
			if force {
				fmt.Fprintln(os.Stderr, "Source repo path can not be empty in non-interactive mode.")
				os.Exit(1)
			}
			return errors.New("Source repo path can not be empty (<repo-name>:<branch-name>:<node-path>): ")
		} else {
			splitPathLen := len(strings.Split(str, ":"))
			if splitPathLen != 3 {
				if force {
					fmt.Fprintf(os.Stderr, "Source repo path '%s' must have the following format <repo-name>:<branch-name>:<node-path>\n", str)
					os.Exit(1)
				}
				return errors.Errorf("Source repo path '%s' must have the following format <repo-name>:<branch-name>:<node-path>: ", str)
			} else {
				return nil
			}
		}
	}

	path = util.PromptString("Enter source repo path (<repo-name>:<branch-name>:<node-path>)", path, "", pathValidator)

	c.Set("path", path)
}
