package snapshot

import (
	"fmt"
	"github.com/urfave/cli"
)

var New = cli.Command{
	Name:  "new",
	Usage: "Stores a snapshot of the current state of the repository.",
	Action: func(c *cli.Context) error {
		fmt.Println("stop xp: ", c.Args().First())
		return nil
	},
}
