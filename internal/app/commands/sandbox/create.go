package sandbox

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"fmt"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"os"
	"regexp"
	"strings"
)

var SANDBOX_NAME_TPL = "Sandbox%d"

var Create = cli.Command{
	Name:      "create",
	Usage:     "Create a new sandbox.",
	ArgsUsage: "<name>",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "version, v",
			Usage: "Use specific distro version.",
		},
		cli.BoolFlag{
			Name:  "all, a",
			Usage: "List all distro versions.",
		},
		common.FORCE_FLAG,
	},
	Action: func(c *cli.Context) error {

		var name string
		if c.NArg() > 0 {
			name = c.Args().First()
		}

		SandboxCreateWizard(name, c.String("version"), "", c.Bool("all"), true, common.IsForceMode(c))

		return nil
	},
}

func SandboxCreateWizard(name, versionStr, minDistroVersion string, includeUnstable, showSuccessMessage, force bool) *Sandbox {

	name = ensureUniqueNameArg(name, minDistroVersion, force)
	version, _ := ensureVersionCorrect(versionStr, minDistroVersion, true, includeUnstable, force)

	box := createSandbox(name, version)

	distroPath, _ := EnsureDistroExists(box.Distro)
	CopyHomeFolder(distroPath, box.Name)

	if showSuccessMessage {
		fmt.Fprintf(os.Stdout, "\nSandbox '%s' created with distro '%s'.\n", box.Name, box.Distro)
	}

	return box
}

func ensureUniqueNameArg(name, minDistroVersion string, force bool) string {
	existingBoxes := listSandboxes(minDistroVersion)
	defaultSandboxName := getFirstValidSandboxName(existingBoxes)

	nameRegex, _ := regexp.Compile("^[a-zA-Z0-9_]+$")
	var sandboxValidator = func(val interface{}) error {
		str := val.(string)
		if len(strings.TrimSpace(str)) < 3 {
			if force {
				// Assume defaultSandboxName in force mode
				return nil
			}
			return errors.New("Sandbox name must be at least 3 characters long: ")
		} else {
			if !nameRegex.MatchString(str) {
				if force {
					fmt.Fprintf(os.Stderr, "Sandbox name '%s' is not valid. Use letters, digits or underscore (_) only\n", str)
					os.Exit(1)
				}
				return errors.Errorf("Sandbox name '%s' is not valid. Use letters, digits or underscore (_) only: ", str)
			} else {
				lowerStr := strings.ToLower(str)
				for _, existingBox := range existingBoxes {
					if strings.ToLower(existingBox.Name) == lowerStr {
						if force {
							fmt.Fprintf(os.Stderr, "Sandbox with name '%s' already exists\n", str)
							os.Exit(1)
						}
						return errors.Errorf("Sandbox with name '%s' already exists: ", str)
					}
				}
			}
			return nil
		}
	}

	userSandboxName := util.PromptString("Sandbox name", name, defaultSandboxName, sandboxValidator)
	if !force || userSandboxName != "" {
		return userSandboxName
	} else {
		return defaultSandboxName
	}
}

func getFirstValidSandboxName(sandboxes []*Sandbox) string {
	var name string
	num := 1
	nameInvalid := false

	for ok := true; ok; ok = nameInvalid && num < 1000 {
		name = fmt.Sprintf(SANDBOX_NAME_TPL, num)
		nameInvalid = false
		for _, box := range sandboxes {
			if strings.ToLower(box.Name) == strings.ToLower(name) {
				num++
				nameInvalid = true
				break
			}
		}
	}

	return name
}
