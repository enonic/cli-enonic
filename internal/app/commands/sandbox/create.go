package sandbox

import (
	"github.com/urfave/cli"
	"github.com/enonic/enonic-cli/internal/app/util"
	"strings"
	"fmt"
	"os"
	"github.com/pkg/errors"
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

		SandboxCreateWizard(name, c.String("version"), true)

		return nil
	},
}

func SandboxCreateWizard(name, versionStr string, showSuccessMessage bool) *Sandbox {

	name = ensureUniqueNameArg(name)
	version := ensureVersionCorrect(versionStr)

	box := createSandbox(name, version)
	if showSuccessMessage {
		fmt.Fprintf(os.Stderr, "\nSandbox '%s' created with distro '%s'.\n", box.Name, box.Distro)
	}

	return box
}

func ensureUniqueNameArg(name string) string {
	existingBoxes := listSandboxes()
	defaultSandboxName := getFirstValidSandboxName(existingBoxes)

	var sandboxValidator = func(val interface{}) error {
		str := val.(string)
		if len(strings.TrimSpace(str)) < 3 {
			return errors.New("Sandbox name must be at least 3 characters long")
		} else {
			for _, existingBox := range existingBoxes {
				if existingBox.Name == str {
					return errors.Errorf("Sandbox with name '%s' already exists: ", str)
				}
			}
			return nil
		}
	}

	return util.PromptString("Sandbox name", name, defaultSandboxName, sandboxValidator)
}

func getFirstValidSandboxName(sandboxes []*Sandbox) string {
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
