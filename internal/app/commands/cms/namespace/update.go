package namespace

import (
	"bytes"
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var Update = cli.Command{
	Name:      "update",
	Usage:     "Update a namespace",
	ArgsUsage: "<key>",
	Flags: append([]cli.Flag{
		cli.StringFlag{
			Name:  "desc",
			Usage: "Namespace description",
		},
		common.FORCE_FLAG,
	}, common.AUTH_AND_TLS_FLAGS...),
	Action: func(c *cli.Context) error {

		key := ensureKeyArg(c)
		desc := ensureDescFlag(c)

		body := new(bytes.Buffer)
		params := map[string]interface{}{
			"description": desc,
		}
		json.NewEncoder(body).Encode(params)

		req := common.CreateRequest(c, "PUT", namespaceUrl(key), body)
		res := common.SendRequest(c, req, fmt.Sprintf("Updating namespace '%s'", key))

		var result interface{}
		parseResponse(res, &result)

		fmt.Fprintf(os.Stderr, "Namespace '%s' updated\n", key)
		if result != nil {
			fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(result))
		}

		return nil
	},
}

func ensureDescFlag(c *cli.Context) string {
	descValidator := func(val interface{}) error {
		str := val.(string)
		if len(strings.TrimSpace(str)) == 0 {
			if common.IsForceMode(c) {
				fmt.Fprintln(os.Stderr, "Namespace description can not be empty in non-interactive mode.")
				os.Exit(1)
			}
			return errors.New("Namespace description can not be empty: ")
		}
		return nil
	}

	return util.PromptString("Enter namespace description", c.String("desc"), "", descValidator)
}
