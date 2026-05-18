package dump

import (
	"bytes"
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli"
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
			Usage: "Load dump from archive. Only effective in compat mode (XP 7).",
		},
		common.FORCE_FLAG,
	}, append(common.AUTH_AND_TLS_FLAGS, common.COMPAT_FLAG)...),
	Action: func(c *cli.Context) error {

		util.Fatal(common.ValidateCompatFlag(c), "Invalid argument")

		force := common.IsForceMode(c)
		if force || util.PromptBool("WARNING: This will delete all existing repositories that also present in the system-dump. Continue", false) {

			name := ensureNameFlag(c, false, force)

			req := createLoadRequest(c, name)
			var result LoadDumpResponse

			var status *common.TaskStatus
			if common.IsCompatMode(c) {
				status = common.RunTask(c, req, "Loading dump. Check XP log for progress...", &result)
			} else {
				status = common.RunTaskWithSpinner(c, req, "Loading dump. Check XP log for progress", &result)
			}

			if status == nil {
				return nil
			}

			switch status.State {
			case common.TASK_FINISHED:
				fmt.Fprintf(os.Stderr, "Loaded %d repositories in %s:\n", len(result.Repositories), util.TimeFromNow(status.StartTime))
				fmt.Fprintln(os.Stderr, common.RESTART_ALL_RUNNING_INSTANCES_MSG)
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
	json.NewEncoder(body).Encode(buildLoadParams(c, name))
	return common.CreateRequest(c, "POST", "system/load", body)
}

func buildLoadParams(c *cli.Context, name string) map[string]interface{} {
	normalizedName, isZip := normalizeName(name)
	params := map[string]interface{}{
		"name": normalizedName,
	}

	if common.IsCompatMode(c) {
		if archive := c.Bool("archive") || isZip; archive {
			params["archive"] = archive
		}
	}

	if upgrade := c.Bool("upgrade"); upgrade {
		params["upgrade"] = upgrade
	}
	return params
}

func normalizeName(name string) (string, bool) {
	isZip := filepath.Ext(name) == ".zip"
	if isZip {
		return strings.TrimSuffix(name, ".zip"), true
	}
	return name, false
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
