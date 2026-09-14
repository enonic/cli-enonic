package common

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestIsSchemaProject(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		want  bool
	}{
		{"enonic.yaml", map[string]string{"enonic.yaml": "kind: \"Application\"\n"}, true},
		{"enonic.yml", map[string]string{"enonic.yml": "kind: \"Application\"\n"}, true},
		{"gradle wrapper wins", map[string]string{"enonic.yaml": "kind: \"Application\"\n", "gradlew": ""}, false},
		{"gradle bat wrapper wins", map[string]string{"enonic.yaml": "kind: \"Application\"\n", "gradlew.bat": ""}, false},
		{"empty folder", map[string]string{}, false},
		{"gradle only", map[string]string{"gradlew": ""}, false},
		{"legacy layout", map[string]string{"enonic/enonic.yaml": "kind: \"Application\"\n"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := writeFiles(t, tc.files)
			if got := IsSchemaProject(dir); got != tc.want {
				t.Errorf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

func TestFindCmsDescriptorFile(t *testing.T) {
	dir := writeFiles(t, map[string]string{"cms/cms.yml": "kind: \"CMS\"\n"})
	if got := FindCmsDescriptorFile(dir); got != filepath.Join(dir, "cms", "cms.yml") {
		t.Errorf("unexpected cms descriptor path %q", got)
	}
	if got := FindCmsDescriptorFile(writeFiles(t, map[string]string{"cms.yaml": ""})); got != "" {
		t.Errorf("root cms.yaml must not be recognized, got %q", got)
	}
}

func TestReadAppDescriptor(t *testing.T) {
	dir := writeFiles(t, map[string]string{"enonic.yaml": `kind: "Application"
name: "com.example.myapp"
title: "My App"
description:
  text: "Desc"
  i18n: "app.description"
vendorName: "Enonic AS"
config:
  key: "value"
`})
	descriptor, err := ReadAppDescriptor(dir)
	if err != nil {
		t.Fatal(err)
	}
	if descriptor.Name != "com.example.myapp" || descriptor.Title.Text != "My App" ||
		descriptor.Description.Text != "Desc" || descriptor.Description.I18n != "app.description" ||
		descriptor.VendorName != "Enonic AS" || descriptor.Config == nil {
		t.Errorf("unexpected descriptor %+v", descriptor)
	}

	if descriptor, err := ReadAppDescriptor(t.TempDir()); descriptor != nil || err != nil {
		t.Errorf("expected nil descriptor without error, got %+v, %v", descriptor, err)
	}

	unknown := writeFiles(t, map[string]string{"enonic.yaml": "kind: \"Application\"\ntype: \"Static\"\n"})
	if _, err := ReadAppDescriptor(unknown); err == nil || !strings.Contains(err.Error(), "type") {
		t.Errorf("expected an error naming the unknown field, got %v", err)
	}

	invalid := writeFiles(t, map[string]string{"enonic.yaml": "kind: [\n"})
	if _, err := ReadAppDescriptor(invalid); err == nil {
		t.Error("expected a parse error")
	}
}

func TestValidateAppName(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"com.example.myapp", ""},
		{"myapp", ""},
		{"my_app.v2", ""},
		{"MyApp", ""},
		{strings.Repeat("a", 63), ""},
		{"", "missing"},
		{strings.Repeat("a", 64), "too long"},
		{"com-example", "not valid"},
		{".com", "not valid"},
		{"com.", "not valid"},
		{"com..x", "not valid"},
		{"a b", "not valid"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateAppName(tc.name)
			if tc.want == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("expected error containing %q, got %v", tc.want, err)
			}
		})
	}
}

func TestReadProjectName_FallsBackToDescriptor(t *testing.T) {
	dir := writeFiles(t, map[string]string{"enonic.yaml": "kind: \"Application\"\nname: \"com.example.myapp\"\n"})
	if got := ReadProjectName(dir); got != "com.example.myapp" {
		t.Errorf("expected name from descriptor, got %q", got)
	}
	if got := ReadProjectName(writeFiles(t, map[string]string{})); got != "" {
		t.Errorf("expected empty name, got %q", got)
	}
}

func TestReadProjectDistroVersion_SchemaProject(t *testing.T) {
	dir := writeFiles(t, map[string]string{"enonic.yaml": "kind: \"Application\"\n"})
	if got := ReadProjectDistroVersion(dir); got != SCHEMA_APP_XP_VERSION {
		t.Errorf("expected %s, got %s", SCHEMA_APP_XP_VERSION, got)
	}
}
