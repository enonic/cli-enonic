package project

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/commands/sandbox"
	"cli-enonic/internal/app/util"
	"fmt"
	"github.com/urfave/cli"
	"os"
)

var Sandbox = cli.Command{
	Name:      "sandbox",
	Aliases:   []string{"sbox", "sb"},
	Usage:     "Set the default sandbox associated with the current project",
	ArgsUsage: "<sandbox name>",
	Flags:     []cli.Flag{common.FORCE_FLAG},
	Action: func(c *cli.Context) error {

		ensureValidProjectFolder(".")
		pData := common.ReadProjectData(".")

		var sandboxName string
		if c.NArg() > 0 {
			sandboxName = c.Args().First()
		} else if pData.Sandbox != "" {
			fmt.Fprint(os.Stdout, "\n")
			projectName := common.ReadProjectName(".")
			question := fmt.Sprintf("\"%s\" is using sandbox \"%s\". Change the project's sandbox", projectName, pData.Sandbox)
			answer := util.PromptBool(question, false)
			fmt.Fprint(os.Stdout, "\n")
			if !answer {
				return nil
			}
		}

		sandbox, _ := sandbox.EnsureSandboxExists(c, sandbox.EnsureSandboxOptions{
			MinDistroVersion:   common.ReadProjectDistroVersion("."),
			Name:               sandboxName,
			NoBoxMessage:       "No sandboxes found, do you want to create one",
			SelectBoxMessage:   "Select sandbox to use as default for this project",
			ShowSuccessMessage: true,
			ShowCreateOption:   true,
			ExcludeSandboxes:   []string{pData.Sandbox},
		})
		if sandbox == nil {
			os.Exit(1)
		}
		common.WriteProjectData(&common.ProjectData{sandbox.Name}, ".")

		fmt.Fprintf(os.Stdout, "\nSandbox '%s' set as default.\n", sandbox.Name)

		return nil
	},
}
