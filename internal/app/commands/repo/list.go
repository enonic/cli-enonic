package repo

import (
	"fmt"
	"github.com/enonic/cli-enonic/internal/app/commands/common"
	"github.com/enonic/cli-enonic/internal/app/util"
	"github.com/urfave/cli"
	"os"
)

var List = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Usage:   "List available repos",
	Flags:   append([]cli.Flag{}, common.AUTH_FLAG, common.FORCE_FLAG),
	Action: func(c *cli.Context) error {

		req := common.CreateRequest(c, "GET", "repo/list", nil)
		res := common.SendRequest(req, "Loading")

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
