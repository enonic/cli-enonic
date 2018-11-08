package snapshot

import (
	"github.com/urfave/cli"
	"fmt"
	"os"
	"net/http"
	"encoding/json"
	"bytes"
	"github.com/manifoldco/promptui"
	"time"
	"errors"
	"github.com/enonic/xp-cli/internal/app/util"
	"strings"
	"github.com/enonic/xp-cli/internal/app/commands/common"
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

		fmt.Fprint(os.Stderr, "Deleting snapshot(s)...")
		resp := common.SendRequest(req)

		var response DeleteResponse
		//debugResponse(resp)
		common.ParseResponse(resp, &response)
		fmt.Fprintf(os.Stderr, "%d Deleted\n", len(response.DeletedSnapshots))

		return nil
	},
}

func ensureSnapshotOrBeforeFlag(c *cli.Context) {
	snapshot := c.String("snapshot")
	before := c.String("before")

	if snapshot == "" && before == "" {
		var val string
		val = util.PromptUntilTrue(val, func(val string, ind byte) string {
			if val == "" && ind == 0 {
				return "Select by [N]ame or by [D]ate? "
			} else if upper := strings.ToUpper(val); upper != "N" && upper != "D" {
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
	if c.String("before") == "" {

		validate := func(input string) error {
			if _, err := time.Parse(DATE_FORMAT, input); err != nil {
				return errors.New("not a valid date")
			}
			return nil
		}

		prompt := promptui.Prompt{
			Label:    fmt.Sprintf("Enter date in the format %s: ", time.Now().Format(DATE_FORMAT)),
			Validate: validate,
		}

		result, err := prompt.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Prompt failed %v\n", err)
			return
		}

		c.Set("before", result)
	}
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

	return common.CreateRequest(c, "POST", "api/repo/snapshot/delete", body)
}

type DeleteResponse struct {
	DeletedSnapshots []string `json:deletedSnapshots`
}
