package cms

import (
	"github.com/urfave/cli"
	"github.com/enonic/xp-cli/internal/app/commands/common"
	"fmt"
	"os"
	"net/http"
	"bytes"
	"encoding/json"
	"github.com/enonic/xp-cli/internal/app/util"
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

		fmt.Fprint(os.Stderr, "Reprocessing...")
		res := common.SendRequest(req)

		var result ReprocessResponse
		common.ParseResponse(res, &result)
		fmt.Fprintf(os.Stderr, "Updated %d content(s) with %d error(s)\n", len(result.UpdatedContent), len(result.Errors))
		fmt.Fprintln(os.Stderr, util.PrettyPrintJSON(result))

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

	return common.CreateRequest(c, "POST", "content/reprocess", body)
}

func ensurePathFlag(c *cli.Context) {
	var path = c.String("path")

	path = util.PromptUntilTrue(path, func(val string, ind byte) string {
		if len(strings.TrimSpace(val)) == 0 {
			switch ind {
			case 0:
				return "Enter target content path (<branch-name>:<content-path>): "
			default:
				return "Target content path can not be empty. Format: <branch-name>:<content-path>. e.g 'draft:/': "
			}
		} else {
			splitPathLen := len(strings.Split(val, ":"))
			if splitPathLen != 2 {
				return fmt.Sprintf("Target content path '%s' must have the following format <branch-name>:<content-path>. e.g 'draft:/': ", val)
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
