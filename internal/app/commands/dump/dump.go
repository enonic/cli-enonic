package dump

import (
	"github.com/urfave/cli"
	"github.com/enonic/enonic-cli/internal/app/util"
	"io/ioutil"
	"path/filepath"
	"fmt"
	"github.com/AlecAivazis/survey"
	"strings"
	"github.com/enonic/enonic-cli/internal/app/commands/sandbox"
	"os"
	"regexp"
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
	if len(existingDumps) == 0 && !mustNotExist {
		fmt.Fprintln(os.Stderr, "No existing dumps found")
		os.Exit(0)
	}

	nameRegex, _ := regexp.Compile("^[a-zA-Z0-9_]+$")

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
			if !nameRegex.MatchString(*val) {
				return fmt.Sprintf("Dump name '%s' is not valid. Use letters, digits and underscore (_) only: ", *val)
			} else {
				lowerVal := strings.ToLower(*val)
				for _, dumpName := range existingDumps {
					if strings.ToLower(dumpName) == lowerVal {
						exists = true
						break
					}
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
