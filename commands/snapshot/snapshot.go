package snapshot

import (
	"github.com/urfave/cli"
	"net/http"
	"log"
	"fmt"
	"strings"
	"enonic.com/xp-cli/util"
	"io"
	"io/ioutil"
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
		Name:  "host, t",
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

	if auth == "" {
		log.Fatal("required parameter -a is missing")
	}

	req, err := http.NewRequest(method, fmt.Sprintf("%s://%s:%s/%s", scheme, host, port, url), body)
	if err != nil {
		log.Fatal("Params error: ", err)
	}

	splitAuth := strings.Split(auth, ":")
	if len(splitAuth) != 2 {
		log.Fatal("parameter -a must have the following format `user:password`")
	} else {
		req.SetBasicAuth(splitAuth[0], splitAuth[1])
	}

	return req
}

func sendRequest(req *http.Request) *http.Response {
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		log.Fatal("Request error: ", err)
	}
	return resp
}

func parseResponse(resp *http.Response) string {
	defer resp.Body.Close()
	var text string
	if bodyBytes, err := ioutil.ReadAll(resp.Body); err != nil {
		log.Fatal("Response error: ", err)
	} else if resp.StatusCode == http.StatusOK {
		prettyBytes, err := util.PrettyPrintJSON(bodyBytes)
		if err != nil {
			prettyBytes = bodyBytes
		}
		text = string(prettyBytes)
	} else {
		text = fmt.Sprintf("Response: [%d] %s", resp.StatusCode, resp.Status)
	}
	return text
}
