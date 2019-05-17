package main

import (
	"github.com/enonic/cli-enonic/internal/app/commands"
	"github.com/urfave/cli"
	survey "gopkg.in/AlecAivazis/survey.v1/core"
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

	app.CustomAppHelpTemplate = appHelp
	cli.CommandHelpTemplate = commandHelp
	cli.SubcommandHelpTemplate = subCommandHelp

	survey.ErrorIcon = ">>"
	survey.ErrorTemplate = `{{color "red"}}{{ ErrorIcon }}{{color "reset"}} {{color "white"}}{{.Error}}{{color "reset"}}
`

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

var subCommandHelp = `
{{if .Description}}{{.Description}}{{else}}{{.Usage}}{{end}}

USAGE:
   {{if .UsageText}}{{.UsageText}}{{else}}{{.HelpName}} command{{if .VisibleFlags}} [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{end}}

COMMANDS:{{range .VisibleCategories}}{{if .Name}}
   {{.Name}}:{{end}}{{range .VisibleCommands}}
     {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}
{{end}}{{if .VisibleFlags}}
OPTIONS:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{end}}
`

var commandHelp = `
{{if .Description}}{{.Description}}{{else}}{{.Usage}}{{end}}

USAGE:
   {{if .UsageText}}{{.UsageText}}{{else}}{{.HelpName}}{{if .VisibleFlags}} [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{end}}{{if .Category}}

CATEGORY:
   {{.Category}}{{end}}{{if .VisibleFlags}}

OPTIONS:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{end}}
`

var appHelp = `
{{.Name}} v.{{.Version}}
{{.Usage}}

USAGE:
   {{.HelpName}} {{if .VisibleFlags}}[global options]{{end}}{{if .Commands}} command [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{if .VisibleCommands}}

COMMANDS:{{range .VisibleCategories}}{{if .Name}}
   {{ "\n" }}{{.Name}}:{{end}}{{range .VisibleCommands}}
     {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}{{end}}{{if .VisibleFlags}}

GLOBAL OPTIONS:
{{range .VisibleFlags}}   {{.}}{{ "\n" }}{{end}}{{end}}

{{if .Copyright }}{{.Copyright}}{{end}}
`
