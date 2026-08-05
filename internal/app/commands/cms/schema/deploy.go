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
	"gopkg.in/yaml.v3"
)

var Deploy = cli.Command{
	Name:      "deploy",
	Usage:     "Create or update schemas from a file",
	ArgsUsage: "<file>",
	Flags: append([]cli.Flag{
		cli.StringFlag{
			Name:  "namespace",
			Usage: "Namespace to deploy the schemas to",
		},
		cli.StringFlag{
			Name:  "name",
			Usage: "Schema name, single-descriptor files only (defaults to the file name without extension; 'style'/'cms' for the style/cms kinds)",
		},
		common.FORCE_FLAG,
	}, common.AUTH_AND_TLS_FLAGS...),
	Action: func(c *cli.Context) error {

		file := ensureFileArg(c)

		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not read file '%s': %v\n", file, err)
			os.Exit(1)
		}

		mode, docs := detectDeployMode(file, content)
		if mode == modeEnvelope {
			return deployMulti(c, file, docs)
		}

		k := resolveKind(file, content)
		namespace, name := resolveTarget(c, k, file)
		target := k.target(namespace, name)

		result, verb, err := deployOne(c, k, namespace, name, string(content))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failure: %s\n", err)
			os.Exit(1)
		}

		fmt.Fprintf(os.Stderr, "Schema %s '%s' %s\n", k.name, target, verb)
		if result != nil {
			fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(result))
		}

		return nil
	},
}

// deployOne creates or updates a single schema; resource is sent verbatim.
// Returns the parsed JSON response and the verb 'created' or 'updated'
func deployOne(c *cli.Context, k *kind, namespace, name, resource string) (interface{}, string, error) {
	target := k.target(namespace, name)

	body := new(bytes.Buffer)
	params := map[string]interface{}{
		"resource": resource,
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
	err := parseResponseErr(res, &result)
	return result, verb, err
}

// deployMulti deploys every envelope document of a multi-descriptor file in
// file order, continuing on per-document failures; exits 1 if any document failed
func deployMulti(c *cli.Context, file string, docs []*yaml.Node) error {
	if strings.TrimSpace(c.String("name")) != "" {
		fmt.Fprintln(os.Stderr, "--name can not be used with a multi-descriptor file: names come from the 'name' field of each document.")
		os.Exit(1)
	}
	namespace := ensureNamespace(c)

	type docResult struct {
		Kind   string      `json:"kind,omitempty"`
		Target string      `json:"target,omitempty"`
		Status string      `json:"status"`
		Error  string      `json:"error,omitempty"`
		Result interface{} `json:"result,omitempty"`
	}
	results := make([]docResult, 0, len(docs))
	seen := make(map[string]int)
	failed := 0

	for i, root := range docs {
		env, err := decodeEnvelope(root, i+1)
		var k *kind
		if err == nil {
			k, err = resolveEnvelopeKind(env)
		}
		if err == nil {
			err = validateEnvelopeName(env, k, seen)
		}
		if err != nil {
			failed++
			fmt.Fprintf(os.Stderr, "Failed: %s\n", err)
			results = append(results, docResult{Status: "failed", Error: err.Error()})
			continue
		}

		name := env.Name
		if !k.named { // the fixed name of style/cms is not part of the URL
			name = ""
		}

		target := k.target(namespace, name)
		result, verb, err := deployOne(c, k, namespace, name, env.Resource)
		if err != nil {
			failed++
			fmt.Fprintf(os.Stderr, "Schema %s '%s' failed: %s\n", k.name, target, err)
			results = append(results, docResult{Kind: k.name, Target: target, Status: "failed", Error: err.Error()})
			continue
		}

		fmt.Fprintf(os.Stderr, "Schema %s '%s' %s\n", k.name, target, verb)
		results = append(results, docResult{Kind: k.name, Target: target, Status: verb, Result: result})
	}

	fmt.Fprintf(os.Stderr, "\nDeployed %d of %d schemas from '%s'", len(results)-failed, len(results), filepath.Base(file))
	if failed > 0 {
		fmt.Fprintf(os.Stderr, ", %d failed", failed)
	}
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(results))

	if failed > 0 {
		os.Exit(1)
	}
	return nil
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

// resolveKind detects the kind from the file contents ('kind:' field of a YAML
// descriptor) or the .properties extension used by phrases bundles
func resolveKind(file string, content []byte) *kind {
	detected := detectKind(filepath.Base(file), content)
	if detected == nil {
		fmt.Fprintf(os.Stderr, "Could not detect schema kind of '%s': the file must have a 'kind:' field or a '.properties' extension.\n", file)
		os.Exit(1)
	}
	return detected
}

// resolveTarget resolves namespace and name, the file name is the fallback
// for the name of named kinds
func resolveTarget(c *cli.Context, k *kind, file string) (string, string) {
	name := strings.TrimSpace(c.String("name"))

	if !k.named {
		if name != "" && name != k.fixedName() {
			fmt.Fprintf(os.Stderr, "'%s' is a single resource per namespace: name must be '%s'.\n", k.name, k.fixedName())
			os.Exit(1)
		}
		name = "" // the fixed name is not part of the URL
	}

	if k.named && name == "" {
		base := filepath.Base(file)
		name = strings.TrimSuffix(base, filepath.Ext(base))
	}

	return ensureNamespace(c), name
}

// ensureNamespace returns the --namespace flag value, prompting if empty
func ensureNamespace(c *cli.Context) string {
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
	return strings.TrimSpace(util.PromptString("Enter namespace", strings.TrimSpace(c.String("namespace")), "", namespaceValidator))
}
