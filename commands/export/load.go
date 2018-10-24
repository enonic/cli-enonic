package export

import (
	"github.com/urfave/cli"
	"enonic.com/xp-cli/commands/common"
	"os"
	"fmt"
	"net/http"
	"bytes"
	"encoding/json"
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

		ensureNameFlag(c)

		req := createLoadRequest(c)

		fmt.Fprint(os.Stderr, "Loading a dump (this may take few minutes)...")
		resp := common.SendRequest(req)

		var result LoadDumpResponse
		common.ParseResponse(resp, &result)
		fmt.Fprintf(os.Stderr, "Loaded %d repositories", len(result.Repositories))

		return nil
	},
}

func createLoadRequest(c *cli.Context) *http.Request {
	body := new(bytes.Buffer)
	params := map[string]interface{}{
		"name": c.String("d"),
	}

	if autoYes := c.Bool("y"); autoYes {
		params["y"] = autoYes
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
