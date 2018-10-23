package common

import (
	"io"
	"net/http"
	"enonic.com/xp-cli/util"
	"strings"
	"fmt"
	"os"
	"encoding/json"
	"github.com/urfave/cli"
	"io/ioutil"
	"time"
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

func CreateRequest(c *cli.Context, method, url string, body io.Reader) *http.Request {
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

func SendRequest(req *http.Request) *http.Response {
	client := &http.Client{
		Timeout: time.Duration(3 * time.Minute),
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

func DebugResponse(resp *http.Response) {
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading response", err)
	}
	prettyBytes, err := util.PrettyPrintJSONBytes(bodyBytes)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error formatting response", err)
	}
	fmt.Fprintln(os.Stderr, string(prettyBytes))
}
