package env

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/commands/remote"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/urfave/cli"
)

// out is where lists go, swapped in tests
var out io.Writer = os.Stdout

func All() []cli.Command {
	return []cli.Command{
		Add,
		List,
		Use,
		Current,
		Remove,
	}
}

func exitError(err error) error {
	return cli.NewExitError(err.Error(), 1)
}

func exitErrorf(format string, args ...interface{}) error {
	return cli.NewExitError(fmt.Sprintf(format, args...), 1)
}

// expectArgs shows the command's usage and fails if it was not given n arguments
func expectArgs(c *cli.Context, n int) error {
	if c.NArg() == n {
		return nil
	}
	cli.ShowCommandHelp(c, c.Command.Name)
	return cli.NewExitError("", 1)
}

func readEnvironments() (*remote.EnvironmentsData, error) {
	data, err := remote.ReadEnvironments()
	if err != nil {
		return nil, exitError(err)
	}
	return data, nil
}

func parseEnvUrl(text string) (*remote.MarshalledUrl, error) {
	parsed, err := remote.ParseMarshalledUrl(text)
	if err != nil || !remote.ValidEnvUrl(parsed) {
		return nil, exitErrorf("Invalid url '%s'.", text)
	}
	return parsed, nil
}

var Add = cli.Command{
	Name:      "add",
	Usage:     "Add an environment",
	ArgsUsage: "<name> [<management url>]",
	Description: "Without a url, the name is taken as an Enonic Cloud instance: https://<name>-mgmt.enonic.cloud. " +
		"Set " + CLOUD_DOMAIN_VAR + " to use another cloud domain, e.g. enonic-dev.cloud.",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "proxy",
			Usage: "Proxy url for requests to the environment. ENONIC_CLI_HTTP_PROXY overrides it",
		},
	},
	Action: func(c *cli.Context) error {
		if c.NArg() != 1 {
			if err := expectArgs(c, 2); err != nil {
				return err
			}
		}
		name, rawUrl := c.Args().Get(0), c.Args().Get(1)
		if !remote.ValidEnvName(name) {
			return exitErrorf("Invalid environment name '%s'. Use lowercase letters, digits, '-' and '_'.", name)
		}
		if rawUrl == "" {
			var err error
			if rawUrl, err = cloudMgmtUrl(name); err != nil {
				return err
			}
		}
		return addEnv(c, name, rawUrl)
	},
}

const CLOUD_DOMAIN_VAR = "ENONIC_CLI_CLOUD_DOMAIN"
const DEFAULT_CLOUD_DOMAIN = "enonic.cloud"

var cloudInstancePattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// cloudMgmtUrl is the management endpoint of the Enonic Cloud instance with the name
func cloudMgmtUrl(name string) (string, error) {
	if !cloudInstancePattern.MatchString(name) {
		return "", exitErrorf("'%s' is not an Enonic Cloud instance name. Give the management url: enonic env add %s <url>", name, name)
	}
	domain := strings.Trim(strings.TrimSpace(os.Getenv(CLOUD_DOMAIN_VAR)), ".")
	if domain == "" {
		domain = DEFAULT_CLOUD_DOMAIN
	}
	return fmt.Sprintf("https://%s-mgmt.%s", name, domain), nil
}

func addEnv(c *cli.Context, name, rawUrl string) error {
	data, err := readEnvironments()
	if err != nil {
		return err
	}
	if _, exists := data.Environments[name]; exists || name == remote.LOCAL_ENV {
		return exitErrorf("Environment '%s' already exists.", name)
	}
	envUrl, err := parseEnvUrl(rawUrl)
	if err != nil {
		return err
	}
	env := remote.Environment{Url: envUrl}
	if rawProxy := c.String("proxy"); rawProxy != "" {
		if env.Proxy, err = parseEnvUrl(rawProxy); err != nil {
			return err
		}
	}
	data.Environments[name] = env
	if err := remote.WriteEnvironments(data); err != nil {
		return exitErrorf("Could not save %s: %v", remote.EnvironmentsFile(), err)
	}
	fmt.Fprintf(os.Stderr, "Added environment '%s' (%s)\n", name, envUrl.String())
	return nil
}

