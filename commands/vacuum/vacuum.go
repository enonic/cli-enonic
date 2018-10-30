package vacuum

import (
	"github.com/urfave/cli"
	"enonic.com/xp-cli/commands/common"
	"fmt"
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

		req := common.CreateRequest(c, "POST", "api/system/vacuum", nil)

		fmt.Fprint(os.Stderr, "Vacuuming...")
		resp := common.SendRequest(req)

		var result VacuumResponse
		common.ParseResponse(resp, &result)
		fmt.Fprintf(os.Stderr, "Done %d tasks", len(result.TaskResults))

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
