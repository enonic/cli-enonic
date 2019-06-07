package repo

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

		ensureRepoFlag(c)
		ensureBranchesFlag(c)

		req := createNewRequest(c)

		resp := common.SendRequest(req, "Reindexing")
		var result ReindexResponse
		common.ParseResponse(resp, &result)
		fmt.Fprintf(os.Stderr, "Done %d nodes\n", result.NumberReindexed)
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

func createNewRequest(c *cli.Context) *http.Request {
	body := new(bytes.Buffer)
	params := map[string]interface{}{
		"repository": c.String("r"),
		"branches":   c.String("b"),
	}
	if init := c.Bool("i"); init {
		params["initialize"] = init
	}
	json.NewEncoder(body).Encode(params)

	return common.CreateRequest(c, "POST", "repo/index/reindex", body)
}

type ReindexResponse struct {
	RepositoryId    string    `json:repositoryId`
	Branches        []string  `json:branches`
	NumberReindexed uint32    `json:numberReindexed`
	StartTime       time.Time `json:startTime`
	EndTime         time.Time `json:endTime`
	Duration        string    `json:duration`
}