var List = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Usage:   "List environments, marking the active one with *",
	Action: func(c *cli.Context) error {
		if err := expectArgs(c, 0); err != nil {
			return err
		}
		data, err := readEnvironments()
		if err != nil {
			return err
		}
		names := make([]string, 0, len(data.Environments))
		for name := range data.Environments {
			names = append(names, name)
		}
		sort.Strings(names)
		names = append([]string{remote.LOCAL_ENV}, names...)

		// a target that cannot be resolved, e.g. an unknown ENONIC_CLI_ENV, marks none
		var activeName string
		if active, err := remote.ResolveTarget(); err == nil {
			activeName = active.Name
		}
		width := 0
		for _, name := range names {
			width = max(width, len(name))
		}
		for _, name := range names {
			target, err := remote.EnvTarget(name, "")
			if err != nil {
				return exitError(err)
			}
			marker, suffix := " ", ""
			if name == activeName {
				marker = "*"
			}
			if common.HasAccessToken(target) {
				suffix = "  logged in"
			}
			fmt.Fprintf(out, "%s %-*s  %s%s\n", marker, width, name, target.Url.String(), suffix)
		}
		return nil
	},
}

var Use = cli.Command{
	Name:      "use",
	Usage:     "Make an environment the active one",
	ArgsUsage: "<name>",
	Action: func(c *cli.Context) error {
		if err := expectArgs(c, 1); err != nil {
			return err
		}
		name := c.Args().First()
		data, err := readEnvironments()
		if err != nil {
			return err
		}
		if _, exists := data.Environments[name]; !exists && name != remote.LOCAL_ENV {
			return exitError(remote.UnknownEnvError(name))
		}
		target, err := remote.EnvTarget(name, "")
		if err != nil {
			return exitError(err)
		}
		data.Active = name
		if name == remote.LOCAL_ENV {
			data.Active = ""
		}
		if err := remote.WriteEnvironments(data); err != nil {
			return exitErrorf("Could not save %s: %v", remote.EnvironmentsFile(), err)
		}
		fmt.Fprintf(os.Stderr, "Using environment '%s' (%s)\n", name, target.Url.String())
		return nil
	},
}

var Current = cli.Command{
	Name:  "current",
	Usage: "Show the environment commands run against, and what selected it",
	Action: func(c *cli.Context) error {
		if err := expectArgs(c, 0); err != nil {
			return err
		}
		target, err := remote.ResolveTarget()
		if err != nil {
			return exitError(err)
		}
		label := target.Name
		if label == "" {
			label = target.Url.String()
		}
		fmt.Fprintf(out, "%s (%s), selected by %s\n", label, target.Url.String(), target.Source)
		return nil
	},
}

var Remove = cli.Command{
	Name:      "remove",
	Aliases:   []string{"rm"},
	Usage:     "Remove an environment and its login",
	ArgsUsage: "<name>",
	Action: func(c *cli.Context) error {
		if err := expectArgs(c, 1); err != nil {
			return err
		}
		name := c.Args().First()
		if name == remote.LOCAL_ENV {
			return exitErrorf("The local environment cannot be removed.")
		}
		data, err := readEnvironments()
		if err != nil {
			return err
		}
		if _, exists := data.Environments[name]; !exists {
			return exitError(remote.UnknownEnvError(name))
		}
		delete(data.Environments, name)
		if data.Active == name {
			data.Active = ""
		}
		if err := remote.WriteEnvironments(data); err != nil {
			return exitErrorf("Could not save %s: %v", remote.EnvironmentsFile(), err)
		}
		if _, err := common.RemoveAccessToken(&remote.RemoteData{Name: name}); err != nil {
			return exitErrorf("Could not remove the login to '%s': %v", name, err)
		}
		fmt.Fprintf(os.Stderr, "Removed environment '%s'\n", name)
		return nil
	},
}
