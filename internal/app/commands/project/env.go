package project

import (
	"fmt"
	"github.com/enonic/cli-enonic/internal/app/commands/common"
	"github.com/enonic/cli-enonic/internal/app/commands/sandbox"
	"github.com/enonic/cli-enonic/internal/app/util"
	"github.com/urfave/cli"
	"os"
)

var Env = cli.Command{
	Name:  "env",
	Usage: "Exports enonic environment variables as string to be used in any third-party shell",
	Flags: []cli.Flag{common.FORCE_FLAG},
	Action: func(c *cli.Context) error {

		ensureValidProjectFolder(".")

		pData := common.ReadProjectData(".")
		sBox := sandbox.ReadSandboxData(pData.Sandbox)

		prjJavaHome := sandbox.GetDistroJdkPath(sBox.Distro)
		prjXpHome := sandbox.GetSandboxHomePath(pData.Sandbox)

		var exportStr string
		switch util.GetCurrentOs() {
		case "windows":
			exportStr = fmt.Sprintf("\n# Exports enonic environment variables as string to be used in any third-party shell\n"+
				"# Usage in cmd: enonic project env > tmpFile && set /p myvar= < tmpFile && del tmpFile && %%myvar%%\n\n"+
				"set XP_HOME=\"%s\" & set JAVA_HOME=\"%s\"", prjXpHome, prjJavaHome)
		default:
			exportStr = fmt.Sprintf("\n`# Exports enonic environment variables as string to be used in any third-party shell`\n"+
				"`# Usage in bash terminal: eval $(enonic project env)`\n\n"+
				"export XP_HOME=\"%s\"\n"+
				"JAVA_HOME=\"%s\"\n", prjXpHome, prjJavaHome)
		}

		_, err := os.Stdout.Write([]byte(exportStr))
		util.Fatal(err, "Error writing to command output")

		return nil
	},
}
