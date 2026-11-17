package commands

import (
	"cli-enonic/internal/app/commands/remote"
	"cli-enonic/internal/app/util"
	"strings"

	"github.com/urfave/cli"
)

func betaError() error {
	return cli.NewExitError(util.BETA_MESSAGE, 1)
}

// gateBeta hides the named top-level commands unless beta is on, and refuses to run them
func gateBeta(cmds []cli.Command, names ...string) []cli.Command {
	for i := range cmds {
		for _, name := range names {
			if cmds[i].Name != name {
				continue
			}
			cmds[i].Hidden = !util.IsBeta()
			original := cmds[i].Before
			cmds[i].Before = func(c *cli.Context) error {
				if !util.IsBeta() {
					return betaError()
				}
				if original != nil {
					return original(c)
				}
				return nil
			}
		}
	}
	return cmds
}

func hasFlag(cmd *cli.Command, name string) bool {
	for _, flag := range cmd.Flags {
		for _, flagName := range flagNames(flag) {
			if flagName == name {
				return true
			}
		}
	}
	return false
}

// flagNames are the flag's name and aliases, from "name, alias"
func flagNames(flag cli.Flag) []string {
	var names []string
	for _, part := range strings.Split(flag.GetName(), ",") {
		if part = strings.TrimSpace(part); part != "" {
			names = append(names, part)
		}
	}
	return names
}

// wireEnvFlag makes every command with the --env flag select that environment before it runs, and refuse
// the flag unless beta is on
func wireEnvFlag(cmds []cli.Command) []cli.Command {
	for i := range cmds {
		if len(cmds[i].Subcommands) > 0 {
			cmds[i].Subcommands = wireEnvFlag(cmds[i].Subcommands)
		}
		if !hasFlag(&cmds[i], "env") {
			continue
		}
		original := cmds[i].Before
		cmds[i].Before = func(c *cli.Context) error {
			if c.IsSet("env") && !util.IsBeta() {
				return betaError()
			}
			remote.SelectEnv(c.String("env"))
			if original != nil {
				return original(c)
			}
			return nil
		}
	}
	return cmds
}
