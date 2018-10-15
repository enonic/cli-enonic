package snapshot

import (
	"fmt"
	"github.com/urfave/cli"
)

var Restore = cli.Command{
	Name:  "restore",
	Usage: "Restores a snapshot of a previous state of the repository.",
	Action: func(c *cli.Context) error {
		fmt.Println("set xp version: ", c.Args().First())
		return nil
	},
}
