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
			Usage: "Archive created dump.",
		},
	}, common.AUTH_FLAG, common.CRED_FILE_FLAG, common.FORCE_FLAG, common.CLIENT_KEY_FLAG, common.CLIENT_CERT_FLAG),
	Action: func(c *cli.Context) error {

		name := ensureNameFlag(c.String("d"), true, common.IsForceMode(c))

		req := createNewRequest(c, name)

		var result NewDumpResponse
		status := common.RunTask(c, req, "Creating dump", &result)

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
	params := map[string]interface{}{
		"name": name,
	}

	params["includeVersions"] = !c.Bool("skip-versions")

	if archive := c.Bool("archive"); archive {
		params["archive"] = archive
	}
	if maxAge := c.String("max-version-age"); maxAge != "" {
		params["maxAge"] = maxAge
	}
	if maxVersions := c.String("max-versions"); maxVersions != "" {
		params["maxVersions"] = maxVersions
	}
	json.NewEncoder(body).Encode(params)

	return common.CreateRequest(c, "POST", "system/dump", body)
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
