package snapshot

import (
	"fmt"
	"github.com/urfave/cli"
	"net/http"
	"enonic.com/xp-cli/commands/util"
	"log"
)

var List = cli.Command{
	Name:  "ls",
	Usage: "Returns a list of existing snapshots with name and status.",
	Action: func(c *cli.Context) error {
		fmt.Println("Snapshots:")
		config := util.GetConfig()
		fmt.Println("Creating request to: ", config.GetUrl())
		resp, err := http.Get(config.GetUrl())
		if err != nil {
			log.Fatal("Can't make a request: ", err)
		}
		fmt.Println("Request response: ", resp)

		return nil
	},
}
