package env

import (
	"cli-enonic/internal/app/commands/remote"
	"fmt"
	"os"
	"strings"
)

// Activate makes the environment that arg names the active one and returns it as a target. arg is an
// environment name or a management url. An environment it does not name yet is added first: a name as
// an Enonic Cloud instance, like 'enonic env add <name>', a url under the first label of its host
// without a trailing -mgmt
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
			return nil, exitErrorf("Invalid environment name '%s'. Use lowercase letters, digits, '-' and '_', not starting with '%s'.", name, remote.ONE_OFF_TOKEN_PREFIX)
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

// nameFromUrl is the first label of the url's host without a trailing -mgmt: https://acme-mgmt.enonic.cloud is acme
func nameFromUrl(u *remote.MarshalledUrl) string {
	label := strings.ToLower(strings.SplitN(u.Hostname(), ".", 2)[0])
	return strings.TrimSuffix(label, "-mgmt")
}
