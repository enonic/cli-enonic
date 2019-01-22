package sandbox

import (
	"github.com/urfave/cli"
	"github.com/enonic/xp-cli/internal/app/util"
	"strings"
	"fmt"
	"os"
)

var Create = cli.Command{
	Name:  "create",
	Usage: "Create a new sandbox.",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "version, v",
			Usage: "Use specific distro version.",
		},
	},
	Action: func(c *cli.Context) error {

		var name string
		if c.NArg() > 0 {
			name = c.Args().First()
		}

		SandboxCreateWizard(name, c.String("version"))

		return nil
	},
}

func SandboxCreateWizard(name, versionStr string) Sandbox {

	name = ensureUniqueNameArg(name)
	version := ensureVersionCorrect(versionStr)

	box := createSandbox(name, version)
	fmt.Fprintf(os.Stderr, "Sandbox '%s' created with distro '%s'\n", box.Name, box.Distro)

	return box
}

func ensureUniqueNameArg(name string) string {
	existingBoxes := listSandboxes()
	return util.PromptUntilTrue(name, func(val *string, i byte) string {
		length := len(strings.TrimSpace(*val))
		if length == 0 && i == 0 {
			return "Enter the name of the sandbox: "
		} else if length < 3 {
			return "Name of the sandbox must be at least 3 characters long: "
		} else {
			for _, existingBox := range existingBoxes {
				if existingBox.Name == *val {
					return fmt.Sprintf("Sandbox with the name '%s' already exists: ", *val)
				}
			}
			return ""
		}
	})
}
