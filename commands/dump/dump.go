package dump

import (
	"github.com/urfave/cli"
	"enonic.com/xp-cli/util"
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
		name = util.PromptUntilTrue(name, func(val string, ind byte) string {
			if len(strings.TrimSpace(val)) == 0 {
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

type Branch struct {
	Branch     string `json:branch`
	Successful int64  `json:successful`
	Errors []struct {
		message string `json:message`
	} `json:errors`
}

type Repo struct {
	RepositoryId string   `json:repositoryId`
	Versions     int64    `json:versions`
	Branches     []Branch `json:branches`
}

type Dump struct {
	Repositories []Repo `json:repositories`
}
