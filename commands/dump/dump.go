package dump

import (
	"github.com/urfave/cli"
)

func All() []cli.Command {
	return []cli.Command{
		New,
		Upgrade,
		Load,
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
