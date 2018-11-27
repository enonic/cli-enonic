package project

import (
	"github.com/urfave/cli"
	"github.com/enonic/xp-cli/internal/app/util"
	"os"
	"github.com/enonic/xp-cli/internal/app/commands/sandbox"
	"fmt"
)

func All() []cli.Command {
	return []cli.Command{
		Create,
		Sandbox,
		Clean,
		Build,
		Deploy,
		Install,
	}
}

type ProjectData struct {
	Sandbox string `toml:"sandbox"`
}

func isProject() bool {
	_, err := os.Stat(".enonic")
	return !os.IsNotExist(err)
}

func readProjectData() ProjectData {
	file := util.OpenOrCreateDataFile(".enonic", true)
	defer file.Close()

	var data ProjectData
	util.DecodeTomlFile(file, &data)
	return data
}

func writeProjectData(data ProjectData) {
	file := util.OpenOrCreateDataFile(".enonic", false)
	defer file.Close()

	util.EncodeTomlFile(file, data)
}

func ensureProjectFolder() ProjectData {
	data := readProjectData()
	if data.Sandbox == "" {
		if !util.YesNoPrompt("Current folder is not a project, do you to create one ?") {
			fmt.Fprintln(os.Stderr, "Aborted")
			os.Exit(1)
		}
		sbox := sandbox.EnsureSandboxNameExists(nil, "Select a sandbox to associate with project:")
		data.Sandbox = sbox.Name
		writeProjectData(data)
	}
	return data
}

