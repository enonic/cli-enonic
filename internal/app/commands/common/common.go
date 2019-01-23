package common

import (
	"io"
	"net/http"
	"github.com/enonic/xp-cli/internal/app/util"
	"fmt"
	"os"
	"encoding/json"
	"github.com/urfave/cli"
	"time"
	"strings"
	"github.com/enonic/xp-cli/internal/app/commands/remote"
	"net/url"
)

var FLAGS = []cli.Flag{
	cli.StringFlag{
		Name:  "auth, a",
		Usage: "Authentication token for basic authentication (user:password)",
	},
}

func EnsureAuth(authString string) (string, string) {
	var splitAuth []string
	util.PromptUntilTrue(authString, func(val *string, ind byte) string {
		if len(strings.TrimSpace(*val)) == 0 {
			switch ind {
			case 0:
				return "Enter authentication token (<user>:<password>): "
			default:
				return "Authentication token can not be empty (<user>:<password>): "
			}
		} else {
			splitAuth = strings.Split(*val, ":")
			if len(splitAuth) != 2 {
				return fmt.Sprintf("Authentication token '%s' must have the following format <user>:<password>: ", *val)
			} else {
				return ""
			}
		}
	})
	return splitAuth[0], splitAuth[1]
}

func CreateRequest(c *cli.Context, method, url string, body io.Reader) *http.Request {
	auth := c.String("auth")
	if auth == "" {
		activeRemote := remote.GetActiveRemote()
		if activeRemote.User != "" || activeRemote.Pass != "" {
			auth = fmt.Sprintf("%s:%s", activeRemote.User, activeRemote.Pass)
		}
	}
	user, pass := EnsureAuth(auth)

	return doCreateRequest(method, url, user, pass, body)
}

func doCreateRequest(method, reqUrl, user, pass string, body io.Reader) *http.Request {
	var (
		host, scheme, port, path string
	)

	parsedUrl, err := url.Parse(reqUrl)
	util.Fatal(err, "Not a valid url: "+reqUrl)

	if parsedUrl.IsAbs() {
		host = parsedUrl.Hostname()
		port = parsedUrl.Port()
		scheme = parsedUrl.Scheme
		path = parsedUrl.Path
	} else {
		activeRemote := remote.GetActiveRemote()
		host = activeRemote.Url.Hostname()
		port = activeRemote.Url.Port()
		scheme = activeRemote.Url.Scheme

		runeUrl := []rune(reqUrl)
		if runeUrl[0] == '/' {
			// absolute path
			path = reqUrl
		} else {
			// relative path
			path = activeRemote.Url.Path + "/" + reqUrl
		}
		// remove leading slash
		path = path[1:]
	}

	req, err := http.NewRequest(method, fmt.Sprintf("%s://%s:%s/%s", scheme, host, port, path), body)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Params error: ", err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(user, pass)
	return req
}

func SendRequest(req *http.Request) *http.Response {
	client := &http.Client{
		Timeout: time.Duration(5 * time.Minute),
	}
	resp, err := client.Do(req)

	if err != nil {
		fmt.Fprintln(os.Stderr, "Request error: ", err)
		os.Exit(1)
	}
	return resp
}

func ParseResponse(resp *http.Response, target interface{}) {
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	if resp.StatusCode == http.StatusOK {
		if err := decoder.Decode(target); err != nil {
			fmt.Fprint(os.Stderr, "Error parsing response ", err)
			os.Exit(1)
		}
	} else {
		var enonicError EnonicError
		if err := decoder.Decode(&enonicError); err == nil && enonicError.Message != "" {
			fmt.Fprintf(os.Stderr, "%d %s\n", enonicError.Status, enonicError.Message)
		} else {
			fmt.Fprintln(os.Stderr, resp.Status)
		}
		os.Exit(1)
	}
}

type EnonicError struct {
	Status  uint16 `json:status`
	Message string `json:message`
	Context struct {
		Authenticated bool     `json:authenticated`
		Principals    []string `json:principals`
	} `json:context`
}
