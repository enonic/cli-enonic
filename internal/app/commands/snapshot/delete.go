package snapshot

import (
	"bytes"
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"net/http"
	"os"
	"strings"
	"time"
)

const DATE_FORMAT = "2 Jan 06"

var Delete = cli.Command{
	Name:    "delete",
	Aliases: []string{"del"},
	Usage:   "Deletes snapshots, either before a given timestamp or by name.",
	Flags: append([]cli.Flag{
		cli.StringFlag{
			Name:  "before, b",
			Usage: "Delete snapshots before this timestamp",
		},
		cli.StringFlag{
			Name:  "snapshot, snap",
			Usage: "The name of the snapshot to delete",
		},
	}, common.AUTH_FLAG, common.CRED_FILE_FLAG, common.FORCE_FLAG),
	Action: func(c *cli.Context) error {

		snapshot, before := ensureSnapshotOrBeforeFlag(c)

		req := createDeleteRequest(c, snapshot, before)

		resp := common.SendRequest(req, "Deleting snapshot(s)")

		var result DeleteResult
		common.ParseResponse(resp, &result)
		fmt.Fprintf(os.Stderr, "%d Deleted\n", len(result.DeletedSnapshots))
		fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(result))

		return nil
	},
}

func ensureSnapshotOrBeforeFlag(c *cli.Context) (string, string) {
	snapshot := c.String("snapshot")
	before := c.String("before")
	force := common.IsForceMode(c)

	if snapshot == "" && before == "" {
		if force {
			fmt.Fprintf(os.Stderr, "Either before or snapshot flag should not be empty in non-interactive mode.")
			os.Exit(1)
		}
		choiceValidator := func(val interface{}) error {
			str := val.(string)
			if upper := strings.ToUpper(str); upper != "N" && upper != "D" {
				return errors.Errorf("'%s' is not a valid choice. Please use 'N' for Name or 'D' for Date: ", str)
			}
			return nil
		}
		val := util.PromptString("Select by [N]ame or by [D]ate", "", "N", choiceValidator)
		switch val {
		case "N", "n":
			snapshot = ensureSnapshotFlag(c)
		case "D", "d":
			before = ensureBeforeFlag(c)
		}
	}

	return snapshot, before
}
func ensureBeforeFlag(c *cli.Context) string {
	force := common.IsForceMode(c)
	timeFormat := time.Now().Format(DATE_FORMAT)
	dateValidator := func(val interface{}) error {
		str := val.(string)
		if len(strings.TrimSpace(str)) == 0 {
			if force {
				fmt.Fprintln(os.Stderr, "Before flag can not be empty in non-interactive mode.")
				os.Exit(1)
			}
			return fmt.Errorf("Before date can not be empty. Format %s: ", timeFormat)
		} else {
			if _, err := time.Parse(DATE_FORMAT, str); err != nil {
				if force {
					fmt.Fprintf(os.Stderr, "Not a valid date: %s", str)
					os.Exit(1)
				}
				return errors.New("Not a valid date.")
			} else {
				return nil
			}
		}
	}
	label := fmt.Sprintf("Delete snapshots before the date (format: %s)", timeFormat)
	before := util.PromptString(label, c.String("before"), time.Now().AddDate(0, 0, -7).Format(DATE_FORMAT), dateValidator)

	return before
}

func createDeleteRequest(c *cli.Context, snapshot, before string) *http.Request {
	body := new(bytes.Buffer)
	params := map[string]interface{}{}
	if snapshot != "" {
		params["snapshotNames"] = []string{snapshot}
	}
	if before != "" {
		parsedTime, err := time.Parse(DATE_FORMAT, before)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Parsing failed %v\n", before)
			os.Exit(1)
		}
		params["before"] = parsedTime
	}
	json.NewEncoder(body).Encode(params)

	return common.CreateRequest(c, "POST", "repo/snapshot/delete", body)
}

type DeleteResult struct {
	DeletedSnapshots []string `json:"deletedSnapshots"`
}
