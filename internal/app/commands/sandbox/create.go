package sandbox

import (
	"fmt"
	"github.com/enonic/cli-enonic/internal/app/util"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"os"
	"strings"
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
		cli.BoolFlag{
			Name:  "all, a",
			Usage: "List all distro versions.",
		},
	},
	Action: func(c *cli.Context) error {

		var name string
		if c.NArg() > 0 {
			name = c.Args().First()
		}

		SandboxCreateWizard(name, c.String("version"), "", c.Bool("all"), true)

		return nil
	},
}

func SandboxCreateWizard(name, versionStr, minDistroVersion string, includeUnstable, showSuccessMessage bool) *Sandbox {

	name = ensureUniqueNameArg(name, minDistroVersion)
	version := ensureVersionCorrect(versionStr, minDistroVersion, includeUnstable)

	box := createSandbox(name, version)

	distroPath, _ := EnsureDistroExists(box.Distro)
	CopyHomeFolder(distroPath, box.Name)

	if showSuccessMessage {
		fmt.Fprintf(os.Stdout, "\nSandbox '%s' created with distro '%s'.\n", box.Name, box.Distro)
	}

	return box
}

func ensureUniqueNameArg(name, minDistroVersion string) string {
	existingBoxes := listSandboxes(minDistroVersion)
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
