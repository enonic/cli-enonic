package common

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Masterminds/semver"
	"gopkg.in/yaml.v3"
)

const APP_DIR_NAME = "enonic"

const APP_TYPE_STATIC = "Static"
const APP_TYPE_BUNDLE = "Bundle"

const STATIC_APP_VERSION = "0.0.0"

const STATIC_APP_XP_VERSION = "8.1.0-SNAPSHOT"

var APP_DESCRIPTOR_FILES = []string{"enonic.yaml", "enonic.yml"}

type LocalizedText struct {
	Text string `yaml:"text"`
	I18n string `yaml:"i18n"`
}

func (t *LocalizedText) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		t.Text = node.Value
		t.I18n = ""
		return nil
	case yaml.MappingNode:
		type plain LocalizedText
		var value plain
		if err := node.Decode(&value); err != nil {
			return err
		}
		*t = LocalizedText(value)
		return nil
	default:
		return fmt.Errorf("line %d: localized text must be a string or an object", node.Line)
	}
}

type AppDescriptor struct {
	Kind        string        `yaml:"kind"`
	Type        string        `yaml:"type"`
	Title       LocalizedText `yaml:"title"`
	Description LocalizedText `yaml:"description"`
	VendorName  string        `yaml:"vendorName"`
	VendorUrl   string        `yaml:"vendorUrl"`
	Url         string        `yaml:"url"`
}

func (d *AppDescriptor) IsStatic() bool {
	return d != nil && d.Type == APP_TYPE_STATIC
}

func GetAppDir(prjPath string) string {
	return filepath.Join(prjPath, APP_DIR_NAME)
}

func FindAppDescriptorFile(prjPath string) string {
	appDir := GetAppDir(prjPath)
	for _, name := range APP_DESCRIPTOR_FILES {
		file := filepath.Join(appDir, name)
		if stat, err := os.Stat(file); err == nil && !stat.IsDir() {
			return file
		}
	}
	return ""
}

func ReadAppDescriptor(prjPath string) (*AppDescriptor, error) {
	file := FindAppDescriptorFile(prjPath)
	if file == "" {
		return nil, nil
	}

	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("could not read '%s': %w", file, err)
	}

	var descriptor AppDescriptor
	if err := yaml.Unmarshal(data, &descriptor); err != nil {
		return nil, fmt.Errorf("could not parse '%s': %w", file, err)
	}
	return &descriptor, nil
}

func IsStaticProject(prjPath string) bool {
	descriptor, err := ReadAppDescriptor(prjPath)
	if err != nil {
		return false
	}
	return descriptor.IsStatic()
}

func SystemVersionRange(xpVersion string) (string, error) {
	ver, err := semver.NewVersion(xpVersion)
	if err != nil {
		return "", fmt.Errorf("invalid XP version '%s': %w", xpVersion, err)
	}
	return fmt.Sprintf("[%d.%d,%d)", ver.Major(), ver.Minor(), ver.Major()+1), nil
}
