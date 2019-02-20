package dump

import (
	"github.com/enonic/enonic-cli/internal/app/commands/common"
	"github.com/urfave/cli"
	"path/filepath"
	"io/ioutil"
	"github.com/enonic/enonic-cli/internal/app/commands/sandbox"
	"fmt"
	"os"
)

var List = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Usage:   "List available dumps",
	Flags:   append([]cli.Flag{}, common.FLAGS...),
	Action: func(c *cli.Context) error {

		dumps := listExistingDumpNames()
		for _, dump := range dumps {
			fmt.Fprintln(os.Stderr, dump)
		}

		return nil
	},
}

func listExistingDumpNames() []string {
	homePath := sandbox.GetActiveHomePath()
	dumpsDir := filepath.Join(homePath, "data", "dump")
	dumps, err := ioutil.ReadDir(dumpsDir)
	if err != nil {
		return []string{}
	}

	dumpNames := make([]string, len(dumps))
	for i, dump := range dumps {
		dumpNames[i] = dump.Name()
	}
	return dumpNames
}
