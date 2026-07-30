package schema

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli"
)

var Get = cli.Command{
	Name:  "get",
	Usage: "Fetch a schema by key",
	Flags: append([]cli.Flag{
		KEY_FLAG,
		KIND_FLAG,
		cli.StringFlag{
			Name:  "out",
			Usage: "Write the schema resource to a file instead of printing the response",
		},
		common.FORCE_FLAG,
	}, common.AUTH_AND_TLS_FLAGS...),
	Action: func(c *cli.Context) error {

		k := ensureKindFlag(c)
		namespace, name := ensureKeyFlag(c, k)
		target := k.target(namespace, name)

		req := common.CreateRequest(c, "GET", k.url(namespace, name), nil)
		res := common.SendRequest(c, req, fmt.Sprintf("Fetching %s '%s'", k.name, target))

		var result interface{}
		parseResponse(res, &result)

		fmt.Fprintln(os.Stderr, "Done")

		if out := strings.TrimSpace(c.String("out")); out != "" {
			writeResource(result, out)
			fmt.Fprintf(os.Stderr, "Schema resource written to '%s'\n", out)
		} else {
			fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(result))
		}

		return nil
	},
}

func writeResource(result interface{}, out string) {
	object, ok := result.(map[string]interface{})
	if !ok {
		fmt.Fprintln(os.Stderr, "Unexpected response format: can not extract schema resource")
		os.Exit(1)
	}
	resource, ok := object["resource"].(string)
	if !ok {
		fmt.Fprintln(os.Stderr, "Unexpected response format: can not extract schema resource")
		os.Exit(1)
	}
	if err := os.WriteFile(out, []byte(resource), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Could not write file '%s': %v\n", out, err)
		os.Exit(1)
	}
}
