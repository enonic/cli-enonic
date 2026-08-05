package schema

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

func All() []cli.Command {
	return []cli.Command{
		List,
		Get,
		Deploy,
		Delete,
	}
}

type kind struct {
	name     string // CLI value, e.g. "content-type"
	yamlKind string // value of the 'kind' field in a descriptor, e.g. "ContentType"; empty if descriptors of this kind have none
	category string // URL segment after /server:schema/
	typed    bool   // category is followed by a kind segment (schemas, components)
	named    bool   // resources of this kind have a name
}

var kinds = []kind{
	{"content-type", "ContentType", "schemas", true, true},
	{"form-fragment", "FormFragment", "schemas", true, true},
	{"mixin", "Mixin", "schemas", true, true},
	{"part", "Part", "components", true, true},
	{"layout", "Layout", "components", true, true},
	{"page", "Page", "components", true, true},
	{"macro", "Macro", "macros", false, true},
	{"style", "Style", "styles", false, false},
	{"cms", "CMS", "cms", false, false},
	{"phrases", "", "phrases", false, true},
}

var KEY_FLAG = cli.StringFlag{
	Name:  "key",
	Usage: "Schema key '<namespace>:<name>' ('<namespace>' for style and cms)",
}

var KIND_FLAG = cli.StringFlag{
	Name:  "kind",
	Usage: fmt.Sprintf("Schema kind (%s)", strings.Join(kindNames(), ", ")),
}

func kindNames() []string {
	names := make([]string, len(kinds))
	for i, k := range kinds {
		names[i] = k.name
	}
	return names
}

func findKind(name string) *kind {
	for i, k := range kinds {
		if k.name == name {
			return &kinds[i]
		}
	}
	return nil
}

func findKindByYamlKind(yamlKind string) *kind {
	for i, k := range kinds {
		if k.yamlKind != "" && strings.EqualFold(k.yamlKind, yamlKind) {
			return &kinds[i]
		}
	}
	return nil
}

func (k kind) url(namespace, name string) string {
	segments := []string{"/server:schema", k.category, url.PathEscape(namespace)}
	if k.typed {
		segments = append(segments, k.name)
	}
	if name != "" {
		segments = append(segments, url.PathEscape(name))
	}
	return strings.Join(segments, "/")
}

// fixedName is the only accepted name of unnamed kinds (single resource per
// namespace): 'style' and 'cms'. The name is informational and not part of the key
func (k kind) fixedName() string {
	return strings.ToLower(k.yamlKind)
}

// target is the human readable id of a schema used in messages
func (k kind) target(namespace, name string) string {
	if name == "" {
		return namespace
	}
	return namespace + ":" + name
}

var kindFieldRegex = regexp.MustCompile(`(?m)^kind:\s*["']?([A-Za-z]+)["']?`)

// detectKind resolves the kind from file contents ('kind:' field of a YAML
// descriptor) or the .properties extension used by phrases bundles
func detectKind(fileName string, content []byte) *kind {
	if strings.HasSuffix(strings.ToLower(fileName), ".properties") {
		return findKind("phrases")
	}
	if match := kindFieldRegex.FindSubmatch(content); match != nil {
		return findKindByYamlKind(string(match[1]))
	}
	return nil
}

func ensureKindFlag(c *cli.Context) *kind {
	kindValidator := func(val interface{}) error {
		str := val.(string)
		if findKind(strings.TrimSpace(str)) == nil {
			if common.IsForceMode(c) {
				fmt.Fprintf(os.Stderr, "Schema kind must be one of: %s.\n", strings.Join(kindNames(), ", "))
				os.Exit(1)
			}
			return errors.Errorf("Schema kind must be one of [%s]: ", strings.Join(kindNames(), ", "))
		}
		return nil
	}

	value := util.PromptString("Enter schema kind", c.String("kind"), "", kindValidator)
	return findKind(strings.TrimSpace(value))
}

// parseKey splits '<namespace>[:<name>]' into namespace and name
func parseKey(key string) (string, string) {
	if index := strings.Index(key, ":"); index != -1 {
		return key[:index], key[index+1:]
	}
	return key, ""
}

func ensureKeyFlag(c *cli.Context, k *kind) (string, string) {
	usage := "<namespace>:<name>"
	if !k.named {
		usage = "<namespace>"
	}

	keyValidator := func(val interface{}) error {
		namespace, name := parseKey(strings.TrimSpace(val.(string)))
		var message string
		switch {
		case namespace == "":
			message = fmt.Sprintf("Schema key '%s' can not be empty", usage)
		case k.named && name == "":
			message = fmt.Sprintf("%s key must include a name '%s'", k.name, usage)
		case !k.named && name != "":
			message = fmt.Sprintf("%s key must not include a name '%s'", k.name, usage)
		default:
			return nil
		}
		if common.IsForceMode(c) {
			fmt.Fprintln(os.Stderr, message+" in non-interactive mode.")
			os.Exit(1)
		}
		return errors.New(message + ": ")
	}

	key := util.PromptString(fmt.Sprintf("Enter schema key (%s)", usage), c.String("key"), "", keyValidator)
	return parseKey(strings.TrimSpace(key))
}

// parseResponseErr accepts any 2xx status as success, unlike common.ParseResponse
// which requires 200. A body-less success (e.g. 204 No Content) leaves target unset.
// Failures are returned instead of exiting, for callers that must continue after
// a failed request (multi-descriptor deploy).
func parseResponseErr(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if err := decoder.Decode(target); err != nil && err != io.EOF {
			return errors.Errorf("Error parsing response %v", err)
		}
		return nil
	}
	var enonicError common.EnonicError
	if err := decoder.Decode(&enonicError); err == nil && enonicError.Message != "" {
		return errors.New(enonicError.Message)
	}
	return errors.New(resp.Status)
}

// parseResponse is the exiting variant of parseResponseErr used by single-request commands
func parseResponse(resp *http.Response, target interface{}) {
	if err := parseResponseErr(resp, target); err != nil {
		fmt.Fprintf(os.Stderr, "Failure: %s\n", err)
		os.Exit(1)
	}
}
