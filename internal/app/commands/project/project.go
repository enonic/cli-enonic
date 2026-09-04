package project

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/commands/sandbox"
	"cli-enonic/internal/app/util"
	"cli-enonic/internal/app/util/system"
	"fmt"
	"os"
	"path"

	"github.com/Masterminds/semver"
	"github.com/urfave/cli"
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
		Dev,
		Test,
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
		gradlewFile = fmt.Sprintf(".%c%s.bat", os.PathSeparator, gradlewFile)
	case "mac", "linux":
		gradlewFile = fmt.Sprintf(".%c%s", os.PathSeparator, gradlewFile)
	}
	return gradlewFile
}

func hasGradleWrapper(prjPath string) bool {
	_, err := os.Stat(path.Join(prjPath, getOsGradlewFile()))
	return !os.IsNotExist(err)
}

func ensureValidProjectFolder(prjPath string) {
	if !hasGradleWrapper(prjPath) && !common.IsStaticProject(prjPath) {
		fmt.Fprintln(os.Stderr, "Not a valid project folder")
		os.Exit(1)
	}
}

func ensureProjectDataExists(c *cli.Context, prjPath, sandboxName, noBoxMessage string) (*common.ProjectData, bool) {
	return ensureProjectData(c, prjPath, sandboxName, noBoxMessage, false)
}

func ensureProjectData(c *cli.Context, prjPath, sandboxName, noBoxMessage string, sandboxOptional bool) (*common.ProjectData, bool) {
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

	badSandbox := !sandbox.Exists(projectData.Sandbox)
	force := common.IsForceMode(c)

	if force && badSandbox && sandboxName == "" {
		// allow project without a sandbox in force mode
		return projectData, newBox
	} else if badSandbox || sandboxName != "" {
		selectBoxMessage := "A sandbox is required for your project, select one or create new"
		if sandboxOptional {
			selectBoxMessage = "Select a sandbox to use with the project"
		}
		sBox, newBox = sandbox.EnsureSandboxExists(c, sandbox.EnsureSandboxOptions{
			MinDistroVersion: minDistroVersion,
			Name:             sandboxName,
			NoBoxMessage:     noBoxMessage,
			SelectBoxMessage: selectBoxMessage,
			ShowCreateOption: true,
			ShowSkipOption:   sandboxOptional,
		})
		if sBox == nil {
			if sandboxOptional {
				return projectData, newBox
			}
			return nil, newBox
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
	distroPath, newDistro := sandbox.EnsureDistroExists(c, sBox.Distro)

	if newBox || newDistro {
		sandbox.CopyHomeFolder(distroPath, projectData.Sandbox)

		if newBox {
			fmt.Fprintf(os.Stderr, "Sandbox '%s' created.\n", sBox.Name)
		}
	}

	return projectData, newBox
}

func runGradleTask(projectData *common.ProjectData, message string, tasks ...string) {
	if !hasGradleWrapper(".") {
		if common.IsStaticProject(".") {
			fmt.Fprintln(os.Stderr, "Gradle tasks are not supported for Static applications. Use 'enonic project deploy' to build and install the application.")
		} else {
			fmt.Fprintln(os.Stderr, "Not a valid project folder")
		}
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, message)
	args := tasks
	env := os.Environ()
	if projectData.Sandbox != "" && sandbox.Exists(projectData.Sandbox) {
		sandboxData := sandbox.ReadSandboxData(projectData.Sandbox)
		javaHome := sandbox.GetDistroJdkPath(sandboxData.Distro)
		xpHome := sandbox.GetSandboxHomePath(projectData.Sandbox)

		if javaHome != "" {
			args = append(args, fmt.Sprintf("-Dorg.gradle.java.home=%s", javaHome))
			env = append(env, fmt.Sprintf("JAVA_HOME=%s", javaHome))
		}
		args = append(args, fmt.Sprintf("-Dxp.home=%s", xpHome))

		env = append(env, fmt.Sprintf("XP_HOME=%s", xpHome))
	}

	command := getOsGradlewFile()

	system.Run(command, args, env)
}
