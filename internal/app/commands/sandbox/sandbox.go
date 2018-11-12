package sandbox

import (
	"github.com/urfave/cli"
	"fmt"
	"os"
	"github.com/mitchellh/go-homedir"
	"path/filepath"
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
