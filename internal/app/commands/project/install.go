package project

import (
	"github.com/urfave/cli"
	"github.com/enonic/enonic-cli/internal/app/commands/app"
	"io/ioutil"
	"github.com/enonic/enonic-cli/internal/app/util"
	"path/filepath"
	"os"
	"fmt"
	"github.com/enonic/enonic-cli/internal/app/commands/common"
)

var Install = cli.Command{
	Name:    "install",
	Aliases: []string{"i"},
	Usage:   "Build current project and install it to Enonic XP",
	Flags:   append([]cli.Flag{}, common.FLAGS...),
	Action: func(c *cli.Context) error {

		buildProject(c)
		jarPath := findJarPath()
		app.InstallFromFile(c, jarPath)

		return nil
	},
}

func findJarPath() string {
	libsDir := filepath.Join("build", "libs")
	infos, err := ioutil.ReadDir(libsDir)
	util.Fatal(err, fmt.Sprintf("Could not read '%s' folder", libsDir))

	for _, info := range infos {
		if filepath.Ext(info.Name()) == ".jar" {
			return filepath.Join(libsDir, info.Name())
		}
	}

	fmt.Fprintln(os.Stderr, "Could not find file to install")
	os.Exit(1)
	return ""
}
