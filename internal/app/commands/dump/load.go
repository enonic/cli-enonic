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

			fmt.Fprint(os.Stderr, "Loading a dump (this may take few minutes)...")
			resp := common.SendRequest(req)

			var result LoadDumpResponse
			common.ParseResponse(resp, &result)
			fmt.Fprintf(os.Stderr, "Loaded %d repositories", len(result.Repositories))
		}

		return nil
	},
}

func acceptToDeleteExistingRepos() bool {
	fmt.Fprintln(os.Stderr, "WARNING: This will delete all existing repositories that also present in the system-dump.")
	answer := util.PromptUntilTrue("", func(val string, ind byte) string {
		if ind == 0 {
			return "Continue ? [Y/n] "
		} else {
			switch val {
			case "Y", "n":
				return ""
			default:
				return "Please type 'Y' for yes, or 'n' for no: "
			}
		}
	})
	return answer == "Y"
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
