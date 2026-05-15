package dump

import (
	"bytes"
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/urfave/cli"
)

var Create = cli.Command{
	Name:  "create",
	Usage: "Export data from every repository.",
	Flags: append([]cli.Flag{
		cli.StringFlag{
			Name:  "d",
			Usage: "Dump name.",
		},
		cli.BoolFlag{
			Name:  "skip-versions",
			Usage: "Don't dump version-history, only current versions included.",
		},
		cli.StringFlag{
			Name:  "max-version-age",
			Usage: "Max age of versions to include, in days, in addition to current version.",
		},
		cli.StringFlag{
			Name:  "max-versions",
			Usage: "Max number of versions to dump in addition to current version.",
		},
		cli.BoolFlag{
			Name:  "archive",
			Usage: "Archive created dump. Only effective in compat mode (XP 7).",
		},
		common.FORCE_FLAG,
	}, append(common.AUTH_AND_TLS_FLAGS, common.COMPAT_FLAG)...),
	Action: func(c *cli.Context) error {

		util.Fatal(common.ValidateCompatFlag(c), "Invalid argument")

		name := ensureNameFlag(c, true, common.IsForceMode(c))

		req := createNewRequest(c, name)

		var result NewDumpResponse
		var status *common.TaskStatus
		if common.IsCompatMode(c) {
			status = common.RunTask(c, req, "Creating dump", &result)
		} else {
			status = common.RunTaskWithSpinner(c, req, "Creating dump", &result)
		}

		if status == nil {
			return nil
		}

		switch status.State {
		case common.TASK_FINISHED:
			fmt.Fprintf(os.Stderr, "Dumped %d repositories in %s:\n", len(result.Repositories), util.TimeFromNow(status.StartTime))
		case common.TASK_FAILED:
			fmt.Fprintf(os.Stderr, "Failed to dump repository: %s\n", status.Progress.Info)
		}
		fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(result))

		return nil
	},
}

func createNewRequest(c *cli.Context, name string) *http.Request {
	body := new(bytes.Buffer)
	json.NewEncoder(body).Encode(buildNewParams(c, name))
	return common.CreateRequest(c, "POST", "system/dump", body)
}

func buildNewParams(c *cli.Context, name string) map[string]interface{} {
	params := map[string]interface{}{
		"name":            name,
		"includeVersions": !c.Bool("skip-versions"),
	}

	if common.IsCompatMode(c) {
		if archive := c.Bool("archive"); archive {
			params["archive"] = archive
		}
	}
	if maxAge := c.String("max-version-age"); maxAge != "" {
		params["maxAge"] = maxAge
	}
	if maxVersions := c.String("max-versions"); maxVersions != "" {
		params["maxVersions"] = maxVersions
	}
	return params
}

type NewDumpResponse struct {
	Repositories []struct {
		RepositoryId string `json:"repositoryId"`
		Versions     int64  `json:"versions"`
		Branches     []struct {
			Branch     string `json:"branch"`
			Successful int64  `json:"successful"`
			Errors     []struct {
				Message string `json:"message"`
			} `json:"errors"`
		} `json:"branches"`
	} `json:"repositories"`
}
