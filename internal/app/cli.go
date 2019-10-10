package main

import (
	"github.com/enonic/cli-enonic/internal/app/commands"
	"github.com/enonic/cli-enonic/internal/app/commands/common"
	"github.com/enonic/cli-enonic/internal/app/util"
	"github.com/mgutz/ansi"
	"github.com/urfave/cli"
	"log"
	"os"
	"text/template"
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
	app.Commands = commands.All()

	funcMap := template.FuncMap{
		"color":          ansi.ColorCode,
		"versionMessage": common.ProduceCheckVersionFunction(app.Version),
	}

	util.SetupTemplates(app, funcMap)

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
