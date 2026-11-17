package remote

import (
	"bytes"
	"cli-enonic/internal/app/util"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

const LOCAL_ENV = "local"
const CLI_ENV = "ENONIC_CLI_ENV"

const (
	SOURCE_FLAG    = "--env"
	SOURCE_ENV_VAR = CLI_ENV
	SOURCE_URL     = CLI_REMOTE_URL
	SOURCE_SAVED   = "enonic env use"
	SOURCE_DEFAULT = "default"
)

var envNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

var cloudEnvNamePattern = regexp.MustCompile(`^[a-z0-9]+/[a-z0-9]+/[a-z0-9]+$`)

// IsCloudEnvName reports whether the name is an Enonic Cloud environment: <account>/<solution>/<environment>
func IsCloudEnvName(name string) bool {
	return cloudEnvNamePattern.MatchString(name)
}

// Environment is a named instance, kept in environments.toml
type Environment struct {
	Url   *MarshalledUrl `toml:"url"`
	Proxy *MarshalledUrl `toml:"proxy,omitempty"`
}

type EnvironmentsData struct {
	Active       string                 `toml:"active"`
	Environments map[string]Environment `toml:"environments"`
}

// selectedEnv is the value of --env on the running command
var selectedEnv string

// SelectEnv sets the environment chosen with --env, empty for none
func SelectEnv(name string) {
	selectedEnv = name
}

func EnvironmentsFile() string {
	return util.GetInEnonicDir("environments.toml")
}

// ONE_OFF_TOKEN_PREFIX starts the token file names of one-off ENONIC_CLI_REMOTE_URL logins, so no
// environment name may start with it
const ONE_OFF_TOKEN_PREFIX = "url-"

// ValidEnvName reports whether the name can name an environment: an Enonic Cloud name, or lowercase letters,
// digits, '-' and '_' not starting with ONE_OFF_TOKEN_PREFIX
func ValidEnvName(name string) bool {
	return IsCloudEnvName(name) || envNamePattern.MatchString(name) && !strings.HasPrefix(name, ONE_OFF_TOKEN_PREFIX)
}

// TokenKey is the name of an environment's login file, without .json: an Enonic Cloud name with its '/' as '-'
func TokenKey(name string) string {
	return strings.ReplaceAll(name, "/", "-")
}

// LoginClash is the other environment whose login would share a file with an environment of this name, or empty
// string if there is none
func (data *EnvironmentsData) LoginClash(name string) string {
	key := TokenKey(name)
	for other := range data.Environments {
		if other != name && TokenKey(other) == key {
			return other
		}
	}
	return ""
}

// LoginClashError tells that two environments would share a login file
func LoginClashError(name, other string) error {
	return fmt.Errorf("Environment '%s' would share its login with '%s'. Choose another name.", name, other)
}

// INVALID_ENV_NAME_HINT says what an environment name may be
const INVALID_ENV_NAME_HINT = "Use <account>/<solution>/<environment> for Enonic Cloud, or lowercase letters, digits, '-' and '_', not starting with '" + ONE_OFF_TOKEN_PREFIX + "'."

// ValidEnvUrl reports whether the url is an http(s) url with a host name, so not e.g. http://:4848
func ValidEnvUrl(u *MarshalledUrl) bool {
	return u != nil && (u.Scheme == "http" || u.Scheme == "https") && u.Hostname() != ""
}

func UnknownEnvError(name string) error {
	return fmt.Errorf("Unknown environment '%s'. Run 'enonic env list'.", name)
}

// ReadEnvironments reads environments.toml. A missing file is no environments; the file is never created
func ReadEnvironments() (*EnvironmentsData, error) {
	path := EnvironmentsFile()
	data := &EnvironmentsData{}
	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		data.Environments = make(map[string]Environment)
		return data, nil
	} else if err != nil {
		return nil, fmt.Errorf("Could not read %s: %v", path, err)
	}
	if err := toml.Unmarshal(content, data); err != nil {
		return nil, fmt.Errorf("Could not parse %s: %v", path, err)
	}
	if data.Environments == nil {
		data.Environments = make(map[string]Environment)
	}
	for name, env := range data.Environments {
		if !ValidEnvName(name) || name == LOCAL_ENV {
			return nil, fmt.Errorf("Invalid environment name '%s' in %s", name, path)
		}
		if other := data.LoginClash(name); other != "" {
			first, second := name, other
			if second < first {
				first, second = second, first
			}
			return nil, fmt.Errorf("Environments '%s' and '%s' in %s would share a login. Rename one of them.", first, second, path)
		}
		if !ValidEnvUrl(env.Url) {
			return nil, fmt.Errorf("Invalid url for environment '%s' in %s", name, path)
		}
		if env.Proxy != nil && !ValidEnvUrl(env.Proxy) {
			return nil, fmt.Errorf("Invalid proxy for environment '%s' in %s", name, path)
		}
	}
	if data.Active != "" && data.Active != LOCAL_ENV && !ValidEnvName(data.Active) {
		return nil, fmt.Errorf("Invalid active environment '%s' in %s", data.Active, path)
	}
	return data, nil
}

