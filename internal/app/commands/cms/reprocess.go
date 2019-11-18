package cms

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

var Reprocess = cli.Command{
	Name:  "reprocess",
	Usage: "Reprocesses content in the repository.",
	Flags: append([]cli.Flag{
		cli.StringFlag{
			Name:  "path",
			Usage: "Target content path to be reprocessed. Format: <branch-name>:<content-path>. e.g 'draft:/'",
		},
		cli.BoolFlag{
			Name:  "skip-children",
			Usage: "Flag to skip processing of content children.",
		},
	}, common.FLAGS...),
	Action: func(c *cli.Context) error {

		ensurePathFlag(c)

		req := createReprocessRequest(c)

		var result ReprocessResponse

		status := common.RunTask(req, "Reprocessing", &result)

		switch status.State {
		case common.TASK_FINISHED:
			fmt.Fprintf(os.Stderr, "Updated %d content(s) with %d error(s)\n", len(result.UpdatedContent), len(result.Errors))
		case common.TASK_FAILED:
			fmt.Fprintf(os.Stderr, "Failed to reprocess: %s\n", status.Progress.Info)
		}
		fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(result))

		return nil
	},
}

func createReprocessRequest(c *cli.Context) *http.Request {
	body := new(bytes.Buffer)
	params := map[string]interface{}{
		"sourceBranchPath": c.String("path"),
	}

	if skipChildren := c.Bool("skip-children"); skipChildren {
		params["skipChildren"] = skipChildren
	}
	json.NewEncoder(body).Encode(params)

	return common.CreateRequest(c, "POST", "content/reprocessTask", body)
}

func ensurePathFlag(c *cli.Context) {
	var path = c.String("path")

	path = util.PromptUntilTrue(path, func(val *string, ind byte) string {
		if len(strings.TrimSpace(*val)) == 0 {
			switch ind {
			case 0:
				return "Enter target content path (<branch-name>:<content-path>): "
			default:
				return "Target content path can not be empty. Format: <branch-name>:<content-path>. e.g 'draft:/': "
			}
		} else {
			splitPathLen := len(strings.Split(*val, ":"))
			if splitPathLen != 2 {
				return fmt.Sprintf("Target content path '%s' must have the following format <branch-name>:<content-path>. e.g 'draft:/': ", *val)
			} else {
				return ""
			}
		}
	})

	c.Set("path", path)
}

type ReprocessResponse struct {
	Errors         []string `json:errors`
	UpdatedContent []string `json:updatedContent`
}
