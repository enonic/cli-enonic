package dump

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/enonic/cli-enonic/internal/app/commands/common"
	"github.com/enonic/cli-enonic/internal/app/util"
	"github.com/urfave/cli"
	"net/http"
	"os"
)

var Load = cli.Command{
	Name:  "load",
	Usage: "Import data from a dump.",
	Flags: append([]cli.Flag{
		cli.StringFlag{
			Name:  "d",
			Usage: "Dump name",
		},
		cli.StringFlag{
			Name:  "new-auth, na",
			Usage: "Authentication token for basic authentication in a new dump (user:password)",
		},
		cli.BoolFlag{
			Name:  "f, force",
			Usage: "assume “Yes” as answer to all prompts and run non-interactively",
		},
		cli.BoolFlag{
			Name:  "upgrade",
			Usage: "Upgrade the dump if necessary (default is false)",
		},
	}, common.FLAGS...),
	Action: func(c *cli.Context) error {

		if c.Bool("f") || util.PromptBool("WARNING: This will delete all existing repositories that also present in the system-dump. Continue ?", false) {

			name := ensureNameFlag(c.String("d"), false)

			req := createLoadRequest(c, name)
			var result LoadDumpResponse
			params := make(map[string]string)
			if newAuth := c.String("new-auth"); newAuth != "" {
				user, pass := common.EnsureAuth(newAuth)
				params["user"] = user
				params["pass"] = pass
			}
			status := common.RunTaskWithParams(req, "Loading dump", &result, params)

			switch status.State {
			case common.TASK_FINISHED:
				fmt.Fprintf(os.Stderr, "Loaded %d repositories in %s:\n", len(result.Repositories), util.TimeFromNow(status.StartTime))
			case common.TASK_FAILED:
				fmt.Fprintf(os.Stderr, "Failed to load dump: %s\n", status.Progress.Info)
			}
			fmt.Fprintln(os.Stderr, util.PrettyPrintJSON(result))
		}

		return nil
	},
}

func createLoadRequest(c *cli.Context, name string) *http.Request {
	body := new(bytes.Buffer)
	params := map[string]interface{}{
		"name": name,
	}

	if upgrade := c.Bool("upgrade"); upgrade {
		params["upgrade"] = upgrade
	}
	json.NewEncoder(body).Encode(params)

	return common.CreateRequest(c, "POST", "system/load", body)
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
