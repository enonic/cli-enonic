package dump

import (
	"cli-enonic/internal/app/util"
	"fmt"
	"github.com/urfave/cli"
	"gopkg.in/AlecAivazis/survey.v1"
	"os"
	"regexp"
	"strings"
)

func All() []cli.Command {
	return []cli.Command{
		Create,
		Upgrade,
		Load,
		List,
	}
}

func ensureNameFlag(name string, mustNotExist bool) string {
	existingDumps := listExistingDumpNames()
	if len(existingDumps) == 0 && !mustNotExist {
		fmt.Fprintln(os.Stderr, "No existing dumps found")
		os.Exit(1)
	}

	nameRegex, _ := regexp.Compile("^[a-zA-Z0-9_.]+$")

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
				return fmt.Sprintf("Dump name '%s' is not valid. Use letters, digits, dot (.) or underscore (_) only: ", *val)
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
