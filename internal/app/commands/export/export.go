package export

import (
	"github.com/urfave/cli"
	"github.com/enonic/enonic-cli/internal/app/util"
	"strings"
	"fmt"
)

func All() []cli.Command {
	return []cli.Command{
		New,
		Load,
	}
}

func ensureNameFlag(c *cli.Context) {
	if c.String("t") == "" {

		var name string
		name = util.PromptUntilTrue(name, func(val *string, ind byte) string {
			if len(strings.TrimSpace(*val)) == 0 {
				switch ind {
				case 0:
					return "Enter target name: "
				default:
					return "Target name can not be empty: "
				}
			} else {
				return ""
			}
		})

		c.Set("t", name)
	}
}

func ensurePathFlag(c *cli.Context) {
	var path = c.String("path")

	path = util.PromptUntilTrue(path, func(val *string, ind byte) string {
		if len(strings.TrimSpace(*val)) == 0 {
			switch ind {
			case 0:
				return "Enter source repo path (<repo-name>:<branch-name>:<node-path>): "
			default:
				return "Source repo path can not be empty (<repo-name>:<branch-name>:<node-path>): "
			}
		} else {
			splitPathLen := len(strings.Split(*val, ":"))
			if splitPathLen != 3 {
				return fmt.Sprintf("Source repo path '%s' must have the following format <repo-name>:<branch-name>:<node-path>: ", *val)
			} else {
				return ""
			}
		}
	})

	c.Set("path", path)
}
