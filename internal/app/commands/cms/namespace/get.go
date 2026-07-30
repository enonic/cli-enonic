package namespace

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"fmt"
	"os"

	"github.com/urfave/cli"
)

var Get = cli.Command{
	Name:      "get",
	Usage:     "Get a namespace",
	ArgsUsage: "<key>",
	Flags:     append([]cli.Flag{common.FORCE_FLAG}, common.AUTH_AND_TLS_FLAGS...),
	Action: func(c *cli.Context) error {

		key := ensureKeyArg(c)

		req := common.CreateRequest(c, "GET", namespaceUrl(key), nil)
		res := common.SendRequest(c, req, "Loading")

		var result interface{}
		parseResponse(res, &result)

		fmt.Fprintln(os.Stderr, "Done")
		fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(result))

		return nil
	},
}
