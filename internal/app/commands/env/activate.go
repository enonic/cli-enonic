package env

import (
	"cli-enonic/internal/app/commands/remote"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Activate makes the environment that arg names the active one and returns it as a target. arg is an
// environment name or a management url. An environment it does not name yet is added first: a name as
// an Enonic Cloud environment, like 'enonic env add <account>/<solution>/<environment>', a url under the
// name nameFromUrl gives it
func Activate(arg string) (*remote.RemoteData, error) {
	data, err := readEnvironments()
	if err != nil {
		return nil, err
	}

	name := arg
	var envUrl *remote.MarshalledUrl
	if strings.Contains(arg, "://") {
		if envUrl, err = parseEnvUrl(arg); err != nil {
			return nil, err
		}
		name = envNameOf(data, envUrl)
		if name == "" {
			name = nameFromUrl(envUrl)
			if !remote.ValidEnvName(name) || name == remote.LOCAL_ENV || data.Environments[name].Url != nil {
				return nil, exitErrorf("Cannot name an environment for %s. Add it with a name: enonic env add <name> %s", envUrl.String(), envUrl.String())
			}
		} else {
			envUrl = nil
		}
	} else if _, exists := data.Environments[name]; !exists && name != remote.LOCAL_ENV {
		if !remote.ValidEnvName(name) {
			return nil, exitErrorf("Invalid environment name '%s'. %s", name, remote.INVALID_ENV_NAME_HINT)
		}
		rawUrl, err := cloudMgmtUrl(name)
		if err != nil {
			return nil, err
		}
		if envUrl, err = parseEnvUrl(rawUrl); err != nil {
			return nil, err
		}
	}

	if envUrl != nil {
		if other := data.LoginClash(name); other != "" {
			return nil, exitError(remote.LoginClashError(name, other))
		}
		data.Environments[name] = remote.Environment{Url: envUrl}
		fmt.Fprintf(os.Stderr, "Added environment '%s' (%s)\n", name, envUrl.String())
	}
	data.Active = name
	if name == remote.LOCAL_ENV {
		data.Active = ""
	}
	if err := remote.WriteEnvironments(data); err != nil {
		return nil, exitErrorf("Could not save %s: %v", remote.EnvironmentsFile(), err)
	}
	target, err := remote.EnvTarget(name, remote.SOURCE_SAVED)
	if err != nil {
		return nil, exitError(err)
	}
	fmt.Fprintf(os.Stderr, "Using environment '%s' (%s)\n", name, target.Url.String())
	return target, nil
}

// envNameOf is the name of the environment with the url, local included, or empty string if there is none
func envNameOf(data *remote.EnvironmentsData, u *remote.MarshalledUrl) string {
	if sameUrl(remote.LocalTarget("").Url, u) {
		return remote.LOCAL_ENV
	}
	for name, env := range data.Environments {
		if sameUrl(env.Url, u) {
			return name
		}
	}
	return ""
}

func sameUrl(a, b *remote.MarshalledUrl) bool {
	return a != nil && b != nil && strings.TrimRight(a.String(), "/") == strings.TrimRight(b.String(), "/")
}

var cloudMgmtLabel = regexp.MustCompile(`^([a-z0-9]+)-([a-z0-9]+)-([a-z0-9]+)-mgmt$`)

// nameFromUrl names the environment of a url: an Enonic Cloud management host <account>-<solution>-<environment>-mgmt
// is <account>/<solution>/<environment>, any other host the first label without a trailing -mgmt, e.g.
// https://acme-mgmt.example.com is acme
func nameFromUrl(u *remote.MarshalledUrl) string {
	label := strings.ToLower(strings.SplitN(u.Hostname(), ".", 2)[0])
	if parts := cloudMgmtLabel.FindStringSubmatch(label); parts != nil {
		return strings.Join(parts[1:], "/")
	}
	return strings.TrimSuffix(label, "-mgmt")
}
