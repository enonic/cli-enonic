package repo

import (
	"bytes"
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"encoding/json"
	"fmt"
	"github.com/urfave/cli"
	"net/http"
	"os"
	"strings"
	"time"
)

var Reindex = cli.Command{
	Name:  "reindex",
	Usage: "Reindex content in search indices for the given repository and branches.",
	Flags: append([]cli.Flag{
		cli.StringFlag{
			Name:  "b",
			Usage: "A comma-separated list of branches to be reindexed.",
		},
		cli.StringFlag{
			Name:  "r",
			Usage: "The name of the repository to reindex.",
		},
		cli.BoolFlag{
			Name:  "i",
			Usage: "If true, the indices will be deleted before recreated.",
		},
	}, common.FLAGS...),
	Action: func(c *cli.Context) error {

		var result ReindexResponse
		requestLabel := "Reindexing"

		ensureRepoFlag(c)
		ensureBranchesFlag(c)

		req := createReindexRequest(c, "repo/index/reindexTask")
		res, err := common.SendRequestCustom(req, "", 3)
		util.Fatal(err, "Reindex request error")

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
				newReq := createReindexRequest(c, "repo/index/reindex")
				resp := common.SendRequest(newReq, requestLabel)
				common.ParseResponse(resp, &result)

				fmt.Fprintf(os.Stderr, "Reindexed %d node(s)\n", result.NumberReindexed)
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
				fmt.Fprintf(os.Stderr, "Reindexed %d node(s)\n", result.NumberReindexed)
			case common.TASK_FAILED:
				fmt.Fprintf(os.Stderr, "Failed to reindex: %s\n", status.Progress.Info)
			}

		}
		fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(result))

		return nil
	},
}

func ensureRepoFlag(c *cli.Context) {
	if c.String("r") == "" {

		var name string
		name = util.PromptUntilTrue(name, func(val *string, ind byte) string {
			if len(strings.TrimSpace(*val)) == 0 {
				switch ind {
				case 0:
					return "Enter repository name: "
				default:
					return "Repository name can not be empty: "
				}
			} else {
				return ""
			}
		})

		c.Set("r", name)
	}
}

func ensureBranchesFlag(c *cli.Context) {
	if c.String("b") == "" {
		var param string
		param = util.PromptUntilTrue(param, func(val *string, ind byte) string {
			if len(strings.TrimSpace(*val)) == 0 {
				switch ind {
				case 0:
					return "Enter comma separated list of branches: "
				default:
					return "List branches must contain at least one branch: "
				}
			} else {
				return ""
			}
		})

		branches := strings.Split(param, ",")
		for i, b := range branches {
			branches[i] = strings.TrimSpace(b)
		}

		c.Set("b", strings.Join(branches, ","))
	}
}

func createReindexRequest(c *cli.Context, url string) *http.Request {
	body := new(bytes.Buffer)
	params := map[string]interface{}{
		"repository": c.String("r"),
		"branches":   c.String("b"),
	}
	if init := c.Bool("i"); init {
		params["initialize"] = init
	}
	json.NewEncoder(body).Encode(params)

	return common.CreateRequest(c, "POST", url, body)
}

type ReindexResponse struct {
	RepositoryId    string    `json:repositoryId`
	Branches        []string  `json:branches`
	NumberReindexed uint32    `json:numberReindexed`
	StartTime       time.Time `json:startTime`
	EndTime         time.Time `json:endTime`
	Duration        string    `json:duration`
}
