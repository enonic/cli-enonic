package auditlog

import (
	"bytes"
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/senseyeio/duration"
	"github.com/urfave/cli"
	"net/http"
	"os"
	"strings"
)

var Cleanup = cli.Command{
	Name:  "cleanup",
	Usage: "Deletes records from audit log repository.",
	Flags: append([]cli.Flag{
		cli.StringFlag{
			Name:  "age",
			Usage: "Age of records to be removed. The format based on the ISO-8601 duration format PnDTnHnMn.nS with days considered to be exactly 24 hours.",
		},
	}, common.AUTH_FLAG, common.CRED_FILE_FLAG, common.FORCE_FLAG, common.CLIENT_KEY_FLAG, common.CLIENT_CERT_FLAG),
	Action: func(c *cli.Context) error {

		age := ensureAgeParam(c)

		req := createCleanRequest(c, age)

		var result CleanAuditlogResponse

		status := common.RunTask(c, req, "Cleaning auditlog", &result)

		switch status.State {
		case common.TASK_FINISHED:
			fmt.Fprintf(os.Stderr, "Cleaned auditlog in %s\n", util.TimeFromNow(status.StartTime))
		case common.TASK_FAILED:
			fmt.Fprintf(os.Stderr, "Failed to clean auditlog: %s\n", status.Progress.Info)
		}

		return nil
	},
}

func ensureAgeParam(c *cli.Context) string {
	force := common.IsForceMode(c)
	return util.PromptString("Enter age threshold in ISO-8601 based duration format (PnDTnHnMn.nS)", c.String("age"), "", func(val interface{}) error {
		str := val.(string)
		if len(strings.TrimSpace(str)) == 0 {
			if force {
				fmt.Fprintln(os.Stderr, "Age threshold can not be empty in non-interactive mode.")
				os.Exit(1)
			}
			return errors.New("Age threshold can not be empty")
		} else {
			if _, err := duration.ParseISO8601(str); err != nil {
				if force {
					fmt.Fprintf(os.Stderr, "Invalid age threshold format '%s'. Should be ISO-8601 based duration (PnDTnHnMn.nS)", str)
					os.Exit(1)
				}
				return errors.Errorf("Invalid age threshold format '%s'. Should be ISO-8601 based duration (PnDTnHnMn.nS): ", str)
			}
			return nil
		}
	})
}

func createCleanRequest(c *cli.Context, age string) *http.Request {
	body := new(bytes.Buffer)
	params := map[string]interface{}{
		"ageThreshold": age,
	}
	json.NewEncoder(body).Encode(params)

	return common.CreateRequest(c, "POST", "auditlog/cleanup", body)
}

type CleanAuditlogResponse struct {
}
