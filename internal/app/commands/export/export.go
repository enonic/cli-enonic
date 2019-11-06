package export

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/enonic/cli-enonic/internal/app/commands/common"
	"github.com/enonic/cli-enonic/internal/app/util"
	"github.com/urfave/cli"
	"net/http"
	"os"
	"strings"
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
	DryRun           bool     `json:dryRun`
	ExportedBinaries []string `json:exportedBinaries`
	ExportedNodes    []string `json:exportedNodes`
	Errors           []struct {
		message string `json:message`
	} `json:exportErrors`
}

func ensureNameFlag(c *cli.Context) {
	if c.String("t") == "" {

		var name string
		name = util.PromptUntilTrue(name, func(val *string, ind byte) string {
			if len(strings.TrimSpace(*val)) == 0 {
				switch ind {
				case 0:
					return "Enter target name: "
				default:
					return "Target name can not be empty: "
				}
			} else {
				return ""
			}
		})

		c.Set("t", name)
	}
}

func ensurePathFlag(c *cli.Context) {
	var path = c.String("path")

	path = util.PromptUntilTrue(path, func(val *string, ind byte) string {
		if len(strings.TrimSpace(*val)) == 0 {
			switch ind {
			case 0:
				return "Enter source repo path (<repo-name>:<branch-name>:<node-path>): "
			default:
				return "Source repo path can not be empty (<repo-name>:<branch-name>:<node-path>): "
			}
		} else {
			splitPathLen := len(strings.Split(*val, ":"))
			if splitPathLen != 3 {
				return fmt.Sprintf("Source repo path '%s' must have the following format <repo-name>:<branch-name>:<node-path>: ", *val)
			} else {
				return ""
			}
		}
	})

	c.Set("path", path)
}
