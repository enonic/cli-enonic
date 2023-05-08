package util

import (
	"github.com/urfave/cli"
	"gopkg.in/AlecAivazis/survey.v1"
	surveyCore "gopkg.in/AlecAivazis/survey.v1/core"
	"io"
)

func SetupTemplates(app *cli.App, funcMap map[string]interface{}) {
	app.CustomAppHelpTemplate = appHelp
	cli.CommandHelpTemplate = commandHelp
	cli.SubcommandHelpTemplate = subCommandHelp
	survey.ConfirmQuestionTemplate = confirmQuestionTemplate
	survey.InputQuestionTemplate = inputQuestionTemplate
	surveyCore.ErrorIcon = ">>"
	surveyCore.ErrorTemplate = `{{color "red"}}{{ ErrorIcon }}{{color "reset"}} {{color "white"}}{{.Error}}{{color "reset"}}
`
	var originalHelpPrinter = cli.HelpPrinterCustom
	cli.HelpPrinterCustom = func(out io.Writer, templ string, data interface{}, customFunc map[string]interface{}) {
		if customFunc != nil {
			for key, value := range customFunc {
				funcMap[key] = value
			}
		}
		originalHelpPrinter(out, templ, data, funcMap)
	}

	cli.HelpPrinter = func(out io.Writer, templ string, data interface{}) {
		cli.HelpPrinterCustom(out, templ, data, funcMap)
	}
}

var versionMessage = `{{with $msg := versionMessage}}{{if ne $msg ""}}
{{color "magenta+b"}}{{$msg}}{{color "default"}}
{{end}}{{end}}`

var confirmQuestionTemplate = `
{{- if .ShowHelp }}{{- color "cyan"}}{{ HelpIcon }} {{ .Help }}{{color "reset"}}{{"\n"}}{{end}}
{{- color "green+hb"}}{{ QuestionIcon }} {{color "reset"}}
{{- color "default+hb"}}{{ .Message }}? {{color "reset"}}
{{- if .Answer}}
  {{- color "cyan"}}{{.Answer}}{{color "reset"}}{{"\n"}}
{{- else }}
  {{- if and .Help (not .ShowHelp)}}{{color "cyan"}}[{{ HelpInputRune }} for help]{{color "reset"}} {{end}}
  {{- color "white"}}{{if .Default}}[Y/n] {{else}}[y/N] {{end}}{{color "reset"}}
{{- end}}`

var inputQuestionTemplate = `
{{- if .ShowHelp }}{{- color "cyan"}}{{ HelpIcon }} {{ .Help }}{{color "reset"}}{{"\n"}}{{end}}
{{- color "green+hb"}}{{ QuestionIcon }} {{color "reset"}}
{{- color "default+hb"}}{{ .Message }}: {{color "reset"}}
{{- if .ShowAnswer}}
  {{- color "cyan"}}{{.Answer}}{{color "reset"}}{{"\n"}}
{{- else }}
  {{- if and .Help (not .ShowHelp)}}{{color "cyan"}}[{{ HelpInputRune }} for help]{{color "reset"}} {{end}}
  {{- if .Default}}{{color "white"}}[{{.Default}}] {{color "reset"}}{{end}}
{{- end}}`

var subCommandHelp = `
{{if .Description}}{{.Description}}{{else}}{{.Usage}}{{end}}
USAGE:
   {{if .UsageText}}{{.UsageText}}{{else}}{{.HelpName}} [command]{{if .VisibleFlags}} [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{end}}

COMMANDS:{{range .VisibleCategories}}{{if .Name}}
   {{.Name}}:{{end}}{{range .VisibleCommands}}
     {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}
{{end}}{{if .VisibleFlags}}
OPTIONS:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{end}}` + versionMessage + `
`

var commandHelp = `
{{if .Description}}{{.Description}}{{else}}{{.Usage}}{{end}}
USAGE:
   {{if .UsageText}}{{.UsageText}}{{else}}{{.HelpName}}{{if .VisibleFlags}} [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{end}}{{if .Category}}

CATEGORY:
   {{.Category}}{{end}}{{if .VisibleFlags}}

OPTIONS:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{end}}` + versionMessage + `
`

var appHelp = `
Enonic CLI {{.Version}}
{{.Usage}}
USAGE:
   {{.HelpName}} {{if .VisibleFlags}}[global options]{{end}}{{if .Commands}} [command] [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{if .VisibleCommands}}

COMMANDS:{{range .VisibleCategories}}{{if .Name}}
   {{ "\n" }}{{.Name}}:{{end}}{{range .VisibleCommands}}
     {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}{{end}}{{if .VisibleFlags}}

GLOBAL OPTIONS:
{{range .VisibleFlags}}   {{.}}{{ "\n" }}{{end}}{{end}}` + versionMessage + `
{{if .Copyright }}{{.Copyright}}{{end}}`
