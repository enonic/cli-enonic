package commands

import (
	"cli-enonic/internal/app/commands/remote"
	"strings"

	"github.com/urfave/cli"
)

// without leaves out the named top-level commands: beta commands do not exist unless beta is on
func without(cmds []cli.Command, names ...string) []cli.Command {
	kept := make([]cli.Command, 0, len(cmds))
	for _, cmd := range cmds {
		beta := false
		for _, name := range names {
			beta = beta || cmd.Name == name
		}
		if !beta {
			kept = append(kept, cmd)
		}
	}
	return kept
}

// withoutEnvFlag leaves the beta --env flag out of every command
func withoutEnvFlag(cmds []cli.Command) []cli.Command {
	for i := range cmds {
		cmds[i].Subcommands = withoutEnvFlag(cmds[i].Subcommands)
		flags := make([]cli.Flag, 0, len(cmds[i].Flags))
		for _, flag := range cmds[i].Flags {
			if flag.GetName() != "env" {
				flags = append(flags, flag)
			}
		}
		cmds[i].Flags = flags
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

// wireEnvFlag makes every command with the --env flag select that environment before it runs
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
			remote.SelectEnv(c.String("env"))
			if original != nil {
				return original(c)
			}
			return nil
		}
	}
	return cmds
}
