package repo

import (
	"github.com/enonic/enonic-cli/internal/app/commands/common"
	"github.com/urfave/cli"
	"github.com/enonic/enonic-cli/internal/app/util"
	"fmt"
	"os"
)

var List = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Usage:   "List available repos",
	Flags:   append([]cli.Flag{}, common.FLAGS...),
	Action: func(c *cli.Context) error {

		req := common.CreateRequest(c, "GET", "repo/index/listRepositories", nil)
		res := common.SendRequest(req, "Loading")

		var result RepositoriesResult
		common.ParseResponse(res, &result)

		fmt.Fprintln(os.Stderr, util.PrettyPrintJSON(result))

		return nil
	},
}

type RepositoriesResult struct {
	Repositories []struct {
		Branches []string
		Id       string
	}
}
