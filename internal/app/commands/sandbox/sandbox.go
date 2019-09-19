package sandbox

import (
	"fmt"
	"github.com/AlecAivazis/survey"
	"github.com/enonic/cli-enonic/internal/app/commands/common"
	"github.com/enonic/cli-enonic/internal/app/util"
	"github.com/otiai10/copy"
	"github.com/urfave/cli"
	"io/ioutil"
	"os"
	"path/filepath"
)

func All() []cli.Command {
	ensureDirStructure()

	return []cli.Command{
		List,
		Start,
		Stop,
		Create,
		Delete,
		Upgrade,
	}
}

var CREATE_NEW_BOX = "Create new sandbox"

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
	return filepath.Join(common.GetEnonicDir(), "sandboxes")
}

func GetActiveHomePath() string {
	var homePath string
	rData := common.ReadRuntimeData()
	if rData.Running != "" {
		homePath = GetSandboxHomePath(rData.Running)
	} else {
		homePath = os.Getenv(common.ENV_XP_HOME)
	}
	return homePath
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

func EnsureSandboxExists(c *cli.Context, noBoxMessage, selectBoxMessage string, showSuccessMessage, showCreateOption bool) (*Sandbox, bool) {
	existingBoxes := listSandboxes()

	if len(existingBoxes) == 0 {
		if !util.PromptBool(noBoxMessage, true) {
			return nil, false
		}
		newBox := SandboxCreateWizard("", "", showSuccessMessage)
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

	var selectOptions []string
	if showCreateOption {
		selectOptions = append(selectOptions, CREATE_NEW_BOX)
	}
	var boxName, defaultBox string
	for i, box := range existingBoxes {
		boxName = fmt.Sprintf("%s ( %s )", box.Name, box.Distro)
		if i == 0 {
			defaultBox = boxName
		}
		selectOptions = append(selectOptions, boxName)
	}

	var name string
	prompt := &survey.Select{
		Message: selectBoxMessage,
		Options: selectOptions,
		Default: defaultBox,
	}

	err := survey.AskOne(prompt, &name, nil)
	util.Fatal(err, "Select failed: ")

	if name == CREATE_NEW_BOX {
		newBox := SandboxCreateWizard("", "", showSuccessMessage)
		return newBox, true
	}

	optionIndex := util.IndexOf(name, selectOptions)
	if showCreateOption {
		optionIndex -= 1 // subtract 1 because of 'new sandbox' option
	}
	return existingBoxes[optionIndex], false
}

func CopyHomeFolder(distroPath, sandboxName string) {
	homePath := GetSandboxHomePath(sandboxName)
	if _, err := os.Stat(homePath); os.IsNotExist(err) {
		err := copy.Copy(filepath.Join(distroPath, "home"), homePath)
		util.Fatal(err, "Could not copy home folder from distro: ")
	}
}

func ensureDirStructure() {
	// Using go-homedir instead of user.Current()
	// because of https://github.com/golang/go/issues/6376
	home := util.GetHomeDir()
	createFolderIfNotExist(home, ".enonic", "distributions")
	createFolderIfNotExist(home, ".enonic", "sandboxes")

	if util.GetCurrentOs() == "linux" {
		if snapCommon, snapExists := os.LookupEnv(SNAP_ENV_VAR); snapExists {
			linkPath := filepath.Join(snapCommon, "dot-enonic")
			if _, err := os.Stat(linkPath); os.IsNotExist(err) {
				err := os.Link(filepath.Join(home, ".enonic"), linkPath)
				util.Fatal(err, "Error creating a symbolic link to '.enonic' folder")
			}
		}
	}
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
