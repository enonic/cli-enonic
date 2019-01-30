package sandbox

import (
	"github.com/urfave/cli"
	"github.com/enonic/xp-cli/internal/app/util"
	"strings"
	"fmt"
	"os"
)

var SANDBOX_NAME_TPL = "Sandbox%d"

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
	fmt.Fprintf(os.Stderr, "\nSandbox '%s' created with distro '%s'.\n", box.Name, box.Distro)

	return box
}

func ensureUniqueNameArg(name string) string {
	existingBoxes := listSandboxes()
	defaultSandboxName := getFirstValidSandboxName(existingBoxes)
	return util.PromptUntilTrue(name, func(val *string, i byte) string {
		if *val == "" {
			if i == 0 {
				return fmt.Sprintf("\nSandbox name (default: '%s'):", defaultSandboxName)
			} else {
				*val = defaultSandboxName
				fmt.Fprintln(os.Stderr, *val+"\n")
				return ""
			}
		} else if len(strings.TrimSpace(*val)) < 3 {
			return "Sandbox name must be at least 3 characters long: "
		} else {
			for _, existingBox := range existingBoxes {
				if existingBox.Name == *val {
					return fmt.Sprintf("Sandbox with name '%s' already exists: ", *val)
				}
			}
			return ""
		}
	})
}
func getFirstValidSandboxName(sandboxes []Sandbox) string {
	var name string
	num := 1
	nameInvalid := false

	for ok := true; ok; ok = nameInvalid && num < 1000 {
		name = fmt.Sprintf(SANDBOX_NAME_TPL, num)
		nameInvalid = false
		for _, box := range sandboxes {
			if box.Name == name {
				num++
				nameInvalid = true
				break
			}
		}
	}

	return name
}
