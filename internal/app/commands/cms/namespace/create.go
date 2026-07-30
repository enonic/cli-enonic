package namespace

import (
	"bytes"
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"encoding/json"
	"fmt"
	"os"

	"github.com/urfave/cli"
)

var Create = cli.Command{
	Name:      "create",
	Usage:     "Create a namespace",
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

		body := new(bytes.Buffer)
		params := map[string]interface{}{
			"key": key,
		}
		if desc := c.String("desc"); desc != "" {
			params["description"] = desc
		}
		json.NewEncoder(body).Encode(params)

		req := common.CreateRequest(c, "POST", "/server:schema/namespaces", body)
		res := common.SendRequest(c, req, fmt.Sprintf("Creating namespace '%s'", key))

		var result interface{}
		parseResponse(res, &result)

		fmt.Fprintf(os.Stderr, "Namespace '%s' created\n", key)
		if result != nil {
			fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(result))
		}

		return nil
	},
}
