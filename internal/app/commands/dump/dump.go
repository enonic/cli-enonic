package dump

import (
	"github.com/urfave/cli"
	"github.com/enonic/xp-cli/internal/app/util"
	"strings"
)

func All() []cli.Command {
	return []cli.Command{
		New,
		Upgrade,
		Load,
	}
}

func ensureNameFlag(c *cli.Context) {
	if c.String("d") == "" {

		var name string
		name = util.PromptUntilTrue(name, func(val *string, ind byte) string {
			if len(strings.TrimSpace(*val)) == 0 {
				switch ind {
				case 0:
					return "Enter dump name: "
				default:
					return "Dump name can not be empty: "
				}
			} else {
				return ""
			}
		})

		c.Set("d", name)
	}
}
