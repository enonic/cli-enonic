package project

import (
	"github.com/urfave/cli"
	"github.com/enonic/xp-cli/internal/app/commands/sandbox"
	"fmt"
	"github.com/enonic/xp-cli/internal/app/util"
)

var Deploy = cli.Command{
	Name:  "deploy",
	Usage: "Deploy current project to a sandbox",
	Action: func(c *cli.Context) error {

		projectData := readProjectData()
		noSandbox := projectData.Sandbox == ""
		if noSandbox || c.NArg() > 0 {
			sbox := sandbox.EnsureSandboxNameExists(c, "Select a sandbox to deploy to:")
			projectData.Sandbox = sbox.Name
			if noSandbox && util.YesNoPrompt(fmt.Sprintf("Project has no default sandbox, do you want to set '%s' as default ?", sbox.Name)) {
				writeProjectData(projectData)
			}
		}
		runGradleTask(projectData, "deploy", fmt.Sprintf("Deploying to %s...", projectData.Sandbox))

		return nil
	},
}
