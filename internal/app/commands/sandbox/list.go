package sandbox

import (
	"github.com/urfave/cli"
	"path/filepath"
	"io/ioutil"
	"fmt"
	"os"
	"regexp"
	"github.com/Masterminds/semver"
)

var List = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Usage:   "List all sandboxes",
	Action: func(c *cli.Context) error {
		activeDistro := GetActiveDistro()
		for _, d := range ListDistros() {
			if activeDistro == d {
				fmt.Fprintf(os.Stderr, "* %s\n", d)
			} else {
				fmt.Fprintf(os.Stderr, "  %s\n", d)
			}
		}
		return nil
	},
}

func GetActiveDistro() string {
	//TODO
	return ""
}

func ListDistros() []string {
	distrosDir := filepath.Join(getHomeDir(), ".enonic", "distributions")
	files, err := ioutil.ReadDir(distrosDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not list distros: ", err)
	}
	return filterDistros(files, distrosDir)
}

func filterDistros(vs []os.FileInfo, distrosDir string) []string {
	vsf := make([]string, 0)
	for _, v := range vs {
		if isDistro(v) {
			vsf = append(vsf, v.Name())
		} else {
			if err := os.RemoveAll(filepath.Join(distrosDir, v.Name())); err != nil {
				fmt.Fprintln(os.Stderr, "Could not remove invalid distro: ", err)
			}
		}
	}
	return vsf
}

func isDistro(v os.FileInfo) bool {
	distroRegexp := regexp.MustCompile(semver.SemVerRegex)
	return v.IsDir() && distroRegexp.MatchString(v.Name())
}
