package sandbox

import (
	"github.com/urfave/cli"
	"path/filepath"
	"io/ioutil"
	"fmt"
	"os"
	"github.com/enonic/xp-cli/internal/app/util"
)

var List = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Usage:   "List all sandboxes",
	Action: func(c *cli.Context) error {
		data := readSandboxesData()

		for _, b := range ListSandboxes() {
			if data.Running == b {
				fmt.Fprintf(os.Stderr, "* %s\n", b)
			} else {
				fmt.Fprintf(os.Stderr, "  %s\n", b)
			}
		}
		return nil
	},
}

func ListSandboxes() []string {
	sandboxDir := filepath.Join(util.GetHomeDir(), ".enonic", "sandboxes")
	files, err := ioutil.ReadDir(sandboxDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not list sandboxes: ", err)
	}
	return filterSandboxes(files, sandboxDir)
}

func filterSandboxes(vs []os.FileInfo, sandboxDir string) []string {
	vsf := make([]string, 0)
	for _, v := range vs {
		if v.Name() == ".enonic" {
			continue
		}
		if isSandbox(v, sandboxDir) {
			vsf = append(vsf, v.Name())
		} else {
			fmt.Fprintf(os.Stderr, "Warning: '%s' is not a valid sandbox folder.\n", v.Name())
		}
	}
	return vsf
}

func isSandbox(v os.FileInfo, sandboxDir string) bool {
	if v.IsDir() {
		descriptorPath := filepath.Join(sandboxDir, v.Name(), ".enonic")
		if _, err := os.Stat(descriptorPath); err == nil {
			return true
		} else {
			return false
		}
	} else {
		return false
	}
}
