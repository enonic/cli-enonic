package dump

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/commands/sandbox"
	"cli-enonic/internal/app/util"
	"fmt"
	"github.com/urfave/cli"
	"io/ioutil"
	"os"
	"path/filepath"
)

var List = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Usage:   "List available dumps",
	Flags:   append([]cli.Flag{}, common.FLAGS...),
	Action: func(c *cli.Context) error {

		dumps := listExistingDumpNames()
		for _, dump := range dumps {
			fmt.Fprintln(os.Stdout, dump)
		}

		return nil
	},
}

func listExistingDumpNames() []string {
	homePath := sandbox.GetActiveHomePath()
	dumpsDir := filepath.Join(homePath, "data", "dump")
	dumps, err := ioutil.ReadDir(dumpsDir)
	if err != nil {
		util.Warn(err, "Error reading dumps folder:")
		return []string{}
	}

	dumpNames := make([]string, len(dumps))
	for i, dump := range dumps {
		dumpNames[i] = dump.Name()
	}
	return dumpNames
}
