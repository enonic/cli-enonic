package sandbox

import (
	"github.com/urfave/cli"
	"fmt"
	"os"
	"path/filepath"
	"github.com/enonic/xp-cli/internal/app/util"
	"github.com/AlecAivazis/survey"
	"io/ioutil"
)

func All() []cli.Command {
	ensureDirStructure()

	return []cli.Command{
		List,
		Start,
		Create,
		Delete,
		Upgrade,
	}
}

var CREATE_NEW_BOX = "Create new sandbox"

type SandboxesData struct {
	Running string `toml:"running"`
}

type SandboxData struct {
	Distro string `toml:"distro"`
}

type Sandbox struct {
	Name   string
	Distro string
}

func createSandbox(name string, version string) *Sandbox {
	dir := createFolderIfNotExist(getSandboxesDir(), name)

	file := util.OpenOrCreateDataFile(filepath.Join(dir, ".enonic"), false)
	defer file.Close()

	data := SandboxData{formatDistroVersion(version, util.GetCurrentOs(), true)}
	util.EncodeTomlFile(file, data)

	return &Sandbox{name, data.Distro}
}

func readSandboxesData() SandboxesData {
	path := filepath.Join(getSandboxesDir(), ".enonic")
	file := util.OpenOrCreateDataFile(path, true)
	defer file.Close()

	var data SandboxesData
	util.DecodeTomlFile(file, &data)
	return data
}

func writeSandboxesData(data SandboxesData) {
	path := filepath.Join(getSandboxesDir(), ".enonic")
	file := util.OpenOrCreateDataFile(path, false)
	defer file.Close()

	util.EncodeTomlFile(file, data)
}

func ReadSandboxData(name string) *Sandbox {
	path := filepath.Join(getSandboxesDir(), name, ".enonic")
	file := util.OpenOrCreateDataFile(path, true)
	defer file.Close()

	var data SandboxData
	util.DecodeTomlFile(file, &data)
	return &Sandbox{name, data.Distro}
}

func writeSandboxData(data *Sandbox) {
	path := filepath.Join(getSandboxesDir(), data.Name, ".enonic")
	file := util.OpenOrCreateDataFile(path, false)
	defer file.Close()

	util.EncodeTomlFile(file, SandboxData{data.Distro})
}

func getSandboxesDir() string {
	return filepath.Join(util.GetHomeDir(), ".enonic", "sandboxes")
}

func GetSandboxHomePath(name string) string {
	return filepath.Join(getSandboxesDir(), name, "home")
}

func getSandboxesUsingDistro(distroName string) []*Sandbox {
	usedBy := make([]*Sandbox, 0)
	for _, box := range listSandboxes() {
		if data := ReadSandboxData(box.Name); data.Distro == distroName {
			usedBy = append(usedBy, box)
		}
	}
	return usedBy
}

func deleteSandbox(name string) {
	err := os.RemoveAll(filepath.Join(getSandboxesDir(), name))
	util.Warn(err, fmt.Sprintf("Could not delete sandbox '%s' folder: ", name))
}

func listSandboxes() []*Sandbox {
	sandboxesDir := getSandboxesDir()
	files, err := ioutil.ReadDir(sandboxesDir)
	util.Fatal(err, "Could not list sandboxes: ")
	return filterSandboxes(files, sandboxesDir)
}

func filterSandboxes(vs []os.FileInfo, sandboxDir string) []*Sandbox {
	vsf := make([]*Sandbox, 0)
	for _, v := range vs {
		if !v.IsDir() {
			continue
		}
		if isSandbox(v, sandboxDir) {
			data := ReadSandboxData(v.Name())
			vsf = append(vsf, data)
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

func Exists(name string) bool {
	dir := getSandboxesDir()
	if info, err := os.Stat(filepath.Join(dir, name)); os.IsNotExist(err) {
		return false
	} else {
		return isSandbox(info, dir)
	}
}

func EnsureSandboxExists(c *cli.Context, noBoxMessage, selectBoxMessage string) (*Sandbox, bool) {
	existingBoxes := listSandboxes()

	if len(existingBoxes) == 0 {
		if !util.YesNoPrompt(noBoxMessage) {
			return nil, false
		}
		newBox := SandboxCreateWizard("", "")
		return newBox, true
	}

	if c != nil && c.NArg() > 0 {
		name := c.Args().First()
		for _, existingBox := range existingBoxes {
			if existingBox.Name == name {
				return existingBox, false
			}
		}
	}

	selectOptions := []string{CREATE_NEW_BOX}
	for _, box := range existingBoxes {
		selectOptions = append(selectOptions, fmt.Sprintf("%s ( %s )", box.Name, box.Distro))
	}

	var name string
	prompt := &survey.Select{
		Message: selectBoxMessage,
		Options: selectOptions,
	}

	err := survey.AskOne(prompt, &name, nil)
	util.Fatal(err, "Select failed: ")

	if name == CREATE_NEW_BOX {
		newBox := SandboxCreateWizard("", "")
		return newBox, true
	}

	// subtract 1 because of 'new sandbox' option
	return existingBoxes[util.IndexOf(name, selectOptions)-1], false
}

func ensureDirStructure() {
	// Using go-homedir instead of user.Current()
	// because of https://github.com/golang/go/issues/6376
	home := util.GetHomeDir()
	createFolderIfNotExist(home, ".enonic", "distributions")
	createFolderIfNotExist(home, ".enonic", "sandboxes")
}

func createFolderIfNotExist(paths ...string) string {
	fullPath := filepath.Join(paths...)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		err = os.MkdirAll(fullPath, 0755)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Could not create dir: ", err)
			os.Exit(1)
		}
	}
	return fullPath
}
