package vacuum

import (
	"fmt"
	"github.com/enonic/cli-enonic/internal/app/commands/common"
	"github.com/enonic/cli-enonic/internal/app/util"
	"github.com/urfave/cli"
	"os"
)

var Vacuum = cli.Command{
	Name:  "vacuum",
	Usage: "Removes unused blobs and binaries from blobstore",
	Flags: append([]cli.Flag{
		cli.StringFlag{
			Name:  "repo, r",
			Usage: "The name of the repository to restore",
		},
		cli.StringFlag{
			Name:  "snapshot, snap",
			Usage: "The name of the snapshot to restore",
		},
	}, common.FLAGS...),
	Action: func(c *cli.Context) error {

		req := common.CreateRequest(c, "POST", "system/vacuum", nil)

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

type VacuumResponse struct {
	TaskResults []struct {
		Deleted   int32  `json:deleted`
		Failed    int32  `json:failed`
		InUse     int32  `json:inUse`
		Processed int32  `json:processed`
		TaskName  string `json:taskName`
	} `json:taskResults`
}
