package common

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/Masterminds/semver"
	"gopkg.in/yaml.v3"
)

// SCHEMA_APP_XP_VERSION is the minimal XP version supporting schema applications
const SCHEMA_APP_XP_VERSION = "8.1.0-SNAPSHOT"

// APP_ICON_FILE is the application icon, read by XP from the jar root
const APP_ICON_FILE = "enonic.svg"

// CMS_DIR_NAME is the folder holding the schema resources; XP treats an application shipping cms/cms.yaml as the owner of its schema
const CMS_DIR_NAME = "cms"

var SCHEMA_APP_FILE_EXTENSIONS = []string{".yaml", ".yml", ".svg"}

const MAX_APP_NAME_LENGTH = 63

// APP_DESCRIPTOR_FILES lists the application descriptor file names in order of preference, all at the project root
var APP_DESCRIPTOR_FILES = []string{"enonic.yaml", "enonic.yml"}

var CMS_DESCRIPTOR_FILES = []string{"cms.yaml", "cms.yml"}

var GRADLE_WRAPPER_FILES = []string{"gradlew", "gradlew.bat"}

// AppNameRegex is the XP rule for application names (ApplicationKey), which is also the OSGi Bundle-SymbolicName of the jar
var AppNameRegex = regexp.MustCompile(`^\w+(?:\.\w+)*$`)

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

// AppDescriptor mirrors the fields accepted by XP in enonic.yaml. Unknown fields are rejected, like XP does.
type AppDescriptor struct {
	Kind        string        `yaml:"kind"`
	Name        string        `yaml:"name"`
	Title       LocalizedText `yaml:"title"`
	Description LocalizedText `yaml:"description"`
	VendorName  string        `yaml:"vendorName"`
	VendorUrl   string        `yaml:"vendorUrl"`
	Url         string        `yaml:"url"`
	Config      any           `yaml:"config"`
}

func findRegularFile(dir string, names []string) string {
	for _, name := range names {
		file := filepath.Join(dir, name)
		if stat, err := os.Stat(file); err == nil && stat.Mode().IsRegular() {
			return file
		}
	}
	return ""
}

// FindAppDescriptorFile returns the path of enonic.yaml (or enonic.yml) in the project root, or "" when absent
func FindAppDescriptorFile(prjPath string) string {
	return findRegularFile(prjPath, APP_DESCRIPTOR_FILES)
}

// FindCmsDescriptorFile returns the path of cms/cms.yaml (or cms/cms.yml) in the project, or "" when absent
func FindCmsDescriptorFile(prjPath string) string {
	return findRegularFile(filepath.Join(prjPath, CMS_DIR_NAME), CMS_DESCRIPTOR_FILES)
}

// HasGradleWrapper reports whether the project contains a gradle wrapper script for any OS
func HasGradleWrapper(prjPath string) bool {
	return findRegularFile(prjPath, GRADLE_WRAPPER_FILES) != ""
}

// IsSchemaProject reports whether the folder is a schema application: an application descriptor at the project root
// and no gradle build. Schema applications are packaged and installed by CLI itself.
func IsSchemaProject(prjPath string) bool {
	return FindAppDescriptorFile(prjPath) != "" && !HasGradleWrapper(prjPath)
}

// ReadAppDescriptor parses the application descriptor of the project. Returns nil without error when there is none.
func ReadAppDescriptor(prjPath string) (*AppDescriptor, error) {
	file := FindAppDescriptorFile(prjPath)
	if file == "" {
		return nil, nil
	}

	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("could not read '%s': %w", file, err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	var descriptor AppDescriptor
	if err := decoder.Decode(&descriptor); err != nil {
		return nil, fmt.Errorf("could not parse '%s': %w", file, err)
	}
	return &descriptor, nil
}

// ValidateAppName checks the application name against the XP rule for application keys
func ValidateAppName(name string) error {
	if name == "" {
		return fmt.Errorf("application name is missing: add 'name: \"com.example.myapp\"' to %s", APP_DESCRIPTOR_FILES[0])
	}
	if len(name) > MAX_APP_NAME_LENGTH {
		return fmt.Errorf("application name '%s' is too long (%d characters, max %d)", name, len(name), MAX_APP_NAME_LENGTH)
	}
	if !AppNameRegex.MatchString(name) {
		return fmt.Errorf("application name '%s' is not valid: it must consist of [a-zA-Z0-9_] segments separated by periods, e.g. com.example.myapp", name)
	}
	return nil
}

func SystemVersionRange(xpVersion string) (string, error) {
	ver, err := semver.NewVersion(xpVersion)
	if err != nil {
		return "", fmt.Errorf("invalid XP version '%s': %w", xpVersion, err)
	}
	return fmt.Sprintf("[%d.%d,%d)", ver.Major(), ver.Minor(), ver.Major()+1), nil
}
