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
)

var FLAGS = []cli.Flag{
	cli.StringFlag{
		Name:  "auth, a",
		Usage: "Authentication token for basic authentication (user:password)",
	},
	cli.StringFlag{
		Name:  "host",
		Value: "localhost",
		Usage: "Host name for server",
	},
	cli.StringFlag{
		Name:  "port, p",
		Value: "8080",
		Usage: "Port number for server",
	},
	cli.StringFlag{
		Name:  "scheme, s",
		Value: "http",
		Usage: "Scheme",
	},
}

func EnsureAuth(c *cli.Context) (string, string) {
	var splitAuth []string
	util.PromptUntilTrue(c.String("auth"), func(val string, ind byte) string {
		if len(strings.TrimSpace(val)) == 0 {
			switch ind {
			case 0:
				return "Enter authentication token (<user>:<password>): "
			default:
				return "Authentication token can not be empty (<user>:<password>): "
			}
		} else {
			splitAuth = strings.Split(val, ":")
			if len(splitAuth) != 2 {
				return fmt.Sprintf("Authentication token '%s' must have the following format <user>:<password>: ", val)
			} else {
				return ""
			}
		}
	})
	return splitAuth[0], splitAuth[1]
}

func CreateRequest(c *cli.Context, method, url string, body io.Reader) *http.Request {
	host := c.String("host")
	port := c.String("port")
	scheme := c.String("scheme")
	user, pass := EnsureAuth(c)

	req, err := http.NewRequest(method, fmt.Sprintf("%s://%s:%s/%s", scheme, host, port, url), body)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Params error: ", err)
		os.Exit(1)
	}
	req.SetBasicAuth(user, pass)
	req.Header.Set("Content-Type", "application/json")
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
