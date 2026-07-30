package schema

import (
	"bytes"
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var Deploy = cli.Command{
	Name:      "deploy",
	Usage:     "Create or update a schema from a file",
	ArgsUsage: "<file>",
	Flags: append([]cli.Flag{
		KEY_FLAG,
		cli.StringFlag{
			Name:  "namespace",
			Usage: "Namespace to deploy the schema to (alternative to --key)",
		},
		cli.StringFlag{
			Name:  "name",
			Usage: "Schema name (defaults to the file name without extension)",
		},
		KIND_FLAG,
		common.FORCE_FLAG,
	}, common.AUTH_AND_TLS_FLAGS...),
	Action: func(c *cli.Context) error {

		file := ensureFileArg(c)

		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not read file '%s': %v\n", file, err)
			os.Exit(1)
		}

		k := resolveKind(c, file, content)
		namespace, name := resolveTarget(c, k, file)
		target := k.target(namespace, name)

		body := new(bytes.Buffer)
		params := map[string]interface{}{
			"resource": string(content),
		}
		json.NewEncoder(body).Encode(params)
		bodyBytes := body.Bytes()

		req := common.CreateRequest(c, "POST", k.url(namespace, name), bytes.NewBuffer(bodyBytes))
		res := common.SendRequest(c, req, fmt.Sprintf("Deploying %s '%s'", k.name, target))

		verb := "created"
		if res.StatusCode == http.StatusConflict {
			res.Body.Close()
			req = common.CreateRequest(c, "PUT", k.url(namespace, name), bytes.NewBuffer(bodyBytes))
			res = common.SendRequest(c, req, fmt.Sprintf("Updating %s '%s'", k.name, target))
			verb = "updated"
		}

		var result interface{}
		parseResponse(res, &result)

		fmt.Fprintf(os.Stderr, "Schema %s '%s' %s\n", k.name, target, verb)
		if result != nil {
			fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(result))
		}

		return nil
	},
}

func ensureFileArg(c *cli.Context) string {
	var file string
	if c.NArg() > 0 {
		file = c.Args().First()
	}

	fileValidator := func(val interface{}) error {
		str := strings.TrimSpace(val.(string))
		var message string
		if str == "" {
			message = "Schema file can not be empty"
		} else if info, err := os.Stat(str); err != nil || info.IsDir() {
			message = fmt.Sprintf("File '%s' does not exist", str)
		} else {
			return nil
		}
		if common.IsForceMode(c) {
			fmt.Fprintln(os.Stderr, message+" in non-interactive mode.")
			os.Exit(1)
		}
		return errors.New(message + ": ")
	}

	return strings.TrimSpace(util.PromptString("Enter path to a schema file", file, "", fileValidator))
}

// resolveKind detects the kind from the file and validates it against
// an explicitly provided --kind flag if any
func resolveKind(c *cli.Context, file string, content []byte) *kind {
	detected := detectKind(filepath.Base(file), content)
	explicit := strings.TrimSpace(c.String("kind"))

	if explicit != "" {
		k := findKind(explicit)
		if k == nil {
			fmt.Fprintf(os.Stderr, "Unknown schema kind '%s'. Must be one of: %s.\n", explicit, strings.Join(kindNames(), ", "))
			os.Exit(1)
		}
		if detected != nil && detected != k {
			fmt.Fprintf(os.Stderr, "Kind flag '%s' does not match kind '%s' defined in '%s'.\n", k.name, detected.name, file)
			os.Exit(1)
		}
		return k
	}

	if detected == nil {
		return ensureKindFlag(c)
	}
	return detected
}

// resolveTarget resolves namespace and name: --key wins over --namespace/--name,
// the file name is the fallback for the name of named kinds
func resolveTarget(c *cli.Context, k *kind, file string) (string, string) {
	namespace, name := parseKey(strings.TrimSpace(c.String("key")))
	if namespace == "" {
		namespace = strings.TrimSpace(c.String("namespace"))
	}
	if name == "" {
		name = strings.TrimSpace(c.String("name"))
	}

	if !k.named && name != "" {
		fmt.Fprintf(os.Stderr, "'%s' is a single resource per namespace and can not have a name '%s'.\n", k.name, name)
		os.Exit(1)
	}

	if k.named && name == "" {
		base := filepath.Base(file)
		name = strings.TrimSuffix(base, filepath.Ext(base))
	}

	namespaceValidator := func(val interface{}) error {
		if strings.TrimSpace(val.(string)) == "" {
			if common.IsForceMode(c) {
				fmt.Fprintln(os.Stderr, "Namespace can not be empty in non-interactive mode.")
				os.Exit(1)
			}
			return errors.New("Namespace can not be empty: ")
		}
		return nil
	}
	namespace = strings.TrimSpace(util.PromptString("Enter namespace", namespace, "", namespaceValidator))

	return namespace, name
}
