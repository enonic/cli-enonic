package sandbox

import (
	"bufio"
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/magiconair/properties"
	"github.com/otiai10/copy"
	"github.com/urfave/cli"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
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

	data := SandboxData{formatDistroVersion(version)}
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
	if name == "" {
		return false
	}
	dir := getSandboxesDir()
	if info, err := os.Stat(filepath.Join(dir, name)); os.IsNotExist(err) {
		return false
	} else {
		return isSandbox(info, dir)
	}
}

func EnsureSandboxExists(c *cli.Context, minDistroVersion, name string, noBoxMessage, selectBoxMessage string, showSuccessMessage, showCreateOption bool) (*Sandbox, bool) {
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

	if name != "" {
		lowerName := strings.ToLower(name)
		for _, existingBox := range existingBoxes {
			if strings.ToLower(existingBox.Name) == lowerName {
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
	var boxName, defaultBox string
	osWithArch := util.GetCurrentOsWithArch()
	for i, box := range existingBoxes {
		version := parseDistroVersion(box.Distro, false)
		boxName = formatSandboxListItemName(box.Name, version, osWithArch)
		if i == 0 {
			defaultBox = boxName
		}
		selectOptions = append(selectOptions, boxName)
	}

	name, err := util.PromptSelect(&util.SelectOptions{
		Message:  selectBoxMessage,
		Options:  selectOptions,
		Default:  defaultBox,
		PageSize: len(selectOptions),
	})
	util.Fatal(err, "Could not select sandbox: ")

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
	targetHome := GetSandboxHomePath(sandboxName)
	if _, err := os.Stat(targetHome); err == nil {
		// it already exists
		return
	}
	sourceHome := filepath.Join(distroPath, "home")
	if _, err := os.Stat(sourceHome); err == nil {
		copyErr := copy.Copy(sourceHome, targetHome, copy.Options{AddPermission: 0200})
		util.Fatal(copyErr, "Could not copy home folder from distro to sandbox: ")
		updateXPConfig(sandboxName)
	}
}

func updateXPConfig(sandboxName string) {
	configFolder := createFolderIfNotExist(GetSandboxHomePath(sandboxName), "config")
	configPath := filepath.Join(configFolder, "system.properties")
	configFile := util.OpenOrCreateDataFile(configPath, false)
	defer configFile.Close()
	props, readErr := properties.LoadFile(configPath, properties.UTF8)
	if readErr != nil {
		fmt.Fprintln(os.Stderr, "Error reading system.properties file", readErr.Error())
		return
	}
	if prev, _, _ := props.Set("xp.name", sandboxName); prev != sandboxName {
		configWriter := bufio.NewWriter(configFile)
		if _, writerErr := props.Write(configWriter, properties.UTF8); writerErr != nil {
			fmt.Fprintln(os.Stderr, "Error writing system.properties file", readErr.Error())
			return
		}
		if flushErr := configWriter.Flush(); flushErr != nil {
			fmt.Fprintln(os.Stderr, "Error writing system.properties file", readErr.Error())
			return
		}
	}
}

func ensureDirStructure() {
	// Using go-homedir instead of user.Current()
	// because of https://github.com/golang/go/issues/6376
	home := util.GetHomeDir()

	enonicPath := filepath.Join(home, ".enonic")
	enonicPathInfo, _ := os.Lstat(enonicPath)
	enonicPathExists := enonicPathInfo != nil
	enonicPathIsSymlink := enonicPathExists && enonicPathInfo.Mode()&os.ModeSymlink == os.ModeSymlink

	if util.GetCurrentOs() == "linux" {
		if snapCommon, snapExists := os.LookupEnv(common.SNAP_ENV_VAR); snapExists {
			// snapcraft is present

			snapPath := filepath.Join(snapCommon, "dot-enonic")
			_, err := os.Lstat(snapPath)
			snapPathExists := err == nil

			// it may be a symlink (ok) or it may be a leftover from manual installation,
			// in that case replace it with a symlink to a snap folder and move files there
			if enonicPathExists {

				if !snapPathExists {
					// move contents of enonic folder if it is not a symlink
					// symlink will be created later
					if !enonicPathIsSymlink {
						err := os.Rename(enonicPath, snapPath)
						util.Warn(err, fmt.Sprintf("Error moving data from '%s' to '%s'", enonicPath, snapPath))
					} else {
						// create a snap folder if needed
						// symlink will be created later
						createFolderIfNotExist(snapPath)
					}
				}
				// remove old enonic folder or a symlink
				// symlink will be created later
				err := os.RemoveAll(enonicPath)
				util.Warn(err, fmt.Sprintf("Error deleting '%s'", enonicPath))

			} else {
				createFolderIfNotExist(snapPath)
			}

			err = os.Symlink(snapPath, enonicPath)
			util.Fatal(err, fmt.Sprintf("Error creating a symlink '%s' to '%s' folder", enonicPath, snapPath))

		} else {
			// snapcraft is not present
			// enonic path will be a symlink if snapcraft was previously installed
			// follow it and move everything to enonic path
			if enonicPathExists && enonicPathIsSymlink {
				snapPath, err := os.Readlink(enonicPath)
				util.Fatal(err, fmt.Sprintf("Could not resolve a symlink '%s'.", enonicPath))

				err = os.Remove(enonicPath)
				util.Fatal(err, fmt.Sprintf("Error deleting a symlink '%s'. Delete it manually and re-run.", enonicPath))

				err = os.Rename(snapPath, enonicPath)
				util.Warn(err, fmt.Sprintf("Error moving data from '%s' to '%s'", enonicPath, snapPath))
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
