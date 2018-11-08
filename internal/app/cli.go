package app

import (
	"log"
	"os"
	"github.com/urfave/cli"
	"github.com/enonic/xp-cli/internal/app/commands"
)

func main() {
	app := cli.NewApp()
	app.Name = "Enonic CLI"
	app.Usage = "Manage XP instances, home folders and projects"
	app.EnableBashCompletion = true
	app.Commands = commands.All()

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
