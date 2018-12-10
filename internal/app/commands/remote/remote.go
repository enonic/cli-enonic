package remote

import (
	"github.com/urfave/cli"
	"net/url"
	"github.com/enonic/xp-cli/internal/app/util"
	"path/filepath"
	"bytes"
)

func All() []cli.Command {
	ensureDefaultRemoteExists()

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
	Url  *MarshalledUrl
	User string
	Pass string
}

type RemotesData struct {
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

func ensureDefaultRemoteExists() {
	data := readRemotesData()
	defaultUrl, _ := ParseMarshalledUrl("http://localhost:8080")
	if remote, exists := getRemoteByName("default", data.Remotes); !exists || remote.Url != defaultUrl {
		if !exists {
			remote = &RemoteData{defaultUrl, "", ""}
		} else {
			remote.Url = defaultUrl
		}
		data.Remotes["default"] = *remote
		writeRemotesData(data)
	}
}
