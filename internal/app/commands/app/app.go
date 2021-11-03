package app

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"fmt"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"os"
	"strings"
)

func All() []cli.Command {
	return []cli.Command{
		Install,
		Start,
		Stop,
	}
}

func ensureAppKeyArg(c *cli.Context) string {
	force := common.IsForceMode(c)
	keyValidator := func(val interface{}) error {
		str := val.(string)
		if len(strings.TrimSpace(str)) == 0 {
			if force {
				fmt.Fprintln(os.Stderr, "Application key can not be empty in non-interactive mode.")
				os.Exit(1)
			}
			return errors.New("Application key can not be empty")
		} else {
			return nil
		}
	}

	return util.PromptString("Enter application key", c.Args().First(), "", keyValidator)
}
