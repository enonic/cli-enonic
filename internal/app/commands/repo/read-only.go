package repo

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
	"strconv"
	"strings"
)

var ReadOnly = cli.Command{
	Name:  "readonly",
	Usage: "Toggle read-only mode for server or single repository",
	Flags: append([]cli.Flag{
		cli.StringFlag{
			Name:  "r",
			Usage: "Single repository to toggle read-only mode for",
		},
	}, common.AUTH_FLAG, common.CRED_FILE_FLAG, common.FORCE_FLAG),
	Action: func(c *cli.Context) error {

		readOnly := ensureReadOnlyArg(c)
		req := createReadOnlyRequest(c, readOnly)

		var access string
		if readOnly {
			access = "read only"
		} else {
			access = "read/write"
		}
		res := common.SendRequest(req, fmt.Sprintf("Setting access to %s", access))

		var result ReadOnlyResponse
		common.ParseResponse(res, &result)
		fmt.Fprintln(os.Stderr, "Done")
		fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(result))

		return nil
	},
}

func createReadOnlyRequest(c *cli.Context, readOnly bool) *http.Request {
	body := new(bytes.Buffer)
	params := map[string]interface{}{
		"requireClosedIndex": false,
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

	return common.CreateRequest(c, "POST", "repo/index/updateSettings", body)
}

func ensureReadOnlyArg(c *cli.Context) bool {
	argValue := c.Args().First()
	var readOnly bool

	validator := func(val interface{}) error {
		str := val.(string)
		if strings.TrimSpace(str) == "" {
			return errors.New("Enter 'T' for true or 'F' for false: ")
		} else {
			switch strings.ToLower(str) {
			case "t":
				readOnly = true
			case "f":
				readOnly = false
			default:
				var err error
				readOnly, err = strconv.ParseBool(str)
				if err != nil {
					return errors.New("Not a valid read only value. Enter 'T' for true or 'F' for false: ")
				}
			}
			return nil
		}
	}

	util.PromptString("Set read only [T]rue or [F]alse:", argValue, "", validator)

	return readOnly
}

type ReadOnlyResponse struct {
	UpdatedIndexes []string `json:"updatedIndexes"`
}
