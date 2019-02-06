package project

import (
	"github.com/urfave/cli"
	"github.com/enonic/enonic-cli/internal/app/commands/common"
	"os"
	"github.com/enonic/enonic-cli/internal/app/commands/sandbox"
	"os/exec"
	"github.com/enonic/enonic-cli/internal/app/util"
	"fmt"
)

var Shell = cli.Command{
	Name:  "shell",
	Usage: "Creates a new shell with project environment variables",
	Action: func(c *cli.Context) error {

		ensureValidProjectFolder(".")

		pData := readProjectData(".")
		sBox := sandbox.ReadSandboxData(pData.Sandbox)

		cmd := createNewTerminalCommand()

		prjJavaHome := sandbox.GetDistroJdkPath(sBox.Distro)
		prjXpHome := sandbox.GetSandboxHomePath(pData.Sandbox)
		os.Setenv(common.ENV_JAVA_HOME, prjJavaHome)
		os.Setenv(common.ENV_XP_HOME, prjXpHome)
		cmd.Env = os.Environ()

		err := cmd.Start()
		util.Warn(err, "Could not start new shell")
		fmt.Fprintf(os.Stderr, "Started new project shell with PID %d.\n", cmd.Process.Pid)

		cmd.Wait()
		fmt.Fprintln(os.Stderr, "Project shell has finished.")

		return nil
	},
}

func createNewTerminalCommand() *exec.Cmd {
	prjDir, err := os.Getwd()
	util.Warn(err, "Could not get current working dir")

	switch util.GetCurrentOs() {
	case "windows":
		return exec.Command("cmd", "/C", "start", "/d", prjDir)
	case "mac":
		return exec.Command("open", "-F", "-n", "-b", "com.apple.Terminal", prjDir)
	default:
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "bash"
		}
		cmd := exec.Command(shell, "--init-file", "'<(enonic)'")
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout

		return cmd
	}
}
