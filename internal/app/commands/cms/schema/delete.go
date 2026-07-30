package schema

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"fmt"
	"os"

	"github.com/urfave/cli"
)

var Delete = cli.Command{
	Name:    "delete",
	Usage:   "Delete a schema by key",
	Aliases: []string{"del", "rm"},
	Flags: append([]cli.Flag{
		KEY_FLAG,
		KIND_FLAG,
		common.FORCE_FLAG,
	}, common.AUTH_AND_TLS_FLAGS...),
	Action: func(c *cli.Context) error {

		k := ensureKindFlag(c)
		namespace, name := ensureKeyFlag(c, k)
		target := k.target(namespace, name)
		force := common.IsForceMode(c)

		if !force && !util.PromptBool(fmt.Sprintf("This can not be undone ! Do you still want to delete %s '%s'", k.name, target), false) {
			os.Exit(1)
		}

		req := common.CreateRequest(c, "DELETE", k.url(namespace, name), nil)
		res := common.SendRequest(c, req, fmt.Sprintf("Deleting %s '%s'", k.name, target))

		var result interface{}
		parseResponse(res, &result)

		fmt.Fprintf(os.Stderr, "Schema %s '%s' deleted\n", k.name, target)
		if result != nil {
			fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(result))
		}

		return nil
	},
}
