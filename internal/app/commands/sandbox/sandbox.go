package sandbox

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/otiai10/copy"
	"github.com/urfave/cli"
	"gopkg.in/AlecAivazis/survey.v1"
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

	data := SandboxData{formatDistroVersion(version, util.GetCurrentOs())}
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
	return common.GetInEnonicDir("sandboxes")
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
	for _, box := range listSandboxes("") {
		if data := ReadSandboxData(box.Name); data.Distro == distroName {
			usedBy = append(usedBy, box)
		}
	}
	return usedBy
}

func deleteSandbox(name string) {
	common.StartSpinner("Deleting sandbox")
	err := os.RemoveAll(filepath.Join(getSandboxesDir(), name))
	common.StopSpinner()
	util.Fatal(err, fmt.Sprintf("Could not delete sandbox '%s' folder: ", name))
}

func listSandboxes(minDistroVersion string) []*Sandbox {
	sandboxesDir := getSandboxesDir()
	files, err := ioutil.ReadDir(sandboxesDir)
	util.Fatal(err, "Could not list sandboxes: ")
	return filterSandboxes(files, sandboxesDir, minDistroVersion)
}

func filterSandboxes(vs []os.FileInfo, sandboxDir, minDistroVersion string) []*Sandbox {
	minDistroVer, _ := semver.NewVersion(minDistroVersion)
	vsf := make([]*Sandbox, 0)
	for _, v := range vs {
		if !v.IsDir() {
			continue
		}
		if isSandbox(v, sandboxDir) {
			sandboxData := ReadSandboxData(v.Name())
			distroVer, _ := semver.NewVersion(parseDistroVersion(sandboxData.Distro, false))
			if distroVer == nil || minDistroVer == nil || !distroVer.LessThan(minDistroVer) {
				vsf = append(vsf, sandboxData)
			}
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

func EnsureSandboxExists(c *cli.Context, minDistroVersion, noBoxMessage, selectBoxMessage string, showSuccessMessage, showCreateOption, parseArgs bool) (*Sandbox, bool) {
	existingBoxes := listSandboxes(minDistroVersion)
	force := common.IsForceMode(c)

	if len(existingBoxes) == 0 {
		if force {
			fmt.Fprintln(os.Stderr, "No sandboxes found. Create one using 'enonic sandbox create' first.")
			os.Exit(1)
		}
		if !util.PromptBool(noBoxMessage, true) {
			return nil, false
		}
		newBox := SandboxCreateWizard("", "", minDistroVersion, false, showSuccessMessage, force)
		return newBox, true
	}

	if parseArgs && c != nil && c.NArg() > 0 {
		name := c.Args().First()
		for _, existingBox := range existingBoxes {
			if existingBox.Name == name {
				return existingBox, false
			}
		}
		if force {
			fmt.Fprintf(os.Stderr, "Sandbox with name '%s' can not be found\n", name)
			os.Exit(1)
		}
	}
	if force {
		fmt.Fprintln(os.Stderr, "Sandbox name can not be empty in non-interactive mode")
		os.Exit(1)
	}

	var selectOptions []string
	if showCreateOption {
		selectOptions = append(selectOptions, CREATE_NEW_BOX)
	}
	var myOs = util.GetCurrentOs()
	var boxName, defaultBox string
	for i, box := range existingBoxes {
		version := parseDistroVersion(box.Distro, false)
		boxName = formatSandboxListItemName(box.Name, version, myOs)
		if i == 0 {
			defaultBox = boxName
		}
		selectOptions = append(selectOptions, boxName)
	}

	var name string
	prompt := &survey.Select{
		Message:  selectBoxMessage,
		Options:  selectOptions,
		Default:  defaultBox,
		PageSize: len(selectOptions),
	}

	err := survey.AskOne(prompt, &name, nil)
	util.Fatal(err, "Select failed: ")

	if name == CREATE_NEW_BOX {
		newBox := SandboxCreateWizard("", "", minDistroVersion, false, showSuccessMessage, force)
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

	if util.GetCurrentOs() == "linux" {
		if snapCommon, snapExists := os.LookupEnv(common.SNAP_ENV_VAR); snapExists {
			snapPath := createFolderIfNotExist(snapCommon, "dot-enonic")

			enonicPath := filepath.Join(home, ".enonic")
			if _, err := os.Stat(enonicPath); os.IsNotExist(err) {
				err := os.Symlink(snapPath, enonicPath)
				util.Fatal(err, fmt.Sprintf("Error creating a symlink '%s' to '%s' folder", enonicPath, snapPath))
			}
		}
	}

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
