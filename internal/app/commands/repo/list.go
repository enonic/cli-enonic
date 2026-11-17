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

		result := listRepositories(c, "Loading")

		fmt.Fprintln(os.Stderr, "Done")
		fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(result))

		return nil
	},
}

func listRepositories(c *cli.Context, message string) *RepositoriesResult {
	path := "repo/list"
	if common.ServerApis(c) {
		path = common.ServerApiPath("repo")
	}
	res := common.SendRequest(c, common.CreateRequest(c, "GET", path, nil), message)

	var result RepositoriesResult
	common.ParseResponse(res, &result)
	return &result
}

type RepositoriesResult struct {
	Repositories []struct {
		Branches []string
		Id       string
	}
}
