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
	}, common.AUTH_FLAG, common.FORCE_FLAG),
	Action: func(c *cli.Context) error {

		ensureSnapshotOrBeforeFlag(c)

		req := createDeleteRequest(c)

		resp := common.SendRequest(req, "Deleting snapshot(s)")

		var result DeleteResult
		common.ParseResponse(resp, &result)
		fmt.Fprintf(os.Stderr, "%d Deleted\n", len(result.DeletedSnapshots))
		fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(result))

		return nil
	},
}

func ensureSnapshotOrBeforeFlag(c *cli.Context) {
	snapshot := c.String("snapshot")
	before := c.String("before")
	force := common.IsForceMode(c)

	if snapshot == "" && before == "" {
		if force {
			fmt.Fprintf(os.Stderr, "Either before or snapshot flag should not be empty in non-interactive mode.")
			os.Exit(1)
		}
		var val string
		choiceValidator := func(val interface{}) error {
			str := val.(string)
			if upper := strings.ToUpper(str); upper != "N" && upper != "D" {
				return errors.Errorf("'%s' is not a valid choice. Please use 'N' for Name or 'D' for Date: ", str)
			}
			return nil
		}
		util.PromptString("Select by [N]ame or by [D]ate:", val, "N", choiceValidator)
		switch val {
		case "N", "n":
			ensureSnapshotFlag(c)
		case "D", "d":
			ensureBeforeFlag(c)
		}
	}
}
func ensureBeforeFlag(c *cli.Context) {
	force := common.IsForceMode(c)
	label := fmt.Sprintf("Enter date in the format %s: ", time.Now().Format(DATE_FORMAT))
	dateValidator := func(val interface{}) error {
		str := val.(string)
		if len(strings.TrimSpace(str)) == 0 {
			if force {
				fmt.Fprintln(os.Stderr, "Before flag can not be empty in non-interactive mode.")
				os.Exit(1)
			}
			return fmt.Errorf("Before date can not be empty. Format %s: ", time.Now().Format(DATE_FORMAT))
		} else {
			if _, err := time.Parse(DATE_FORMAT, str); err != nil {
				if force {
					fmt.Fprintf(os.Stderr, "Not a valid date: %s", str)
					os.Exit(1)
				}
				return errors.New("Not a valid date: ")
			} else {
				return nil
			}
		}
	}
	before := util.PromptString(label, c.String("before"), label, dateValidator)

	c.Set("before", before)
}

func createDeleteRequest(c *cli.Context) *http.Request {
	body := new(bytes.Buffer)
	params := map[string]interface{}{}
	if name := c.String("snapshot"); name != "" {
		params["snapshotNames"] = []string{name}
	}
	if before := c.String("before"); before != "" {
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
	DeletedSnapshots []string `json:deletedSnapshots`
}
