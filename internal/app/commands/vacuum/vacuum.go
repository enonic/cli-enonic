package vacuum

import (
	"github.com/urfave/cli"
	"github.com/enonic/xp-cli/internal/app/commands/common"
	"fmt"
	"os"
	"github.com/enonic/xp-cli/internal/app/util"
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

		req := common.CreateRequest(c, "POST", "api/system/vacuum", nil)

		var result VacuumResponse
		status := common.RunTask(c, req, "Vacuuming...", &result)

		switch status.State {
		case common.TASK_FINISHED:
			fmt.Fprintf(os.Stderr, "Done %d tasks in %s", len(result.TaskResults), util.TimeFromNow(status.StartTime))
		case common.TASK_FAILED:
			fmt.Fprintf(os.Stderr, "Failed: %s", status.Progress.Info)
		}

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
