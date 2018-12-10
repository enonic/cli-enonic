package remote

import (
	"github.com/urfave/cli"
	"github.com/enonic/xp-cli/internal/app/util"
	"strings"
	"fmt"
	"os"
)

var Remove = cli.Command{
	Name:    "remove",
	Aliases: []string{"rm"},
	Usage:   "Remove a remote from list.",
	Action: func(c *cli.Context) error {

		name := ensureExistingNameArg(c, true)
		if name == DEFAULT {
			fmt.Fprintln(os.Stderr, "Default remote can not be deleted.")
			os.Exit(0)
		}
		data := readRemotesData()
		delete(data.Remotes, name)
		writeRemotesData(data)

		fmt.Fprintf(os.Stderr, "Deleted remote '%s'.\n", name)

		return nil
	},
}

func ensureExistingNameArg(c *cli.Context, allowActive bool) string {
	var name string
	if c.NArg() > 0 {
		name = c.Args().First()
	}
	remotes := readRemotesData()
	return util.PromptUntilTrue(name, func(val string, i byte) string {
		if len(strings.TrimSpace(val)) == 0 {
			if i == 0 {
				return "Enter the name of the remote: "
			} else {
				return "Remote name can not be empty: "
			}
		} else {
			if !allowActive && remotes.Active == val {
				return fmt.Sprintf("Remote '%s' is already set active: ", val)
			}
			if _, exists := getRemoteByName(val, remotes.Remotes); !exists {
				return fmt.Sprintf("Remote '%s' does not exist: ", val)
			}
			return ""
		}
	})
}
