package main

import (
	"github.com/enonic/cli-enonic/internal/app/commands"
	"github.com/enonic/cli-enonic/internal/app/util"
	"github.com/urfave/cli"
	"log"
	"os"
)

// set by goreleaser
// https://goreleaser.com/environment/
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	app := cli.NewApp()
	app.Name = "Enonic CLI"
	app.Version = version
	app.Usage = "Manage XP instances, home folders and projects"
	app.EnableBashCompletion = true
	app.Commands = commands.All()

	util.SetupTemplates(app)

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
