package dump

import (
	"github.com/urfave/cli"
	"github.com/enonic/xp-cli/internal/app/commands/common"
	"os"
	"fmt"
	"net/http"
	"bytes"
	"encoding/json"
	"github.com/enonic/xp-cli/internal/app/util"
	"time"
)

var Load = cli.Command{
	Name:  "load",
	Usage: "Import data from a dump.",
	Flags: append([]cli.Flag{
		cli.StringFlag{
			Name:  "d",
			Usage: "Dump name.",
		},
		cli.BoolFlag{
			Name:  "y",
			Usage: "Automatic yes to prompts; assume “Yes” as answer to all prompts and run non-interactively.",
		},
		cli.BoolFlag{
			Name:  "upgrade",
			Usage: "Upgrade the dump if necessary (default is false)",
		},
	}, common.FLAGS...),
	Action: func(c *cli.Context) error {

		if c.Bool("y") || acceptToDeleteExistingRepos() {
			ensureNameFlag(c)

			req := createLoadRequest(c)
			var result LoadDumpResponse
			status := common.RunTask(c, req, "Loading dump...", &result)

			switch status.State {
			case common.TASK_FINISHED:
				fmt.Fprintf(os.Stderr, "Loaded %d repositories in %v", len(result.Repositories), time.Now().Sub(status.StartTime))
			case common.TASK_FAILED:
				fmt.Fprintf(os.Stderr, "Failed to load dump: %s", status.Progress.Info)
			}
		}

		return nil
	},
}

func acceptToDeleteExistingRepos() bool {
	return util.YesNoPrompt("WARNING: This will delete all existing repositories that also present in the system-dump. Continue ?")
}

func createLoadRequest(c *cli.Context) *http.Request {
	body := new(bytes.Buffer)
	params := map[string]interface{}{
		"name": c.String("d"),
	}

	if upgrade := c.Bool("upgrade"); upgrade {
		params["upgrade"] = upgrade
	}
	json.NewEncoder(body).Encode(params)

	return common.CreateRequest(c, "POST", "api/system/load", body)
}

type LoadDumpResponse struct {
	Repositories []struct {
		Repository string `json:repository`
		Versions struct {
			Errors []struct {
				message string `json:message`
			} `json:errors`
			Successful int64 `json:successful`
		} `json:versions`
		Branches []struct {
			Branch     string `json:branch`
			Successful int64  `json:successful`
			Errors []struct {
				message string `json:message`
			} `json:errors`
		} `json:branches`
	} `json:repositories`
}
