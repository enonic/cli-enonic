package schema

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var List = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Usage:   "List schemas in a namespace",
	Flags: append([]cli.Flag{
		cli.StringFlag{
			Name:  "namespace",
			Usage: "Namespace to list schemas of",
		},
		KIND_FLAG,
		common.FORCE_FLAG,
	}, common.AUTH_AND_TLS_FLAGS...),
	Action: func(c *cli.Context) error {

		namespace := ensureNamespaceFlag(c)

		if kindName := strings.TrimSpace(c.String("kind")); kindName != "" {
			k := findKind(kindName)
			if k == nil {
				fmt.Fprintf(os.Stderr, "Unknown schema kind '%s'. Must be one of: %s.\n", kindName, strings.Join(kindNames(), ", "))
				os.Exit(1)
			}

			req := common.CreateRequest(c, "GET", k.url(namespace, ""), nil)
			res := common.SendRequest(c, req, "Loading")

			var result interface{}
			parseResponse(res, &result)

			fmt.Fprintln(os.Stderr, "Done")
			fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(result))
			return nil
		}

		results := make(map[string]interface{})
		for _, k := range kinds {
			req := common.CreateRequest(c, "GET", k.url(namespace, ""), nil)
			res := common.SendRequest(c, req, fmt.Sprintf("Loading %s", k.name))
			if !k.named && res.StatusCode == http.StatusNotFound {
				res.Body.Close()
				continue
			}

			var result interface{}
			parseResponse(res, &result)
			if result != nil {
				results[k.name] = result
			}
		}

		fmt.Fprintln(os.Stderr, "Done")
		fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(results))
		return nil
	},
}

func ensureNamespaceFlag(c *cli.Context) string {
	namespaceValidator := func(val interface{}) error {
		str := strings.TrimSpace(val.(string))
		var message string
		if str == "" {
			message = "Namespace can not be empty"
		} else if strings.Contains(str, ":") {
			message = fmt.Sprintf("Namespace '%s' must not include a schema name, use 'get' to fetch a single schema", str)
		} else {
			return nil
		}
		if common.IsForceMode(c) {
			fmt.Fprintln(os.Stderr, message+".")
			os.Exit(1)
		}
		return errors.New(message + ": ")
	}

	return strings.TrimSpace(util.PromptString("Enter namespace", c.String("namespace"), "", namespaceValidator))
}
