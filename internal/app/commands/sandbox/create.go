package sandbox

import (
	"bytes"
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var TEMPLATE_FILE = "enonic-xp.template"
var SANDBOX_NAME_TPL = "Sandbox%d"
var MARKET_TEMPLATES_REQUEST = `{
	market {
		query(query: "type = 'com.enonic.app.market:solution-template'", sort: "_manualOrderValue desc") {
			name: _name
			displayName
			... on com_enonic_app_market_SolutionTemplate {
				data {
					description
					applications {
						application {
							displayName
							... on com_enonic_app_market_Application {
								data {
									groupId
									artifactId
								}
							}
						}
						appConfig {
							config
						}
					}
				}
			}
		}
	}
}`

var Create = cli.Command{
	Name:      "create",
	Usage:     "Create a new sandbox.",
	ArgsUsage: "<name>",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "template, t",
			Usage: "Use specific template name.",
		},
		cli.BoolFlag{
			Name:  "skip-template",
			Usage: "Don't use a template.",
		},
		cli.StringFlag{
			Name:  "version, v",
			Usage: "Use specific distro version.",
		},
		cli.BoolFlag{
			Name:  "all",
			Usage: "List all distro versions.",
		},
		cli.BoolFlag{
			Name:   "dev",
			Usage:  "Run Enonic XP distribution in development mode",
			Hidden: true,
		},
		cli.BoolFlag{
			Name:  "prod",
			Usage: "Run Enonic XP distribution in non-development mode",
		},
		cli.BoolFlag{
			Name:  "skip-start",
			Usage: "Don't ask to start sandbox after creation",
		},
		common.FORCE_FLAG,
	},
	Action: func(c *cli.Context) error {

		var name string
		if c.NArg() > 0 {
			name = c.Args().First()
		}
		sbox := SandboxCreateWizard(c, name, c.String("version"), "", c.Bool("all"), true, common.IsForceMode(c))

		if !c.Bool("skip-start") {
			AskToStartSandbox(c, sbox.Name)
		}

		return nil
	},
}

func promptTemplate(c *cli.Context, force bool) *Template {
	if c.Bool("skip-template") {
		return nil
	}

	tplFlag := c.String("template")
	if force && tplFlag == "" {
		fmt.Fprint(os.Stderr, "Parameter 'template' can not be empty in non-interactive mode\n\n")
		os.Exit(1)
	}

	templates := fetchTemplates(c)
	if len(templates) == 0 {
		return nil
	}

	if tplFlag != "" {
		for i, template := range templates {
			if template.Name == tplFlag || template.DisplayName == tplFlag {
				fmt.Fprintf(os.Stderr, "Using template '%s'\n", template.DisplayName)
				return &templates[i]
			}
		}
		if force {
			fmt.Fprintf(os.Stderr, "Could not find template '%s'\n\n", tplFlag)
			os.Exit(1)
		}
	}

	var selectOptions []string
	for _, temp := range templates {
		selectOptions = append(selectOptions, temp.DisplayName)
	}

	_, index, err := util.PromptSelect(&util.SelectOptions{
		Message:  "Select template",
		Options:  selectOptions,
		PageSize: len(selectOptions),
	})
	util.Fatal(err, "Could not select template: ")

	return &templates[index]
}

func addTemplateToSandbox(box *Sandbox, template *Template) {
	boxPath := GetSandboxHomePath(box.Name)
	configDir := createFolderIfNotExist(boxPath, "config")

	var appsJson []interface{}
	for _, app := range template.Data.Applications {
		appsJson = append(appsJson, map[string]interface{}{
			"key":    app.Application.Data.GroupId + "." + app.Application.Data.ArifactId,
			"config": app.AppConfig.Config,
		})
	}

	if len(appsJson) > 0 {
		templateFile := util.OpenOrCreateDataFile(filepath.Join(configDir, TEMPLATE_FILE), false)
		defer templateFile.Close()

		err := json.NewEncoder(templateFile).Encode(appsJson)
		util.Warn(err, "Could not write template to sandbox: ")
	}
}

