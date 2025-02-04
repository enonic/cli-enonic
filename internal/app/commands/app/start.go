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

var Start = cli.Command{
	Name:      "start",
	Usage:     "Start an application",
	Flags:     append([]cli.Flag{}, common.AUTH_FLAG, common.CRED_FILE_FLAG, common.FORCE_FLAG),
	ArgsUsage: "<app key>",
	Action: func(c *cli.Context) error {

		key := ensureAppKeyArg(c)

		startApp(c, key)

		return nil
	},
}

func startApp(c *cli.Context, name string) {
	req := createStartRequest(c, name)

	res := common.SendRequest(c, req, fmt.Sprintf("Requesting start \"%s\"", name))

	var status string
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		status = "Done"
	} else {
		status = "Error"
	}

	fmt.Fprintln(os.Stdout, status)
}

func createStartRequest(c *cli.Context, key string) *http.Request {
	body := new(bytes.Buffer)
	params := map[string]string{
		"key": key,
	}
	json.NewEncoder(body).Encode(params)
	req := common.CreateRequest(c, "POST", "app/start", body)

	return req
}

type StartResult struct {
	//TODO: add body when xp#9189 is done
}
