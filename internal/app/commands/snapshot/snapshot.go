package snapshot

import (
	"github.com/urfave/cli"
	"time"
)

func All() []cli.Command {
	return []cli.Command{
		List,
		New,
		Restore,
		Delete,
	}
}

type Snapshot struct {
	Name      string    `json:name`
	Reason    string    `json:reason`
	State     string    `json:state`
	Timestamp time.Time `json:timestamp`
	Indices   []string  `json:indices`
}
