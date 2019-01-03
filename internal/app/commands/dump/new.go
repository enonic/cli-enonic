package dump

import (
	"github.com/urfave/cli"
	"github.com/enonic/xp-cli/internal/app/commands/common"
	"fmt"
	"os"
	"net/http"
	"bytes"
	"encoding/json"
	"github.com/enonic/xp-cli/internal/app/util"
)

var New = cli.Command{
	Name:  "new",
	Usage: "Export data from every repository.",
	Flags: append([]cli.Flag{
		cli.StringFlag{
			Name:  "d",
			Usage: "Dump name.",
		},
		cli.StringFlag{
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
	}, common.FLAGS...),
	Action: func(c *cli.Context) error {

		ensureNameFlag(c)

		req := createNewRequest(c)

		var result NewDumpResponse
		status := common.RunTask(req, "Creating dump", &result)

		switch status.State {
		case common.TASK_FINISHED:
			fmt.Fprintf(os.Stderr, "Dumped %d repositories in %s", len(result.Repositories), util.TimeFromNow(status.StartTime))
		case common.TASK_FAILED:
			fmt.Fprintf(os.Stderr, "Failed to dump repository: %s", status.Progress.Info)
		}

		return nil
	},
}

func createNewRequest(c *cli.Context) *http.Request {
	body := new(bytes.Buffer)
	params := map[string]interface{}{
		"name": c.String("d"),
	}

	if includeVersions := c.String("skip-versions"); includeVersions != "" {
		params["includeVersions"] = includeVersions
	}
	if maxAge := c.String("max-version-age"); maxAge != "" {
		params["maxAge"] = maxAge
	}
	if maxVersions := c.String("max-versions"); maxVersions != "" {
		params["maxVersions"] = maxVersions
	}
	json.NewEncoder(body).Encode(params)

	return common.CreateRequest(c, "POST", "api/system/dump", body)
}

type NewDumpResponse struct {
	Repositories []struct {
		RepositoryId string `json:repositoryId`
		Versions     int64  `json:versions`
		Branches []struct {
			Branch     string `json:branch`
			Successful int64  `json:successful`
			Errors []struct {
				message string `json:message`
			} `json:errors`
		} `json:branches`
	} `json:repositories`
}
