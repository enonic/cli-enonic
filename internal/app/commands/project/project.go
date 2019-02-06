package project

import (
	"github.com/urfave/cli"
	"github.com/enonic/enonic-cli/internal/app/util"
	"os"
	"github.com/enonic/enonic-cli/internal/app/commands/sandbox"
	"fmt"
	"os/exec"
	"path/filepath"
	"github.com/otiai10/copy"
	"path"
)

func All() []cli.Command {
	return []cli.Command{
		Create,
		Sandbox,
		Clean,
		Build,
		Deploy,
		Install,
		Shell,
	}
}

type ProjectData struct {
	Sandbox string `toml:"sandbox"`
}

func readProjectData(prjPath string) *ProjectData {
	file := util.OpenOrCreateDataFile(filepath.Join(prjPath, ".enonic"), true)
	defer file.Close()

	var data ProjectData
	util.DecodeTomlFile(file, &data)
	return &data
}

func writeProjectData(data *ProjectData, prjPath string) {
	file := util.OpenOrCreateDataFile(filepath.Join(prjPath, ".enonic"), false)
	defer file.Close()

	util.EncodeTomlFile(file, data)
}

func getOsGradlewFile() string {
	gradlewFile := "gradlew"
	switch util.GetCurrentOs() {
	case "windows":
		gradlewFile += ".bat"
	case "mac", "linux":
		gradlewFile = "./" + gradlewFile

	}
	return gradlewFile
}

func ensureValidProjectFolder(prjPath string) {
	if _, err := os.Stat(path.Join(prjPath, getOsGradlewFile())); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "Not a valid project folder")
		os.Exit(0)
	}
}

func ensureProjectDataExists(c *cli.Context, prjPath, noBoxMessage string) *ProjectData {
	var newBox bool
	var sBox *sandbox.Sandbox

	ensureValidProjectFolder(prjPath)

	projectData := readProjectData(prjPath)
	badSandbox := projectData.Sandbox == "" || !sandbox.Exists(projectData.Sandbox)
	argExist := c != nil && c.NArg() > 0
	if badSandbox || argExist {
		sBox, newBox = sandbox.EnsureSandboxExists(c, noBoxMessage, "A sandbox is required for your project, select one:")
		if sBox == nil {
			return nil
		}
		projectData.Sandbox = sBox.Name
		if badSandbox {
			writeProjectData(projectData, prjPath)
			fmt.Fprintf(os.Stderr, "Project is now linked to sandbox '%s' using '%s'\n", projectData.Sandbox, sBox.Distro)
		}
	} else {
		sBox = sandbox.ReadSandboxData(projectData.Sandbox)
	}
	distroPath, newDistro := sandbox.EnsureDistroExists(sBox.Distro)

	if newBox || newDistro {
		err := copy.Copy(filepath.Join(distroPath, "home"), sandbox.GetSandboxHomePath(projectData.Sandbox))
		util.Fatal(err, "Could not copy home folder from distro: ")
	}

	return projectData
}

func runGradleTask(projectData *ProjectData, task, message string) {

	sandboxData := sandbox.ReadSandboxData(projectData.Sandbox)

	javaHome := fmt.Sprintf("-Dorg.gradle.java.home=%s", sandbox.GetDistroJdkPath(sandboxData.Distro))
	xpHome := fmt.Sprintf("-Dxp.home=%s", sandbox.GetSandboxHomePath(projectData.Sandbox))

	fmt.Fprintln(os.Stderr, message)
	command := getOsGradlewFile()
	cmd := exec.Command(command, task, javaHome, xpHome)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	cmd.Run()
}
