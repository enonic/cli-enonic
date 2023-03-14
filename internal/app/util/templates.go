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
	survey.SelectQuestionTemplate = selectTemplate
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

var selectTemplate = `
{{- if .ShowHelp }}{{- color "cyan"}}{{ HelpIcon }} {{ .Help }}{{color "reset"}}{{"\n"}}{{end}}
{{- color "green+hb"}}{{ QuestionIcon }} {{color "reset"}}
{{- color "default+hb"}}{{ .Message }}{{ .FilterMessage }}{{color "reset"}}
{{- if .ShowAnswer}}{{color "cyan"}} {{.Answer}}{{color "reset"}}{{"\n"}}
{{- else}}
  {{- "  "}}{{- color "cyan"}}[Use arrows to move, type to filter{{- if and .Help (not .ShowHelp)}}, {{ HelpInputRune }} for more help{{end}}]{{color "reset"}}
  {{- "\n"}}
  {{- range $ix, $choice := .PageEntries}}
    {{- if eq $ix $.SelectedIndex}}{{color "cyan+b"}}{{ SelectFocusIcon }} {{else}}{{color "default"}}  {{end}}
    {{- $choice}}
    {{- color "reset"}}{{"\n"}}
  {{- end}}
{{- end}}`

var subCommandHelp = `
{{if .Description}}{{.Description}}{{else}}{{.Usage}}{{end}}
{{with $msg := versionMessage}}{{if ne $msg ""}}
{{color "cyan+b"}}{{$msg}}{{color "default"}}
{{end}}{{end}}
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
{{with $msg := versionMessage}}{{if ne $msg ""}}
{{color "cyan+b"}}{{$msg}}{{color "default"}}
{{end}}{{end}}
USAGE:
   {{if .UsageText}}{{.UsageText}}{{else}}{{.HelpName}}{{if .VisibleFlags}} [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{end}}{{if .Category}}

CATEGORY:
   {{.Category}}{{end}}{{if .VisibleFlags}}

OPTIONS:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{end}}
`

var appHelp = `
{{.Name}} v{{.Version}}
{{.Usage}}
{{with $msg := versionMessage}}{{if ne $msg ""}}
{{color "cyan+b"}}{{$msg}}{{color "default"}}
{{end}}{{end}}
USAGE:
   {{.HelpName}} {{if .VisibleFlags}}[global options]{{end}}{{if .Commands}} command [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{if .VisibleCommands}}

COMMANDS:{{range .VisibleCategories}}{{if .Name}}
   {{ "\n" }}{{.Name}}:{{end}}{{range .VisibleCommands}}
     {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}{{end}}{{if .VisibleFlags}}

GLOBAL OPTIONS:
{{range .VisibleFlags}}   {{.}}{{ "\n" }}{{end}}{{end}}

{{if .Copyright }}{{.Copyright}}{{end}}
`
