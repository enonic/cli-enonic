package snapshot

import (
	"bytes"
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"encoding/json"
	"fmt"
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
	}, common.FLAGS...),
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

	if snapshot == "" && before == "" {
		var val string
		val = util.PromptUntilTrue(val, func(val *string, ind byte) string {
			if *val == "" && ind == 0 {
				return "Select by [N]ame or by [D]ate? "
			} else if upper := strings.ToUpper(*val); upper != "N" && upper != "D" {
				return `Please type "N" for Name or "D" for Date: `
			} else {
				return ""
			}
		})
		switch val {
		case "N", "n":
			ensureSnapshotFlag(c)
		case "D", "d":
			ensureBeforeFlag(c)
		}
	}
}
func ensureBeforeFlag(c *cli.Context) {

	before := util.PromptUntilTrue(c.String("before"), func(val *string, ind byte) string {
		if len(strings.TrimSpace(*val)) == 0 {
			switch ind {
			case 0:
				return fmt.Sprintf("Enter date in the format %s: ", time.Now().Format(DATE_FORMAT))
			default:
				return fmt.Sprintf("Before date can not be empty. Format %s: ", time.Now().Format(DATE_FORMAT))
			}
		} else {
			if _, err := time.Parse(DATE_FORMAT, *val); err != nil {
				return "Not a valid date: "
			} else {
				return ""
			}
		}
	})

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
