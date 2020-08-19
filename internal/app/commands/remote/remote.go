package remote

import (
	"bytes"
	"github.com/enonic/cli-enonic/internal/app/util"
	"github.com/urfave/cli"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const DEFAULT_REMOTE_NAME = "default"
const DEFAULT_REMOTE_URL = "http://localhost:4848"
const CLI_REMOTE_URL = "ENONIC_CLI_REMOTE_URL"
const CLI_REMOTE_USER = "ENONIC_CLI_REMOTE_USER"
const CLI_REMOTE_PASS = "ENONIC_CLI_REMOTE_PASS"
const CLI_REMOTE_PROXY = "ENONIC_CLI_HTTP_PROXY"

func All() []cli.Command {
	// enable if file-based remotes are used
	// ensureDefaultRemoteExists()

	return []cli.Command{
		Add,
		Remove,
		Set,
		List,
	}
}

type MarshalledUrl struct {
	url.URL
}

func ParseMarshalledUrl(text string) (*MarshalledUrl, error) {
	var (
		err       error
		parsedUrl *url.URL
	)
	if parsedUrl, err = url.ParseRequestURI(text); err != nil {
		return nil, err
	}
	return &MarshalledUrl{*parsedUrl}, err
}

func (r *MarshalledUrl) UnmarshalText(text []byte) error {
	var (
		err       error
		parsedUrl *url.URL
	)
	parsedUrl, err = url.Parse(string(text))
	r.URL = *parsedUrl
	return err
}

func (r *MarshalledUrl) MarshalText() ([]byte, error) {
	var (
		err error
		buf bytes.Buffer
	)
	r.User = nil
	_, err = buf.WriteString(r.String())
	return buf.Bytes(), err
}

type RemoteData struct {
	Url   *MarshalledUrl
	User  string
	Pass  string
	Proxy *MarshalledUrl
}

type RemotesData struct {
	Active  string
	Remotes map[string]RemoteData
}

func readRemotesData() RemotesData {
	path := filepath.Join(util.GetHomeDir(), ".enonic", ".enonic")
	file := util.OpenOrCreateDataFile(path, true)
	defer file.Close()

	var data RemotesData
	util.DecodeTomlFile(file, &data)
	if data.Remotes == nil {
		data.Remotes = make(map[string]RemoteData)
	}
	return data
}

func writeRemotesData(data RemotesData) {
	path := filepath.Join(util.GetHomeDir(), ".enonic", ".enonic")
	file := util.OpenOrCreateDataFile(path, false)
	defer file.Close()

	util.EncodeTomlFile(file, data)
}

func getRemoteByName(name string, remotes map[string]RemoteData) (*RemoteData, bool) {
	if remotes == nil {
		return nil, false
	}
	rm, prs := remotes[name]
	return &rm, prs
}

/*
Env vars remote implementation
*/
func GetActiveRemote() *RemoteData {
	remoteUrl := parseUrl(os.Getenv(CLI_REMOTE_URL), DEFAULT_REMOTE_URL)
	user := os.Getenv(CLI_REMOTE_USER)
	pass := os.Getenv(CLI_REMOTE_PASS)
	proxyUrl := parseUrl(os.Getenv(CLI_REMOTE_PROXY), "")
	return &RemoteData{remoteUrl, user, pass, proxyUrl}
}

func parseUrl(urlString string, defaultUrl string) *MarshalledUrl {
	if strings.TrimSpace(urlString) == "" {
		urlString = defaultUrl
	} else if strings.Index(urlString, "http") != 0 {
		urlString = "http://" + urlString
	}
	parsedUrl, err := ParseMarshalledUrl(urlString)
	if err != nil {
		parsedUrl, err = ParseMarshalledUrl(defaultUrl)
	}
	return parsedUrl
}

/*
File-based remotes implementation

func GetActiveRemote() *RemoteData {
	data := readRemotesData()
	active, ok := data.Remotes[data.Active]
	if !ok {
		fmt.Fprintf(os.Stderr, "Could not load active remote '%s'", data.Active)
		os.Exit(0)
	}
	return &active
}
*/

func ensureDefaultRemoteExists() {
	data := readRemotesData()
	defaultUrl, _ := ParseMarshalledUrl(DEFAULT_REMOTE_URL)
	if remote, exists := getRemoteByName(DEFAULT_REMOTE_NAME, data.Remotes); !exists || remote.Url != defaultUrl || data.Active == "" {
		if !exists || remote.Url != defaultUrl {
			data.Remotes[DEFAULT_REMOTE_NAME] = RemoteData{defaultUrl, "", "", nil}
		}
		if data.Active == "" {
			data.Active = DEFAULT_REMOTE_NAME
		}
		writeRemotesData(data)
	}
}
