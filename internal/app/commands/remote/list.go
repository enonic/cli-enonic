package remote

import (
	"github.com/urfave/cli"
	"fmt"
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
				fmt.Fprintf(os.Stderr, "* %s ( %s )\n", name, remote.Url)
			} else {
				fmt.Fprintf(os.Stderr, "  %s ( %s )\n", name, remote.Url)
			}
		}

		return nil
	},
}
