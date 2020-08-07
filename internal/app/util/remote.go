package util

import (
	"bytes"
	"net/url"
	"os"
	"strings"
)

const DEFAULT_REMOTE_URL = "http://localhost:4848"
const CLI_REMOTE_URL = "ENONIC_CLI_REMOTE_URL"
const CLI_REMOTE_USER = "ENONIC_CLI_REMOTE_USER"
const CLI_REMOTE_PASS = "ENONIC_CLI_REMOTE_PASS"
const CLI_REMOTE_PROXY = "ENONIC_CLI_HTTP_PROXY"

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
