package app

import (
	"bytes"
	"cli-enonic/internal/app/commands/common"
	"encoding/json"
	"fmt"
	"github.com/urfave/cli"
	"net/http"
	"os"
)

var Stop = cli.Command{
	Name:      "stop",
	Usage:     "Stop an application",
	ArgsUsage: "<app key>",
	Flags:     common.WithBetaCompat(append([]cli.Flag{common.FORCE_FLAG}, common.AUTH_AND_TLS_FLAGS...)),
	Action: func(c *cli.Context) error {

		key := ensureAppKeyArg(c)

		stopApp(c, key)

		return nil
	},
}

func stopApp(c *cli.Context, name string) {
	req := createStopRequest(c, name, common.ServerApis(c))

	res := common.SendRequest(c, req, fmt.Sprintf("Requesting stop \"%s\"", name))

	var status string
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		status = "Done"
	} else {
		status = "Error"
	}

	fmt.Fprintln(os.Stdout, status)
}

func createStopRequest(c *cli.Context, key string, server bool) *http.Request {
	body := new(bytes.Buffer)
	params := map[string]string{
		"key": key,
	}
	json.NewEncoder(body).Encode(params)
	path := "app/stop"
	if server {
		path = common.ServerApiPath("app", "stop")
	}
	req := common.CreateRequest(c, "POST", path, body)

	return req
}

type StopResult struct {
	//TODO: add body when xp#9189 is done
}
