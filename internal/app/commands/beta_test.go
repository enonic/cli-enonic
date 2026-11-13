package commands

import (
	"bytes"
	"cli-enonic/internal/app/commands/remote"
	"cli-enonic/internal/app/util"
	"os"
	"testing"

	"github.com/urfave/cli"
)

func setupBeta(t *testing.T, beta bool) {
	t.Setenv("ENONIC_CLI_HOME_PATH", t.TempDir())
	os.Unsetenv(util.SNAP_ENV_VAR)
	t.Setenv(remote.CLI_ENV, "")
	t.Setenv(remote.CLI_REMOTE_URL, "")
	if beta {
		t.Setenv(util.BETA_ENV_VAR, "true")
	} else {
		t.Setenv(util.BETA_ENV_VAR, "")
	}
	remote.SelectEnv("")
	t.Cleanup(func() { remote.SelectEnv("") })
}

func newApp(cmds []cli.Command) *cli.App {
	app := cli.NewApp()
	app.Commands = cmds
	app.Writer = &bytes.Buffer{}
	app.ErrWriter = &bytes.Buffer{}
	app.ExitErrHandler = func(*cli.Context, error) {}
	return app
}

func findCommand(cmds []cli.Command, path ...string) *cli.Command {
	for i := range cmds {
		if cmds[i].Name == path[0] {
			if len(path) == 1 {
				return &cmds[i]
			}
			return findCommand(cmds[i].Subcommands, path[1:]...)
		}
	}
	return nil
}

func TestBetaOffHidesCommands(t *testing.T) {
	setupBeta(t, false)
	cmds := All()
	for _, name := range []string{"env", "login", "logout"} {
		if cmd := findCommand(cmds, name); cmd == nil || !cmd.Hidden {
			t.Errorf("%s is not hidden", name)
		}
	}
	for _, args := range [][]string{{"env", "list"}, {"login"}, {"logout"}} {
		err := newApp(All()).Run(append([]string{"enonic"}, args...))
		if err == nil || err.Error() != util.BETA_MESSAGE {
			t.Errorf("%v: got %v", args, err)
		}
	}
}

func TestBetaOnShowsCommands(t *testing.T) {
	setupBeta(t, true)
	cmds := All()
	for _, name := range []string{"env", "login", "logout"} {
		if cmd := findCommand(cmds, name); cmd == nil || cmd.Hidden {
			t.Errorf("%s is hidden", name)
		}
	}
}

func TestEnvFlagRequiresBeta(t *testing.T) {
	setupBeta(t, false)
	cmds := All()
	ran := false
	findCommand(cmds, "app", "list").Action = func(c *cli.Context) error {
		ran = true
		return nil
	}
	err := newApp(cmds).Run([]string{"enonic", "app", "list", "--env", "prod"})
	if err == nil || err.Error() != util.BETA_MESSAGE || ran {
		t.Fatalf("got %v, ran %v", err, ran)
	}

	if err := newApp(cmds).Run([]string{"enonic", "app", "list"}); err != nil || !ran {
		t.Fatalf("without --env: got %v, ran %v", err, ran)
	}
}

func TestEnvFlagSelectsEnvironment(t *testing.T) {
	setupBeta(t, true)
	prodUrl, _ := remote.ParseMarshalledUrl("https://prod.example")
	remote.WriteEnvironments(&remote.EnvironmentsData{Environments: map[string]remote.Environment{"prod": {Url: prodUrl}}})

	var picked string
	cmds := wireEnvFlag([]cli.Command{{
		Name:  "x",
		Flags: []cli.Flag{cli.StringFlag{Name: "env"}},
		Action: func(c *cli.Context) error {
			picked = remote.GetActiveRemote().Name
			return nil
		},
	}})
	if err := newApp(cmds).Run([]string{"enonic", "x", "--env", "prod"}); err != nil || picked != "prod" {
		t.Fatalf("got %q %v", picked, err)
	}
	if err := newApp(cmds).Run([]string{"enonic", "x"}); err != nil || picked != "local" {
		t.Fatalf("without --env got %q %v", picked, err)
	}
}

func TestEnvFlagOnRemoteCommands(t *testing.T) {
	var walk func(cmds []cli.Command, path string)
	walk = func(cmds []cli.Command, path string) {
		for i := range cmds {
			name := path + " " + cmds[i].Name
			walk(cmds[i].Subcommands, name)
			if !hasFlag(&cmds[i], "auth") {
				continue
			}
			fixedUrl := name == " system info" || name == " system latest" || name == " latest"
			if hasEnv := hasFlag(&cmds[i], "env"); hasEnv == fixedUrl {
				t.Errorf("%s: has --env %v", name, hasEnv)
			}
		}
	}
	walk(All(), "")
}