func SandboxCreateWizard(c *cli.Context, name, versionStr, minDistroVersion string, includeUnstable, showSuccessMessage,
	force bool) *Sandbox {
	fmt.Fprint(os.Stderr, "\n")

	template := promptTemplate(c, force)

	name = ensureUniqueNameArg(name, minDistroVersion, force)
	version, _ := ensureVersionCorrect(c, versionStr, minDistroVersion, true, includeUnstable, force)

	box := createSandbox(name, version)

	distroPath, _ := EnsureDistroExists(c, box.Distro)
	CopyHomeFolder(distroPath, box.Name)

	if showSuccessMessage {
		fmt.Fprintf(os.Stdout, "\nSandbox '%s' created with distro '%s'.\n\n", box.Name, box.Distro)
	}

	if template != nil {
		addTemplateToSandbox(box, template)
	}

	return box
}

func ensureUniqueNameArg(name, minDistroVersion string, force bool) string {
	existingBoxes := listSandboxes(minDistroVersion)
	defaultSandboxName := getFirstValidSandboxName(existingBoxes)

	nameRegex, _ := regexp.Compile("^[a-zA-Z0-9_]+$")
	var sandboxValidator = func(val interface{}) error {
		str := val.(string)
		if len(strings.TrimSpace(str)) < 3 {
			if force {
				// Assume defaultSandboxName in force mode
				return nil
			}
			return errors.New("Sandbox name must be at least 3 characters long: ")
		} else {
			if !nameRegex.MatchString(str) {
				if force {
					fmt.Fprintf(os.Stderr, "Sandbox name '%s' is not valid. Use letters, digits or underscore (_) only\n", str)
					os.Exit(1)
				}
				return errors.Errorf("Sandbox name '%s' is not valid. Use letters, digits or underscore (_) only: ", str)
			} else {
				lowerStr := strings.ToLower(str)
				for _, existingBox := range existingBoxes {
					if strings.ToLower(existingBox.Name) == lowerStr {
						if force {
							fmt.Fprintf(os.Stderr, "Sandbox with name '%s' already exists\n", str)
							os.Exit(1)
						}
						return errors.Errorf("Sandbox with name '%s' already exists: ", str)
					}
				}
			}
			return nil
		}
	}

	userSandboxName := util.PromptString("Sandbox name", name, defaultSandboxName, sandboxValidator)
	if !force || userSandboxName != "" {
		return userSandboxName
	} else {
		return defaultSandboxName
	}
}

func getFirstValidSandboxName(sandboxes []*Sandbox) string {
	var name string
	num := 1
	nameInvalid := false

	for ok := true; ok; ok = nameInvalid && num < 1000 {
		name = fmt.Sprintf(SANDBOX_NAME_TPL, num)
		nameInvalid = false
		for _, box := range sandboxes {
			if strings.ToLower(box.Name) == strings.ToLower(name) {
				num++
				nameInvalid = true
				break
			}
		}
	}

	return name
}

func fetchTemplates(c *cli.Context) []Template {
	body := new(bytes.Buffer)
	params := map[string]string{
		"query": MARKET_TEMPLATES_REQUEST,
	}
	json.NewEncoder(body).Encode(params)

	req := common.CreateRequest(c, "POST", common.MARKET_URL, body)
	res, err := common.SendRequestCustom(c, req, "Loading templates from Enonic Market", 1)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error, check your internet connection.")
		return []Template{}
	}

	var result common.MarketResponse[Template]
	common.ParseResponse(res, &result)

	fmt.Fprintln(os.Stderr, "Done.")
	return result.Data.Market.Query
}

type Application struct {
	DisplayName string `json:"displayName"`
	Data        struct {
		GroupId   string `json:"groupId"`
		ArifactId string `json:"artifactId"`
	} `json:"data"`
}

type AppConfig struct {
	Config string `json:"config"`
}

type Template struct {
	Name        string
	DisplayName string
	Data        struct {
		Description  string
		Applications []struct {
			Application Application `json:"application"`
			AppConfig   AppConfig   `json:"appConfig"`
		}
	}
}
