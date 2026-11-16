package repo

import (
	"bytes"
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"net/http"
	"os"
	"strconv"
)

var Replicas = cli.Command{
	Name:  "replicas",
	Usage: "Set the number of replicas in the cluster.",
	Flags: append([]cli.Flag{common.FORCE_FLAG}, common.AUTH_AND_TLS_FLAGS...),
	Action: func(c *cli.Context) error {

		replicasNum := ensureReplicasNumberArg(c)
		message := fmt.Sprintf("Setting replicas number to %d", replicasNum)

		var result ReplicasResponse
		if common.ServerApis(c) {
			result.Repositories = map[string]map[string]string{}
			for _, repository := range listRepositories(c, "Loading repositories").Repositories {
				res := common.SendRequest(c, createReplicasRequest(c, repository.Id, replicasNum), fmt.Sprintf("%s in %s", message, repository.Id))
				var replicas IndexReplicas
				common.ParseResponse(res, &replicas)
				result.Repositories[repository.Id] = replicas.Replicas
			}
		} else {
			res := common.SendRequest(c, createReprocessRequest(c, replicasNum), message)
			common.ParseResponse(res, &result)
		}
		fmt.Fprintln(os.Stderr, "Done")
		fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(result))

		return nil
	},
}

func createReprocessRequest(c *cli.Context, replicasNum int) *http.Request {
	body := new(bytes.Buffer)
	params := map[string]interface{}{
		"settings": map[string]interface{}{
			"index": map[string]interface{}{
				"number_of_replicas": replicasNum,
			},
		},
	}
	json.NewEncoder(body).Encode(params)

	return common.CreateRequest(c, "POST", "repo/index/updateSettings", body)
}

// createReplicasRequest sets the replicas of a repository's search and storage indices
func createReplicasRequest(c *cli.Context, repository string, replicasNum int) *http.Request {
	body := new(bytes.Buffer)
	count := strconv.Itoa(replicasNum)
	json.NewEncoder(body).Encode(IndexReplicas{Replicas: map[string]string{"search": count, "storage": count}})
	return common.CreateRequest(c, "PUT", common.ServerApiPath("index", repository), body)
}

func ensureReplicasNumberArg(c *cli.Context) int {
	var numValidator = func(val interface{}) error {
		if val == "" {
			return errors.New("Number of replicas can not be empty: ")
		} else {
			num, err := strconv.Atoi(val.(string))
			if err != nil || num < 0 || num > 99 {
				return errors.Errorf("Not a valid number of replicas '%s'. Use whole numbers from 0 to 99: ", val)
			}
		}
		return nil
	}
	// There will be no errors here because validator above has made sure it's a number already
	result, _ := strconv.Atoi(util.PromptString("Enter number of replicas (0-99)", c.Args().First(), "0", numValidator))
	return result
}

type ReplicasResponse struct {
	UpdatedIndexes []string                     `json:"updatedIndexes,omitempty"`
	Repositories   map[string]map[string]string `json:"repositories,omitempty"`
}

type IndexReplicas struct {
	Replicas map[string]string `json:"replicas"`
}
