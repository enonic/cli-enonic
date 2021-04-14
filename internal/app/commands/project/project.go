package project

import (
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/enonic/cli-enonic/internal/app/commands/common"
	"github.com/enonic/cli-enonic/internal/app/commands/sandbox"
	"github.com/enonic/cli-enonic/internal/app/util"
	"github.com/urfave/cli"
	"os"
	"os/exec"
	"path"
	"syscall"
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
		os.Exit(1)
	}
}

func ensureProjectDataExists(c *cli.Context, prjPath, noBoxMessage string, parseArgs bool) *common.ProjectData {
	var newBox bool
	var sBox *sandbox.Sandbox

	ensureValidProjectFolder(prjPath)

	projectData := common.ReadProjectData(prjPath)
	minDistroVersion := common.ReadProjectDistroVersion(prjPath)

	minDistroVer := semver.MustParse(minDistroVersion)
	if minDistroVer.LessThan(semver.MustParse(common.MIN_XP_VERSION)) {
		fmt.Fprintf(os.Stderr, "XP version in your application is not supported by CLI. Got %s, expected %s or higher.\n", minDistroVersion, common.MIN_XP_VERSION)
		os.Exit(1)
	}

	badSandbox := projectData.Sandbox == "" || !sandbox.Exists(projectData.Sandbox)
	argExist := parseArgs && c != nil && c.NArg() > 0
	force := common.IsForceMode(c)

	if force && badSandbox {
		if projectData.Sandbox == "" {
			fmt.Fprintln(os.Stderr, "Project sandbox is not set. Set it using 'enonic project sandbox' command")
		} else {
			fmt.Fprintf(os.Stderr, "Project sandbox '%s' can not be found. Update it using 'enonic project sandbox' command\n", projectData.Sandbox)
		}
		os.Exit(1)
	} else if badSandbox || argExist {
		sBox, newBox = sandbox.EnsureSandboxExists(c, minDistroVersion, noBoxMessage, "A sandbox is required for your project, select one:", false, true, parseArgs)
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

	sandbox.EnsureSanboxSupportsProjectVersion(sBox, minDistroVer)

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

	javaHome := sandbox.GetDistroJdkPath(sandboxData.Distro)
	xpHome := sandbox.GetSandboxHomePath(projectData.Sandbox)

	javaHomeArg := fmt.Sprintf("-Dorg.gradle.java.home=%s", javaHome)
	xpHomeArg := fmt.Sprintf("-Dxp.home=%s", xpHome)
	args := append(tasks, javaHomeArg, xpHomeArg)

	fmt.Fprintln(os.Stderr, message)

	command := getOsGradlewFile()
	cmd := exec.Command(command, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("JAVA_HOME=%s", javaHome))
	cmd.Env = append(cmd.Env, fmt.Sprintf("XP_HOME=%s", xpHome))

	if err := cmd.Run(); err != nil {
		os.Stderr.WriteString(fmt.Sprintf("\n%s\n", err.Error()))
		if exitError, ok := err.(*exec.ExitError); ok {
			waitStatus := exitError.Sys().(syscall.WaitStatus)
			os.Exit(waitStatus.ExitStatus())
		} else {
			os.Exit(1)
		}
	}
}
