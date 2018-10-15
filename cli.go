package main

import (
	"log"
	"os"
	"github.com/urfave/cli"
	"enonic.com/xp-cli/commands"
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
