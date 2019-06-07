package remote

import (
	"fmt"
	"github.com/urfave/cli"
	"os"
)

var List = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Usage:   "List all known remotes.",
	Action: func(c *cli.Context) error {

		data := readRemotesData()
		for name, remote := range data.Remotes {
			if data.Active == name {
				fmt.Fprintf(os.Stdout, "* %s ( %s )\n", name, remote.Url)
			} else {
				fmt.Fprintf(os.Stdout, "  %s ( %s )\n", name, remote.Url)
			}
		}

		return nil
	},
}
