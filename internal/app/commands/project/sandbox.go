package project

import (
	"github.com/urfave/cli"
	"github.com/enonic/xp-cli/internal/app/commands/sandbox"
	"github.com/enonic/xp-cli/internal/app/util"
	"fmt"
	"os"
)

var Sandbox = cli.Command{
	Name:    "sandbox",
	Aliases: []string{"sbox", "sb"},
	Usage:   "Set the default sandbox associated with the current project",
	Action: func(c *cli.Context) error {

		sandbox := sandbox.EnsureSandboxNameExists(c, "Select sandbox to attach to:")
		data := readProjectData()
		data.Sandbox = sandbox.Name
		writeProjectData(data)

		fmt.Fprintf(os.Stderr, "Attached current project to sandbox '%s'", sandbox.Name)

		return nil
	},
}

type ProjectData struct {
	Sandbox string `toml:"sandbox"`
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