// WriteEnvironments replaces environments.toml atomically, with mode 0600
func WriteEnvironments(data *EnvironmentsData) error {
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(data); err != nil {
		return err
	}
	return util.WriteFileAtomic(EnvironmentsFile(), buf.Bytes(), 0600)
}

// ProxyFromEnv is the proxy in ENONIC_CLI_HTTP_PROXY, or nil
func ProxyFromEnv() *MarshalledUrl {
	return parseUrl(os.Getenv(CLI_REMOTE_PROXY), "")
}

// LocalTarget is the built-in local environment
func LocalTarget(source string) *RemoteData {
	localUrl, _ := ParseMarshalledUrl(DEFAULT_REMOTE_URL)
	return &RemoteData{
		Url:    localUrl,
		User:   os.Getenv(CLI_REMOTE_USER),
		Pass:   os.Getenv(CLI_REMOTE_PASS),
		Proxy:  ProxyFromEnv(),
		Name:   LOCAL_ENV,
		Source: source,
	}
}

// EnvTarget is the named environment as a target, or an error if it does not exist
func EnvTarget(name, source string) (*RemoteData, error) {
	if name == LOCAL_ENV {
		return LocalTarget(source), nil
	}
	data, err := ReadEnvironments()
	if err != nil {
		return nil, err
	}
	return envTargetIn(data, name, source)
}

func envTargetIn(data *EnvironmentsData, name, source string) (*RemoteData, error) {
	if name == LOCAL_ENV {
		return LocalTarget(source), nil
	}
	env, ok := data.Environments[name]
	if !ok {
		return nil, UnknownEnvError(name)
	}
	proxy := env.Proxy
	if envProxy := ProxyFromEnv(); envProxy != nil {
		proxy = envProxy
	}
	envUrl := *env.Url
	return &RemoteData{
		Url:    &envUrl,
		User:   os.Getenv(CLI_REMOTE_USER),
		Pass:   os.Getenv(CLI_REMOTE_PASS),
		Proxy:  proxy,
		Name:   name,
		Source: source,
	}, nil
}

// ResolveTarget picks the target instance with beta: --env, ENONIC_CLI_ENV, ENONIC_CLI_REMOTE_URL, the
// environment saved with 'enonic env use', then local. A name that does not exist is an error, never a
// fall through to the next step
func ResolveTarget() (*RemoteData, error) {
	if selectedEnv != "" {
		return EnvTarget(selectedEnv, SOURCE_FLAG)
	}
	if name := os.Getenv(CLI_ENV); name != "" {
		return EnvTarget(name, SOURCE_ENV_VAR)
	}
	if strings.TrimSpace(os.Getenv(CLI_REMOTE_URL)) != "" {
		target := masterRemote()
		target.Source = SOURCE_URL
		return target, nil
	}
	data, err := ReadEnvironments()
	if err != nil {
		return nil, err
	}
	if data.Active == "" || data.Active == LOCAL_ENV {
		return LocalTarget(SOURCE_DEFAULT), nil
	}
	return envTargetIn(data, data.Active, SOURCE_SAVED)
}
