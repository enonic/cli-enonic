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
			Value: VERSION_LATEST,
		},
	},
	Action: func(c *cli.Context) error {

		name := ensureUniqueNameArg(c)
		ver := ensureVersionFlag(c)
		distroVer := parseDistroVersion(ver)
		createSandbox(name, distroVer)

		fmt.Fprintf(os.Stderr, "Sandbox '%s' created with distro '%s'\n", name, distroVer)

		return nil
	},
}

func ensureVersionFlag(c *cli.Context) string {
	version := c.String("version")
	if version == "" {
		version = VERSION_LATEST
	}
	return ensureVersionCorrect(version)
}

func ensureUniqueNameArg(c *cli.Context) string {
	var name string
	if c.NArg() > 0 {
		name = c.Args().First()
	}
	existingBoxes := listSandboxes()
	return util.PromptUntilTrue(name, func(val string, i byte) string {
		if len(strings.TrimSpace(val)) == 0 {
			if i == 0 {
				return "Enter the name of the sandbox: "
			} else {
				return "Name of the sandbox can not be empty: "
			}
		} else {
			for _, existingBox := range existingBoxes {
				if existingBox.Name == val {
					return fmt.Sprintf("Sandbox with the name '%s' already exists: ", val)
				}
			}
			return ""
		}
	})
}
