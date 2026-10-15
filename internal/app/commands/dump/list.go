package dump

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"fmt"
	"github.com/urfave/cli"
	"os"
	"time"
)

var List = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Usage:   "List available dumps",
	Flags:   append([]cli.Flag{common.FORCE_FLAG}, append(common.AUTH_AND_TLS_FLAGS, common.COMPAT_FLAG)...),
	Action: func(c *cli.Context) error {

		util.Fatal(common.ValidateCompatFlag(c), "Invalid argument")

		if common.IsCompatMode(c) {
			fmt.Fprintln(os.Stderr, "Listing dumps is not supported in XP 7")
			os.Exit(1)
		}

		dumps := fetchDumpList(c)
		fmt.Fprintln(os.Stderr, "Done")
		fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(dumps))

		return nil
	},
}

func fetchDumpList(c *cli.Context) *DumpList {
	req := common.CreateRequest(c, "GET", "system/dump", nil)
	resp := common.SendRequest(c, req, "Loading dumps")

	var list DumpList
	common.ParseResponse(resp, &list)
	return &list
}

func listExistingDumpNames(c *cli.Context) []string {
	list := fetchDumpList(c)
	names := make([]string, len(list.Dumps))
	for i, d := range list.Dumps {
		names[i] = d.Name
	}
	return names
}

type DumpList struct {
	Dumps []DumpEntry `json:"dumps"`
}

type DumpEntry struct {
	Name         string    `json:"name"`
	Timestamp    time.Time `json:"timestamp"`
	XpVersion    string    `json:"xpVersion"`
	ModelVersion string    `json:"modelVersion"`
	Size         int64     `json:"size"`
}
