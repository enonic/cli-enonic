package remote

import (
	"fmt"
	"github.com/urfave/cli"
	"os"
)

var Set = cli.Command{
	Name:  "set",
	Usage: "Set remote active to be used in all remote api queries.",
	Action: func(c *cli.Context) error {

		name := ensureExistingNameArg(c, false)
		data := readRemotesData()
		data.Active = name
		writeRemotesData(data)
		fmt.Fprintf(os.Stdout, "Remote '%s' set active", name)

		return nil
	},
}
