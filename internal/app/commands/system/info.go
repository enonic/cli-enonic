package system

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"fmt"
	"github.com/urfave/cli"
	"os"
)

var Info = cli.Command{
	Name:    "info",
	Aliases: []string{"i"},
	Usage:   "XP distribution info",
	Flags:   common.FLAGS,
	Action: func(c *cli.Context) error {

		req := common.CreateRequest(c, "GET", "http://localhost:2609/server", nil)
		res := common.SendRequest(req, "Loading")

		var result InfoResponse
		common.ParseResponse(res, &result)

		fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(result))

		return nil
	},
}

type InfoResponse struct {
	Version      string
	Installation string
	RunMode      string
	Build struct {
		Hash      string
		ShortHash string
		Branch    string
		Timestamp string
	}
}
