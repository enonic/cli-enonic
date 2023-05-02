package dump

import (
	"cli-enonic/internal/app/util"
	"fmt"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
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

func ensureNameFlag(name string, mustNotExist, force bool) string {
	existingDumps := listExistingDumpNames()
	if len(existingDumps) == 0 && !mustNotExist {
		fmt.Fprintln(os.Stderr, "No existing dumps found")
		os.Exit(1)
	}
	var selectedOption string
	nameRegex, _ := regexp.Compile("^[a-zA-Z0-9_.]+$")
	dumpValidator := func(val interface{}) error {
		str := val.(string)
		exists := false
		if len(strings.TrimSpace(str)) == 0 {
			if mustNotExist {
				if force {
					fmt.Fprintln(os.Stderr, "Dump name can not be empty in non-interactive mode.")
					os.Exit(1)
				}
				return errors.New("Dump name can not be empty: ")
			}
		} else {
			if !nameRegex.MatchString(str) {
				if force {
					fmt.Fprintf(os.Stderr, "Dump name '%s' is not valid. Use letters, digits, dot (.) or underscore (_) only\n", str)
					os.Exit(1)
				}
				return errors.Errorf("Dump name '%s' is not valid. Use letters, digits, dot (.) or underscore (_) only: ", str)
			} else {
				lowerVal := strings.ToLower(str)
				for _, dumpName := range existingDumps {
					if strings.ToLower(dumpName) == lowerVal {
						exists = true
						break
					}
				}
			}
		}

		if mustNotExist && exists {
			if force {
				fmt.Fprintf(os.Stderr, "Dump with name '%s' already exists.\n", str)
				os.Exit(1)
			}
			return errors.Errorf("Dump with name '%s' already exists: ", str)
		} else if !mustNotExist && !exists {
			if force {
				fmt.Fprintf(os.Stderr, "Dump with name '%s' can not be found.\n", str)
				os.Exit(1)
			}
			var err error
			selectedOption, err = util.PromptSelect(&util.SelectOptions{
				Message: "Select dump",
				Options: existingDumps,
			})
			util.Fatal(err, "Could not select dump: ")

			return nil
		} else {
			return nil
		}
	}
	validatedOption := util.PromptString("Dump name", name, "", dumpValidator)

	if selectedOption != "" {
		return selectedOption
	} else {
		return validatedOption
	}
}
