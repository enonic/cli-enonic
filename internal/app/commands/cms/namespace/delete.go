package namespace

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"fmt"
	"os"

	"github.com/urfave/cli"
)

var Delete = cli.Command{
	Name:      "delete",
	Usage:     "Delete a namespace and the whole schema subtree of the app",
	ArgsUsage: "<key>",
	Aliases:   []string{"del", "rm"},
	Flags:     append([]cli.Flag{common.FORCE_FLAG}, common.AUTH_AND_TLS_FLAGS...),
	Action: func(c *cli.Context) error {

		key := ensureKeyArg(c)
		force := common.IsForceMode(c)

		if !acceptToDeleteNamespace(key, force) {
			os.Exit(1)
		}

		req := common.CreateRequest(c, "DELETE", namespaceUrl(key), nil)
		res := common.SendRequest(c, req, fmt.Sprintf("Deleting namespace '%s'", key))

		var result interface{}
		parseResponse(res, &result)

		fmt.Fprintf(os.Stderr, "Namespace '%s' deleted\n", key)
		if result != nil {
			fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(result))
		}

		return nil
	},
}

func acceptToDeleteNamespace(key string, force bool) bool {
	return force || util.PromptBool(fmt.Sprintf("WARNING: This will delete the whole schema subtree of the app and can not be undone ! Do you still want to delete namespace '%s'", key), false)
}
