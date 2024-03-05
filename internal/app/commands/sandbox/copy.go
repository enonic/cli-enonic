package sandbox

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"fmt"
	copy2 "github.com/otiai10/copy"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"os"
	"path/filepath"
	"strings"
)

var Copy = cli.Command{
	Name:      "copy",
	Aliases:   []string{"cp"},
	Usage:     "Create a copy of a sandbox with all content.",
	ArgsUsage: "<source> <target>",
	Flags: []cli.Flag{
		common.FORCE_FLAG,
	},
	Action: func(c *cli.Context) error {

		var (
			originalName string
			targetName   string
		)
		if c.NArg() > 0 {
			originalName = c.Args().First()
			targetName = c.Args().Get(1)
		}
		sandbox, _ := EnsureSandboxExists(c, EnsureSandboxOptions{
			Name:               originalName,
			SelectBoxMessage:   "Select sandbox to copy",
			ShowSuccessMessage: true,
		})
		if sandbox == nil {
			os.Exit(1)
		}

		targetName = ensureTargetFlag(targetName, common.IsForceMode(c))

		rData := common.ReadRuntimeData()
		// stop if it's currently running before copying
		if rData.Running == sandbox.Name && !AskToStopSandbox(rData, common.IsForceMode(c)) {
			os.Exit(1)
		}

		if err := copy2.Copy(filepath.Join(getSandboxesDir(), sandbox.Name), filepath.Join(getSandboxesDir(), targetName)); err != nil {
			fmt.Fprintf(os.Stdout, "Error copying '%s' to '%s': %s", sandbox.Name, targetName, err.Error())
			os.Exit(1)
		}

		fmt.Fprintf(os.Stdout, "Sandbox '%s' copied to '%s'.\n", sandbox.Name, targetName)

		return nil
	},
}

func ensureTargetFlag(name string, isForce bool) string {
	nameValidator := func(val interface{}) error {
		str := val.(string)
		if len(strings.TrimSpace(str)) == 0 {
			if isForce {
				fmt.Fprintln(os.Stderr, "Target name can not be empty in non-interactive mode.")
				os.Exit(1)
			}
			return errors.New("Sandbox name can not be empty: ")
		} else if Exists(str) {
			if isForce {
				fmt.Fprintf(os.Stderr, "Sandbox with name '%s' already exists.\n", str)
				os.Exit(1)
			}
			return errors.Errorf("Sandbox with name '%s' already exists", str)
		} else {
			return nil
		}
	}
	return util.PromptString("Enter target sandbox name", name, "", nameValidator)
}
