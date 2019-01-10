package cluster

import (
	"github.com/urfave/cli"
	"github.com/enonic/xp-cli/internal/app/commands/common"
	"fmt"
	"os"
	"net/http"
	"bytes"
	"encoding/json"
	"github.com/enonic/xp-cli/internal/app/util"
	"strconv"
)

var Replicas = cli.Command{
	Name:  "replicas",
	Usage: "Set the number of replicas in the cluster.",
	Flags: append([]cli.Flag{}, common.FLAGS...),
	Action: func(c *cli.Context) error {

		replicasNum := ensureReplicasNumberArg(c)

		req := createReprocessRequest(c, replicasNum)

		fmt.Fprintf(os.Stderr, "Setting replicas number to %d...", replicasNum)
		res := common.SendRequest(req)

		var result ReplicasResponse
		common.ParseResponse(res, &result)
		fmt.Fprintln(os.Stderr, "Done")
		fmt.Fprintln(os.Stderr, util.PrettyPrintJSON(result))

		return nil
	},
}

func createReprocessRequest(c *cli.Context, replicasNum int64) *http.Request {
	body := new(bytes.Buffer)
	params := map[string]interface{}{
		"settings": map[string]interface{}{
			"index": map[string]interface{}{
				"number_of_replicas": replicasNum,
			},
		},
	}
	json.NewEncoder(body).Encode(params)

	return common.CreateRequest(c, "POST", "api/repo/index/updateSettings", body)
}

func ensureReplicasNumberArg(c *cli.Context) int64 {
	var replicasNum int64
	util.PromptUntilTrue(c.Args().First(), func(val string, ind byte) string {
		if val == "" {
			switch ind {
			case 0:
				return "Enter number of replicas: "
			default:
				return "Number of replicas can not be empty: "
			}
		} else {
			var err error
			if replicasNum, err = strconv.ParseInt(val, 10, 32); err != nil || replicasNum < 0 {
				return fmt.Sprintf("Not a valid number of replicas '%s'. Only positive numbers allowed: ", val)
			}
			return ""
		}
	})

	return replicasNum
}

type ReplicasResponse struct {
	UpdatedIndexes []string `json:updatedIndexes`
}
