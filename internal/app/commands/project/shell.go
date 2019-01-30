package project

import (
	"github.com/urfave/cli"
	"github.com/enonic/xp-cli/internal/app/commands/common"
	"os"
	"github.com/enonic/xp-cli/internal/app/commands/sandbox"
	"os/exec"
	"github.com/enonic/xp-cli/internal/app/util"
	"fmt"
	"strings"
)

var Shell = cli.Command{
	Name:  "shell",
	Usage: "Creates a new shell with project environment variables",
	Action: func(c *cli.Context) error {

		ensureValidProjectFolder()

		pData := readProjectData()
		sBox := sandbox.ReadSandboxData(pData.Sandbox)

		cmd := createNewTerminalCommand()

		prjJavaHome := sandbox.GetDistroJdkPath(sBox.Distro)
		prjXpHome := sandbox.GetSandboxHomePath(pData.Sandbox)
		cmd.Env = os.Environ()
		for i, keyVal := range cmd.Env {
			key := strings.Split(keyVal, "=")[0]
			if key == common.ENV_JAVA_HOME {
				cmd.Env[i] = key + "=" + prjJavaHome
			} else if key == common.ENV_XP_HOME {
				cmd.Env[i] = key + "=" + prjXpHome
			}
		}

		err := cmd.Start()
		util.Fatal(err, "Could not start a new terminal")

		fmt.Fprintf(os.Stderr, "Started a new shell with PID %d.\n", cmd.Process.Pid)

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
		return exec.Command("open", "-a", "Terminal", "-n")
	default:
		return exec.Command("xterm", "-e", "cd "+prjDir+" && bash")
	}
}
