package dump

import (
	"bytes"
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"encoding/json"
	"fmt"
	"github.com/urfave/cli"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var Load = cli.Command{
	Name:  "load",
	Usage: "Import data from a dump.",
	Flags: append([]cli.Flag{
		cli.StringFlag{
			Name:  "d",
			Usage: "Dump name",
		},
		cli.BoolFlag{
			Name:  "upgrade",
			Usage: "Upgrade the dump if necessary (default is false)",
		},
		cli.BoolFlag{
			Name:  "archive",
			Usage: "Load dump from archive.",
		},
	}, common.AUTH_FLAG, common.FORCE_FLAG),
	Action: func(c *cli.Context) error {

		force := common.IsForceMode(c)
		if force || util.PromptBool("WARNING: This will delete all existing repositories that also present in the system-dump. Continue", false) {

			name := ensureNameFlag(c.String("d"), false, force)

			req := createLoadRequest(c, name)
			var result LoadDumpResponse

			status := common.RunTask(req, "Loading dump", &result)

			switch status.State {
			case common.TASK_FINISHED:
				fmt.Fprintf(os.Stderr, "Loaded %d repositories in %s:\n", len(result.Repositories), util.TimeFromNow(status.StartTime))
			case common.TASK_FAILED:
				fmt.Fprintf(os.Stderr, "Failed to load dump: %s\n", status.Progress.Info)
			}
			fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(result))
		}

		return nil
	},
}

func createLoadRequest(c *cli.Context, name string) *http.Request {
	body := new(bytes.Buffer)
	normalizedName, isZip := normalizeName(name)
	params := map[string]interface{}{
		"name": normalizedName,
	}

	if archive := c.Bool("archive") || isZip; archive {
		// force archive param if zip is detected
		params["archive"] = archive
	}

	if upgrade := c.Bool("upgrade"); upgrade {
		params["upgrade"] = upgrade
	}
	json.NewEncoder(body).Encode(params)

	return common.CreateRequest(c, "POST", "system/load", body)
}

func normalizeName(name string) (string, bool) {
	isZip := filepath.Ext(name) == ".zip"
	var normalName string
	if isZip {
		normalName = strings.TrimSuffix(name, ".zip")
	} else {
		normalName = name
	}

	return normalName, isZip
}

type LoadDumpResponse struct {
	Repositories []struct {
		Repository string `json:"repository"`
		Versions   struct {
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
			Successful int64 `json:"successful"`
		} `json:"versions"`
		Branches []struct {
			Branch     string `json:"branch"`
			Successful int64  `json:"successful"`
			Errors     []struct {
				Message string `json:"message"`
			} `json:"errors"`
		} `json:"branches"`
	} `json:"repositories"`
}
