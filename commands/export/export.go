package export

import (
	"github.com/urfave/cli"
	"enonic.com/xp-cli/util"
	"strings"
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
		name = util.PromptUntilTrue(name, func(val string, ind byte) string {
			if len(strings.TrimSpace(val)) == 0 {
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
