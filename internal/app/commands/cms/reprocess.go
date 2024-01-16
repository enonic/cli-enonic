package cms

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
	}, common.AUTH_FLAG, common.FORCE_FLAG),
	Action: func(c *cli.Context) error {

		var result ReprocessResponse
		requestLabel := "Reprocessing"

		ensurePathFlag(c)

		req := createReprocessRequest(c, "content/reprocessTask")
		res, err := common.SendRequestCustom(req, "", 3)
		util.Fatal(err, "Reprocess request error")

		var taskResult common.TaskResponse
		enonicErr, err := common.ParseResponseCustom(res, &taskResult)

		if enonicErr != nil {
			if enonicErr.Context.Authenticated {
				if user, pass, ok := res.Request.BasicAuth(); ok {
					// save the auth for future requests if any
					c.Set("auth", fmt.Sprintf("%s:%s", user, pass))
				}
			}

			if enonicErr.Status == http.StatusNotFound {
				// Async endpoint was not found, most likely XP version < 7.2 so trying synchronous endpoint
				newReq := createReprocessRequest(c, "content/reprocess")
				newRes := common.SendRequest(newReq, requestLabel)
				common.ParseResponse(newRes, &result)

				fmt.Fprintf(os.Stderr, "Updated %d content(s) with %d error(s)\n", len(result.UpdatedContent), len(result.Errors))
			} else {
				fmt.Fprintf(os.Stderr, "%d %s\n", enonicErr.Status, enonicErr.Message)
				os.Exit(1)
			}

		} else if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)

		} else {
			status := common.DisplayTaskProgress(taskResult.TaskId, requestLabel, &result)

			switch status.State {
			case common.TASK_FINISHED:
				fmt.Fprintf(os.Stderr, "Updated %d content(s) with %d error(s)\n", len(result.UpdatedContent), len(result.Errors))
			case common.TASK_FAILED:
				fmt.Fprintf(os.Stderr, "Failed to reprocess: %s\n", status.Progress.Info)
			}

		}
		fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(result))

		return nil
	},
}

func createReprocessRequest(c *cli.Context, url string) *http.Request {
	body := new(bytes.Buffer)
	params := map[string]interface{}{
		"sourceBranchPath": c.String("path"),
	}

	if skipChildren := c.Bool("skip-children"); skipChildren {
		params["skipChildren"] = skipChildren
	}
	json.NewEncoder(body).Encode(params)

	return common.CreateRequest(c, "POST", url, body)
}

func ensurePathFlag(c *cli.Context) {
	force := common.IsForceMode(c)
	pathValidator := func(val interface{}) error {
		str := val.(string)
		if len(strings.TrimSpace(str)) == 0 {
			if force {
				fmt.Fprintln(os.Stderr, "Target content path can not be empty in non-interactive mode.")
				os.Exit(1)
			}
			return errors.New("Target content path can not be empty. Format: <branch-name>:<content-path>. e.g 'draft:/': ")
		} else {
			splitPathLen := len(strings.Split(str, ":"))
			if splitPathLen != 2 {
				if force {
					fmt.Fprintf(os.Stderr, "Target content path '%s' must have the following format <branch-name>:<content-path>. e.g 'draft:/'\n", str)
					os.Exit(1)
				}
				return errors.Errorf("Target content path '%s' must have the following format <branch-name>:<content-path>. e.g 'draft:/': ", str)
			} else {
				return nil
			}
		}
	}

	path := util.PromptString("Enter target content path (<branch-name>:<content-path>)", c.String("path"), "", pathValidator)

	c.Set("path", path)
}

type ReprocessResponse struct {
	Errors         []string `json:"errors"`
	UpdatedContent []string `json:"updatedContent"`
}
