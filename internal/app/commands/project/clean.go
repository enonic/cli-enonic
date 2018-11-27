package project

import (
	"github.com/urfave/cli"
	"fmt"
	"os"
	"os/exec"
	"github.com/enonic/xp-cli/internal/app/commands/sandbox"
	"github.com/enonic/xp-cli/internal/app/util"
)

var Clean = cli.Command{
	Name:  "clean",
	Usage: "Clean current project",
	Action: func(c *cli.Context) error {

		projectData := ensureProjectFolder()
		sandboxData := sandbox.ReadSandboxData(projectData.Sandbox)
		distroJdk := sandbox.GetDistroJdkPath(sandboxData.Distro)
		javaHome := fmt.Sprintf("-Dorg.gradle.java.home=%s", distroJdk)

		fmt.Fprint(os.Stderr, "Cleaning...")
		err := exec.Command("gradlew", "clean", javaHome).Run()
		util.Fatal(err, "Could not clean the project")

		fmt.Fprintln(os.Stderr, "Done")

		return nil
	},
}
