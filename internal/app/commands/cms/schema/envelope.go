package schema

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type deployMode int

const (
	modeLegacy deployMode = iota
	modeEnvelope
)

// envelope is one document of a multi-descriptor deploy file
type envelope struct {
	Name     string `yaml:"name"`
	Kind     string `yaml:"kind"`
	Resource string `yaml:"resource"`
	index    int    // 1-based document index used in messages
}

// parseEnvelopeDocs decodes content as multi-document YAML. It returns the
// document root nodes decoded before the first error and that error (nil if
// none); empty documents are skipped
func parseEnvelopeDocs(content []byte) ([]*yaml.Node, error) {
	var docs []*yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	for {
		node := new(yaml.Node)
		err := decoder.Decode(node)
		if err == io.EOF {
			return docs, nil
		}
		if err != nil {
			return docs, err
		}
		root := node
		if root.Kind == yaml.DocumentNode && len(root.Content) == 1 {
			root = root.Content[0]
		}
		if root.Kind == 0 || root.Kind == yaml.ScalarNode && root.Tag == "!!null" {
			continue
		}
		docs = append(docs, root)
	}
}

// isEnvelopeNode reports whether a document root is envelope shaped: a mapping
// with a top-level 'resource' key holding a string. Descriptors themselves
// never have one, which makes it a reliable format discriminator
func isEnvelopeNode(root *yaml.Node) bool {
	if root == nil || root.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		key, value := root.Content[i], root.Content[i+1]
		if key.Value == "resource" && value.Kind == yaml.ScalarNode && value.Tag == "!!str" {
			return true
		}
	}
	return false
}

// resourceFieldRegex signals envelope intent in files that fail to parse as
// YAML; scanner errors can surface before any document is decoded, so decoded
// documents alone can not tell an envelope file from a legacy descriptor
var resourceFieldRegex = regexp.MustCompile(`(?m)^resource:`)

// classifyDeployFile decides whether a file is a legacy single descriptor sent
// as-is or a multi-descriptor envelope file. Content that is not valid YAML is
// treated as legacy (the server validates descriptors) unless the
// multi-descriptor intent is clear: several documents, an envelope-shaped
// first document or a top-level 'resource:' field
func classifyDeployFile(fileName string, content []byte) (deployMode, []*yaml.Node, error) {
	if strings.HasSuffix(strings.ToLower(fileName), ".properties") {
		return modeLegacy, nil, nil
	}
	docs, err := parseEnvelopeDocs(content)
	if err != nil {
		if len(docs) >= 2 || len(docs) == 1 && isEnvelopeNode(docs[0]) || resourceFieldRegex.Match(content) {
			return modeEnvelope, nil, errors.Errorf("invalid YAML: %v", err)
		}
		return modeLegacy, nil, nil
	}
	if len(docs) >= 2 || len(docs) == 1 && isEnvelopeNode(docs[0]) {
		return modeEnvelope, docs, nil
	}
	return modeLegacy, nil, nil
}

// detectDeployMode classifies the file, exiting on a malformed multi-descriptor file
func detectDeployMode(file string, content []byte) (deployMode, []*yaml.Node) {
	mode, docs, err := classifyDeployFile(filepath.Base(file), content)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not parse multi-descriptor file '%s': %v\n", file, err)
		os.Exit(1)
	}
	return mode, docs
}

// decodeEnvelope validates and converts one document node (index is 1-based) into an envelope
func decodeEnvelope(root *yaml.Node, index int) (*envelope, error) {
	if root.Kind != yaml.MappingNode {
		return nil, errors.Errorf("document %d is not a schema envelope: expected a mapping with 'name' and 'resource' fields", index)
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		if key := root.Content[i].Value; key != "name" && key != "kind" && key != "resource" {
			return nil, errors.Errorf("document %d has an unknown field '%s': allowed fields are name, kind and resource", index, key)
		}
	}
	env := envelope{index: index}
	if err := root.Decode(&env); err != nil {
		return nil, errors.Errorf("document %d is not a schema envelope: %v", index, err)
	}
	env.Name = strings.TrimSpace(env.Name)
	env.Kind = strings.TrimSpace(env.Kind)
	if strings.TrimSpace(env.Resource) == "" {
		return nil, errors.Errorf("document %d must have a non-empty 'resource' field", index)
	}
	return &env, nil
}

// resolveEnvelopeKind resolves the kind of one envelope from its explicit
// 'kind' field and/or the 'kind:' field inside the resource. An explicit kind
// is required when the resource has none (e.g. phrases)
func resolveEnvelopeKind(env *envelope) (*kind, error) {
	detected := detectKind("", []byte(env.Resource))
	if env.Kind == "" {
		if detected == nil {
			return nil, errors.Errorf("document %d: could not detect schema kind; add a 'kind' field to the envelope (%s)", env.index, strings.Join(kindNames(), ", "))
		}
		return detected, nil
	}
	explicit := findKind(strings.ToLower(env.Kind))
	if explicit == nil {
		explicit = findKindByYamlKind(env.Kind)
	}
	if explicit == nil {
		return nil, errors.Errorf("document %d has an unknown kind '%s': must be one of %s", env.index, env.Kind, strings.Join(kindNames(), ", "))
	}
	if detected != nil && detected != explicit {
		return nil, errors.Errorf("document %d: envelope kind '%s' conflicts with 'kind: %s' in the resource", env.index, env.Kind, detected.yamlKind)
	}
	return explicit, nil
}

// validateEnvelopeName checks the envelope name against the kind naming rules
// and (kind, name) uniqueness within the file
func validateEnvelopeName(env *envelope, k *kind, seen map[string]int) error {
	if !k.named && env.Name != k.fixedName() {
		return errors.Errorf("document %d: name of a '%s' must be '%s'", env.index, k.name, k.fixedName())
	}
	if k.named && env.Name == "" {
		return errors.Errorf("document %d must have a 'name' field for kind '%s'", env.index, k.name)
	}
	key := k.name + ":" + env.Name
	if first, exists := seen[key]; exists {
		return errors.Errorf("document %d: duplicate %s '%s' already deployed by document %d", env.index, k.name, env.Name, first)
	}
	seen[key] = env.index
	return nil
}
