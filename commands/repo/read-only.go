package repo

import (
	"github.com/urfave/cli"
	"enonic.com/xp-cli/commands/common"
	"enonic.com/xp-cli/util"
	"strconv"
	"encoding/json"
	"bytes"
	"net/http"
	"fmt"
	"os"
)

var ReadOnly = cli.Command{
	Name:  "readonly",
	Usage: "Toggle read-only mode for server or single repository",
	Flags: append([]cli.Flag{
		cli.StringFlag{
			Name:  "r",
			Usage: "Single repository to toggle read-only mode for",
		},
	}, common.FLAGS...),
	Action: func(c *cli.Context) error {

		readOnly := ensureReadOnlyArg(c)
		req := createReadOnlyRequest(c, readOnly)

		if readOnly {
			fmt.Fprint(os.Stderr, "Setting read only access...")
		} else {
			fmt.Fprint(os.Stderr, "Setting read/write access...")
		}
		res := common.SendRequest(req)

		var result ReadOnlyResponse
		common.ParseResponse(res, &result)
		fmt.Fprintln(os.Stderr, "Done")

		return nil
	},
}

func createReadOnlyRequest(c *cli.Context, readOnly bool) *http.Request {
	body := new(bytes.Buffer)
	params := map[string]interface{}{
		"requireClosedIndex": true,
		"settings": map[string]interface{}{
			"index": map[string]interface{}{
				"blocks.write": readOnly,
			},
		},
	}
	if repo := c.String("r"); repo != "" {
		params["repositoryId"] = repo
	}
	json.NewEncoder(body).Encode(params)

	return common.CreateRequest(c, "POST", "api/repo/index/updateSettings", body)
}

func ensureReadOnlyArg(c *cli.Context) bool {
	argValue := c.Args().First()
	var readOnly bool

	util.PromptUntilTrue(argValue, func(val string, ind byte) string {
		if val == "" {
			switch ind {
			case 0:
				return "Set read only [T]rue or [F]alse: "
			default:
				return "Enter 'T' for true or 'F' for false: "
			}
		} else {
			switch val {
			case "T", "t":
				readOnly = true
			case "F", "f":
				readOnly = false
			default:
				var err error
				readOnly, err = strconv.ParseBool(val)
				if err != nil {
					return "Not a valid read only value. Enter 'T' for true or 'F' for false: "
				}
			}
			return ""
		}
	})
	return readOnly
}

type ReadOnlyResponse struct {
	UpdatedIndexes [] string `json:updatedIndexes`
}
