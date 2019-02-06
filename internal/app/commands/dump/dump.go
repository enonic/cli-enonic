package dump

import (
	"github.com/urfave/cli"
	"github.com/enonic/xp-cli/internal/app/util"
	"io/ioutil"
	"path/filepath"
	"fmt"
	"github.com/AlecAivazis/survey"
	"strings"
	"github.com/enonic/xp-cli/internal/app/commands/sandbox"
)

func All() []cli.Command {
	return []cli.Command{
		New,
		Upgrade,
		Load,
	}
}

func ensureNameFlag(name string, mustNotExist bool) string {
	existingDumps := listExistingDumpNames()

	return util.PromptUntilTrue(name, func(val *string, ind byte) string {

		exists := false
		if len(strings.TrimSpace(*val)) == 0 {
			if mustNotExist {
				if ind == 0 {
					return "Dump name: "
				} else {
					return "Dump name can not be empty: "
				}
			}
		} else {
			for _, dumpName := range existingDumps {
				if dumpName == *val {
					exists = true
					break
				}
			}
		}

		if mustNotExist && exists {
			return fmt.Sprintf("Dump name '%s' already exists: ", *val)
		} else if !mustNotExist && !exists {
			prompt := &survey.Select{
				Message: "Select dump",
				Options: existingDumps,
			}
			survey.AskOne(prompt, val, nil)
			return ""
		} else {
			return ""
		}
	})
}

func listExistingDumpNames() []string {
	homePath := sandbox.GetActiveHomePath()
	dumpsDir := filepath.Join(homePath, "data", "dump")
	dumps, err := ioutil.ReadDir(dumpsDir)
	if err != nil {
		return []string{}
	}

	dumpNames := make([]string, len(dumps))
	for i, dump := range dumps {
		dumpNames[i] = dump.Name()
	}
	return dumpNames
}
