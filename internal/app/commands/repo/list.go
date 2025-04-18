package repo

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"fmt"
	"github.com/urfave/cli"
	"os"
)

var List = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Usage:   "List available repos",
	Flags:   append([]cli.Flag{common.FORCE_FLAG}, common.AUTH_AND_TLS_FLAGS...),
	Action: func(c *cli.Context) error {

		req := common.CreateRequest(c, "GET", "repo/list", nil)
		res := common.SendRequest(c, req, "Loading")

		var result RepositoriesResult
		common.ParseResponse(res, &result)

		fmt.Fprintln(os.Stderr, "Done")
		fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(result))

		return nil
	},
}

type RepositoriesResult struct {
	Repositories []struct {
		Branches []string
		Id       string
	}
}
