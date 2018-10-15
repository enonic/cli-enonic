package snapshot

import (
	"github.com/urfave/cli"
	"fmt"
)

var Delete = cli.Command{
	Name:  "delete",
	Usage: "Deletes snapshots, either before a given timestamp or by name.",
	Action: func(c *cli.Context) error {
		fmt.Println("list xp installations: ", c.Args().First())
		return nil
	},
}
