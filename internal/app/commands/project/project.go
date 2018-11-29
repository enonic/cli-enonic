package project

import (
	"github.com/urfave/cli"
	"github.com/enonic/xp-cli/internal/app/util"
	"os"
	"github.com/enonic/xp-cli/internal/app/commands/sandbox"
	"fmt"
	"os/exec"
)

func All() []cli.Command {
	return []cli.Command{
		Create,
		Sandbox,
		Clean,
		Build,
		Deploy,
		Install,
	}
}

type ProjectData struct {
	Sandbox string `toml:"sandbox"`
}

func readProjectData() ProjectData {
	file := util.OpenOrCreateDataFile(".enonic", true)
	defer file.Close()

	var data ProjectData
	util.DecodeTomlFile(file, &data)
	return data
}

func writeProjectData(data ProjectData) {
	file := util.OpenOrCreateDataFile(".enonic", false)
	defer file.Close()

	util.EncodeTomlFile(file, data)
}

func ensureProjectDataExists(c *cli.Context) ProjectData {
	projectData := readProjectData()
	noSandbox := projectData.Sandbox == ""
	argExist := c != nil && c.NArg() > 0
	if noSandbox || argExist {
		sbox := sandbox.EnsureSandboxNameExists(c, "Select a sandbox to use:")
		projectData.Sandbox = sbox.Name
		if noSandbox {
			writeProjectData(projectData)
			fmt.Fprintf(os.Stderr, "Sandbox '%s' set as default. You can change it using 'project sandbox command' at any time.\n", projectData.Sandbox)
		}
	}
	return projectData
}

func runGradleTask(projectData ProjectData, task, message string) {

	sandboxData := sandbox.ReadSandboxData(projectData.Sandbox)

	javaHome := fmt.Sprintf("-Dorg.gradle.java.home=%s", sandbox.GetDistroJdkPath(sandboxData.Distro))
	xpHome := fmt.Sprintf("-Dxp.home=%s", sandbox.GetSandboxHomePath(projectData.Sandbox))

	fmt.Fprint(os.Stderr, message)
	err := exec.Command("gradlew", task, javaHome, xpHome).Run()
	util.Fatal(err, fmt.Sprintf("Could not %s the project", task))

	fmt.Fprintln(os.Stderr, "Done")
}
