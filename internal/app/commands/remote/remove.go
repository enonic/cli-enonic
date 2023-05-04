package remote

import (
	"cli-enonic/internal/app/util"
	"fmt"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"os"
	"strings"
)

var Remove = cli.Command{
	Name:    "remove",
	Aliases: []string{"rm"},
	Usage:   "Remove a remote from list.",
	Action: func(c *cli.Context) error {

		name := ensureExistingNameArg(c, true)
		if name == DEFAULT_REMOTE_NAME {
			fmt.Fprintln(os.Stderr, "Default remote can not be deleted.")
			os.Exit(1)
		}
		data := readRemotesData()
		delete(data.Remotes, name)
		if data.Active == name {
			data.Active = DEFAULT_REMOTE_NAME
		}
		writeRemotesData(data)

		fmt.Fprintf(os.Stdout, "Deleted remote '%s'.\n", name)

		return nil
	},
}

func ensureExistingNameArg(c *cli.Context, allowActive bool) string {
	var name string
	if c.NArg() > 0 {
		name = c.Args().First()
	}
	remotes := readRemotesData()
	validator := func(val interface{}) error {
		str := val.(string)
		if strings.TrimSpace(str) == "" {
			return errors.New("Remote name can not be empty: ")
		} else {
			if !allowActive && remotes.Active == str {
				return errors.Errorf("Remote '%s' is already set active: ", str)
			}
			if _, exists := getRemoteByName(str, remotes.Remotes); !exists {
				return errors.Errorf("Remote '%s' does not exist: ", str)
			}
		}
		return nil
	}

	return util.PromptString("Enter the name of the remote", name, "", validator)
}
