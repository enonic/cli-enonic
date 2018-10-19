package snapshot

import (
	"github.com/urfave/cli"
	"net/http"
	"fmt"
	"strings"
	"enonic.com/xp-cli/util"
	"io"
	"os"
	"encoding/json"
	"time"
)

func All() []cli.Command {
	return []cli.Command{
		List,
		New,
		Restore,
		Delete,
	}
}

var SNAPSHOT_FLAGS = []cli.Flag{
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

func createRequest(c *cli.Context, method, url string, body io.Reader) *http.Request {
	auth := c.String("auth")
	host := c.String("host")
	port := c.String("port")
	scheme := c.String("scheme")
	var splitAuth []string

	auth = util.PromptUntilTrue(auth, func(val string, ind byte) string {
		if val == "" {
			return "Enter authentication token (user:password): "
		} else {
			splitAuth = strings.Split(val, ":")
			if len(splitAuth) != 2 {
				return "Authentication token must have the following format `user:password`: "
			} else {
				return ""
			}
		}
	})
	c.Set("auth", auth)

	req, err := http.NewRequest(method, fmt.Sprintf("%s://%s:%s/%s", scheme, host, port, url), body)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Params error: ", err)
		os.Exit(1)
	}
	req.SetBasicAuth(splitAuth[0], splitAuth[1])
	req.Header.Set("Content-Type", "application/json")
	return req
}

func sendRequest(req *http.Request) *http.Response {
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		fmt.Fprintln(os.Stderr, "Request error: ", err)
		os.Exit(1)
	}
	return resp
}

func parseResponse(resp *http.Response, target interface{}) {
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
			fmt.Fprint(os.Stderr, "Error parsing response", err)
			os.Exit(1)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Response status %d: %s", resp.StatusCode, resp.Status)
		os.Exit(1)
	}
}

type Snapshot struct {
	Name      string    `json:name`
	Reason    string    `json:reason`
	State     string    `json:state`
	Timestamp time.Time `json:timestamp`
	Indices   []string  `json:indices`
}
