package project

import (
	"fmt"
	"github.com/enonic/cli-enonic/internal/app/commands/common"
	"github.com/enonic/cli-enonic/internal/app/commands/sandbox"
	"github.com/enonic/cli-enonic/internal/app/util"
	"github.com/urfave/cli"
	"os"
	"os/exec"
)

var Shell = cli.Command{
	Name:  "shell",
	Usage: "Creates a new shell with project environment variables",
	Action: func(c *cli.Context) error {

		ensureValidProjectFolder(".")

		pData := common.ReadProjectData(".")
		sBox := sandbox.ReadSandboxData(pData.Sandbox)

		prjJavaHome := sandbox.GetDistroJdkPath(sBox.Distro)
		prjXpHome := sandbox.GetSandboxHomePath(pData.Sandbox)
		os.Setenv(common.ENV_JAVA_HOME, prjJavaHome)
		os.Setenv(common.ENV_XP_HOME, prjXpHome)

		cmd := createNewShellCommand()
		err := cmd.Start()
		util.Fatal(err, "Could not start new shell")

		fmt.Fprintf(os.Stderr, "Started new project shell with PID %d.\nType 'exit' to close it.\n", cmd.Process.Pid)

		cmd.Wait()
		fmt.Fprintln(os.Stderr, "Project shell has finished.")

		return nil
	},
}

func createNewShellCommand() *exec.Cmd {
	var cmd *exec.Cmd

	switch util.GetCurrentOs() {
	case "windows":
		cmd = exec.Command("cmd", "/K", "prompt enonic$G && enonic")
	default:
		cmd = exec.Command("bash", "-c", `bash --init-file <(echo "export PS1='enonic> ' && enonic")`)
	}

	if !util.IsCommandAvailable(cmd.Path) {
		fmt.Fprintln(os.Stderr, "Shell is not available in your system")
		os.Exit(1)
	}

	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Env = os.Environ()

	return cmd
}
