package sandbox

import (
	"github.com/urfave/cli"
	"github.com/enonic/xp-cli/internal/app/util"
	"strings"
	"fmt"
)

var New = cli.Command{
	Name:  "new",
	Usage: "Create a new sandbox.",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "version, v",
			Usage: "Use specific distro version.",
			Value: "latest",
		},
	},
	Action: func(c *cli.Context) error {

		name := ensureNameArg(c)
		ver := ensureVersionFlag(c)
		ensureDistroPresent(ver)
		createSandbox(name, ver)

		return nil
	},
}

func ensureVersionFlag(c *cli.Context) string {
	version := c.String("version")
	if version == "" {
		version = "latest"
	}
	return version
}

func ensureNameArg(c *cli.Context) string {
	var name string
	if c.NArg() > 0 {
		name = c.Args().First()
	}
	existingBoxes := ListSandboxes()
	return util.PromptUntilTrue(name, func(val string, i byte) string {
		if len(strings.TrimSpace(val)) == 0 {
			if i == 0 {
				return "Enter the name of the sandbox: "
			} else {
				return "Name of the sandbox can not be empty: "
			}
		} else {
			for _, existingBox := range existingBoxes {
				if existingBox == val {
					return fmt.Sprintf("Sandbox with the name '%s' already exists: ", val)
				}
			}
			return ""
		}
	})
}
