package vacuum

import (
	"bytes"
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"encoding/json"
	"fmt"
	"github.com/urfave/cli"
	"net/http"
	"os"
)

var Vacuum = cli.Command{
	Name:  "vacuum",
	Usage: "Removes old version history and segments from content storage",
	Flags: append([]cli.Flag{
		cli.BoolFlag{
			Name:  "blob, b",
			Usage: "Also removes unused blobs from the blobstore",
		},
	}, common.AUTH_FLAG, common.FORCE_FLAG),
	Action: func(c *cli.Context) error {
		req := createVacuumRequest(c)

		var result VacuumResponse
		status := common.RunTask(req, "Vacuuming", &result)

		switch status.State {
		case common.TASK_FINISHED:
			fmt.Fprintf(os.Stdout, "Done %d tasks in %s\n", len(result.TaskResults), util.TimeFromNow(status.StartTime))
		case common.TASK_FAILED:
			fmt.Fprintf(os.Stdout, "Failed: %s\n", status.Progress.Info)
		}
		fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(result))

		return nil
	},
}

func createVacuumRequest(c *cli.Context) *http.Request {
	body := new(bytes.Buffer)
	params := map[string]interface{}{}
	if c.Bool("blob") {
		params["tasks"] = []string{
			"NodeBlobVacuumTask", "BinaryBlobVacuumTask", "SegmentVacuumTask", "VersionTableVacuumTask",
		}
	}
	json.NewEncoder(body).Encode(params)

	return common.CreateRequest(c, "POST", "system/vacuum", body)
}

type VacuumResponse struct {
	TaskResults []struct {
		Deleted   int32  `json:deleted`
		Failed    int32  `json:failed`
		InUse     int32  `json:inUse`
		Processed int32  `json:processed`
		TaskName  string `json:taskName`
	} `json:taskResults`
}
