package project

import (
	"fmt"
	"github.com/enonic/cli-enonic/internal/app/commands/common"
	"github.com/enonic/cli-enonic/internal/app/commands/sandbox"
	"github.com/enonic/cli-enonic/internal/app/util"
	"github.com/urfave/cli"
	"os"
	"os/exec"
	"path"
)

func All() []cli.Command {
	commands := []cli.Command{
		Create,
		Sandbox,
		Clean,
		Build,
		Deploy,
		Install,
		Shell,
		Gradle,
	}

	switch util.GetCurrentOs() {
	case "windows":
		// do not add Env to windows as it's not supported
	default:
		commands = append(commands, Env)
	}

	return commands
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

func ensureProjectDataExists(c *cli.Context, prjPath, noBoxMessage string) *common.ProjectData {
	var newBox bool
	var sBox *sandbox.Sandbox

	ensureValidProjectFolder(prjPath)

	projectData := common.ReadProjectData(prjPath)
	badSandbox := projectData.Sandbox == "" || !sandbox.Exists(projectData.Sandbox)
	argExist := c != nil && c.NArg() > 0
	if badSandbox || argExist {
		sBox, newBox = sandbox.EnsureSandboxExists(c, noBoxMessage, "A sandbox is required for your project, select one:", false, true)
		if sBox == nil {
			return nil
		}
		projectData.Sandbox = sBox.Name
		if badSandbox {
			common.WriteProjectData(projectData, prjPath)
		}
	} else {
		sBox = sandbox.ReadSandboxData(projectData.Sandbox)
	}

	fmt.Fprint(os.Stderr, "\n")
	distroPath, newDistro := sandbox.EnsureDistroExists(sBox.Distro)

	if newBox || newDistro {
		sandbox.CopyHomeFolder(distroPath, projectData.Sandbox)

		if newBox {
			fmt.Fprintf(os.Stderr, "Sandbox '%s' created.\n", sBox.Name)
		}
	}

	return projectData
}

func runGradleTask(projectData *common.ProjectData, message string, tasks ...string) {

	sandboxData := sandbox.ReadSandboxData(projectData.Sandbox)

	javaHome := fmt.Sprintf("-Dorg.gradle.java.home=%s", sandbox.GetDistroJdkPath(sandboxData.Distro))
	xpHome := fmt.Sprintf("-Dxp.home=%s", sandbox.GetSandboxHomePath(projectData.Sandbox))
	args := append(tasks, javaHome, xpHome)

	fmt.Fprintln(os.Stderr, message)

	command := getOsGradlewFile()
	cmd := exec.Command(command, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	cmd.Run()
}
