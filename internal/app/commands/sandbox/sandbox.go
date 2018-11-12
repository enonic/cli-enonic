package sandbox

import (
	"github.com/urfave/cli"
	"fmt"
	"os"
	"github.com/mitchellh/go-homedir"
	"path/filepath"
	"io/ioutil"
	"regexp"
	"github.com/Masterminds/semver"
)

func All() []cli.Command {
	ensureDirStructure()

	return []cli.Command{
		List,
		Start,
		New,
		Delete,
		Version,
	}
}

func ensureDirStructure() {
	// Using go-homedir instead of user.Current()
	// because of https://github.com/golang/go/issues/6376
	home := getHomeDir()
	createFolderIfNotExist(home, ".enonic", "distributions")
	createFolderIfNotExist(home, ".enonic", "sandboxes")
}
func getHomeDir() string {
	home, err := homedir.Dir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not get user home dir: ", err)
		os.Exit(1)
	}
	return home
}

func createFolderIfNotExist(paths ...string) {
	fullPath := filepath.Join(paths...)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		err = os.MkdirAll(fullPath, 0755)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Could not create dir: ", err)
			os.Exit(1)
		}
	}
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
