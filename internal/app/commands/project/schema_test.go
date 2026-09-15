package project

import (
	"archive/zip"
	"cli-enonic/internal/app/commands/common"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeProject creates a project folder with the given files. Keys use forward slashes; a key ending
// with "/" creates an empty directory.
func writeProject(t *testing.T, files map[string]string) string {
	t.Helper()
	prjPath := t.TempDir()
	for name, content := range files {
		path := filepath.Join(prjPath, filepath.FromSlash(name))
		if strings.HasSuffix(name, "/") {
			if err := os.MkdirAll(path, 0755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return prjPath
}

var testManifest = Manifest{{"Manifest-Version", "1.0"}}

// packProject builds the jar of the project and returns the entry names in order and their contents.
func packProject(t *testing.T, prjPath string) ([]string, map[string]string) {
	t.Helper()
	jarPath := filepath.Join(t.TempDir(), "build", "libs", "app.jar")
	if err := writeSchemaJar(prjPath, jarPath, testManifest); err != nil {
		t.Fatalf("writeSchemaJar failed: %v", err)
	}

	reader, err := zip.OpenReader(jarPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	names := make([]string, 0, len(reader.File))
	entries := make(map[string]string)
	for _, f := range reader.File {
		if _, exists := entries[f.Name]; exists {
			t.Fatalf("duplicate entry '%s'", f.Name)
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, f.Name)
		entries[f.Name] = string(data)
	}
	return names, entries
}

func assertNames(t *testing.T, got, want []string) {
	t.Helper()
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("expected entries %v, got %v", want, got)
	}
}

const minimalDescriptor = "kind: \"Application\"\nname: \"com.example.myapp\"\n"
const minimalCms = "kind: \"CMS\"\n"

func TestWriteSchemaJar_MinimalLayout(t *testing.T) {
	prjPath := writeProject(t, map[string]string{
		"enonic.yaml":  minimalDescriptor,
		"cms/cms.yaml": minimalCms,
	})
	names, entries := packProject(t, prjPath)

	assertNames(t, names, []string{MANIFEST_PATH, "enonic.yaml", "cms/", "cms/cms.yaml"})
	if entries["enonic.yaml"] != minimalDescriptor {
		t.Errorf("unexpected enonic.yaml content %q", entries["enonic.yaml"])
	}
	if entries["cms/cms.yaml"] != minimalCms {
		t.Errorf("unexpected cms/cms.yaml content %q", entries["cms/cms.yaml"])
	}
	if !strings.HasPrefix(entries[MANIFEST_PATH], "Manifest-Version: 1.0") {
		t.Errorf("unexpected manifest content %q", entries[MANIFEST_PATH])
	}
}

func TestWriteSchemaJar_IncludesIcon(t *testing.T) {
	icon := "<svg/>"
	prjPath := writeProject(t, map[string]string{
		"enonic.yaml":  minimalDescriptor,
		"enonic.svg":   icon,
		"cms/cms.yaml": minimalCms,
	})
	names, entries := packProject(t, prjPath)

	assertNames(t, names, []string{MANIFEST_PATH, "enonic.yaml", "enonic.svg", "cms/", "cms/cms.yaml"})
	if entries["enonic.svg"] != icon {
		t.Errorf("unexpected enonic.svg content %q", entries["enonic.svg"])
	}
}

func TestWriteSchemaJar_NestedCmsTree(t *testing.T) {
	prjPath := writeProject(t, map[string]string{
		"enonic.yaml":                            minimalDescriptor,
		"cms/cms.yaml":                           minimalCms,
		"cms/content-types/article/article.yaml": "kind: \"ContentType\"\n",
		"cms/content-types/article/article.svg":  "<svg/>",
		"cms/mixins/":                            "",
	})
	names, _ := packProject(t, prjPath)

	assertNames(t, names, []string{
		MANIFEST_PATH, "enonic.yaml", "cms/", "cms/cms.yaml",
		"cms/content-types/", "cms/content-types/article/",
		"cms/content-types/article/article.svg", "cms/content-types/article/article.yaml",
		"cms/mixins/",
	})
	for _, name := range names {
		if strings.Contains(name, "\\") {
			t.Errorf("entry '%s' must use forward slashes", name)
		}
	}
}

func TestWriteSchemaJar_PacksOnlySchemaFiles(t *testing.T) {
	prjPath := writeProject(t, map[string]string{
		"enonic.yaml":                                minimalDescriptor,
		"enonic.svg":                                 "<svg/>",
		"cms/cms.yaml":                               minimalCms,
		"cms/content-types/article/article.yaml":     "kind: \"ContentType\"\n",
		"cms/content-types/article/article.svg":      "<svg/>",
		"cms/mixins/address.yml":                     "kind: \"Mixin\"\n",
		"cms/.gitkeep":                               "",
		"cms/parts/.gitkeep":                         "",
		"cms/README.md":                              "# readme",
		"cms/site.json":                              "{}",
		"cms/content-types/article/icon.png":         "png",
		"cms/content-types/article/notes.txt":        "notes",
		"cms/content-types/article/article.yaml.bak": "backup",
	})
	names, _ := packProject(t, prjPath)

	assertNames(t, names, []string{
		MANIFEST_PATH, "enonic.yaml", "enonic.svg", "cms/", "cms/cms.yaml",
		"cms/content-types/", "cms/content-types/article/",
		"cms/content-types/article/article.svg", "cms/content-types/article/article.yaml",
		"cms/mixins/", "cms/mixins/address.yml",
		"cms/parts/",
	})
}

func TestIsSchemaAppFile(t *testing.T) {
	cases := map[string]bool{
		"a.yaml":                  true,
		"a.yml":                   true,
		"a.svg":                   true,
		"A.YAML":                  true,
		"a.SVG":                   true,
		"cms/cms.yaml":            true,
		"cms/parts/hero/hero.svg": true,
		".gitkeep":                false,
		"a.png":                   false,
		"a.json":                  false,
		"a.txt":                   false,
		"a.yaml.bak":              false,
		"yaml":                    false,
		"a.jpg":                   false,
	}
	for name, want := range cases {
		if got := isSchemaAppFile(name); got != want {
			t.Errorf("isSchemaAppFile(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestWriteSchemaJar_ExcludesProjectFiles(t *testing.T) {
	prjPath := writeProject(t, map[string]string{
		"enonic.yaml":          minimalDescriptor,
		"cms/cms.yaml":         minimalCms,
		"README.md":            "# readme",
		".enonic":              "sandbox = \"box\"\n",
		".git/HEAD":            "ref: refs/heads/master",
		"build/libs/old.jar":   "old",
		"gradle.properties":    "version = 1.0.0",
		"src/main/js/index.js": "",
		"other.yaml":           "kind: \"Other\"\n",
		"icon.svg":             "<svg/>",
		MANIFEST_PATH:          "Manifest-Version: 1.0\r\nBundle-SymbolicName: other\r\n\r\n",
	})
	names, entries := packProject(t, prjPath)

	assertNames(t, names, []string{MANIFEST_PATH, "enonic.yaml", "cms/", "cms/cms.yaml"})
	if strings.Contains(entries[MANIFEST_PATH], "other") {
		t.Errorf("project manifest must not be used, got %q", entries[MANIFEST_PATH])
	}
}

func TestWriteSchemaJar_UsesYmlDescriptor(t *testing.T) {
	prjPath := writeProject(t, map[string]string{
		"enonic.yml":   minimalDescriptor,
		"cms/cms.yaml": minimalCms,
	})
	names, _ := packProject(t, prjPath)

	assertNames(t, names, []string{MANIFEST_PATH, "enonic.yml", "cms/", "cms/cms.yaml"})
}

func TestWriteSchemaJar_PrefersYamlOverYml(t *testing.T) {
	prjPath := writeProject(t, map[string]string{
		"enonic.yaml":  minimalDescriptor,
		"enonic.yml":   "kind: \"Application\"\nname: \"com.example.other\"\n",
		"cms/cms.yaml": minimalCms,
	})
	names, entries := packProject(t, prjPath)

	assertNames(t, names, []string{MANIFEST_PATH, "enonic.yaml", "cms/", "cms/cms.yaml"})
	if entries["enonic.yaml"] != minimalDescriptor {
		t.Errorf("unexpected enonic.yaml content %q", entries["enonic.yaml"])
	}
}

func TestWriteSchemaJar_AcceptsCmsYml(t *testing.T) {
	prjPath := writeProject(t, map[string]string{
		"enonic.yaml": minimalDescriptor,
		"cms/cms.yml": minimalCms,
	})
	names, _ := packProject(t, prjPath)

	assertNames(t, names, []string{MANIFEST_PATH, "enonic.yaml", "cms/", "cms/cms.yml"})
}

func TestWriteSchemaJar_Errors(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{"missing cms dir", map[string]string{"enonic.yaml": minimalDescriptor}, "cms/cms.yaml is required"},
		{"missing cms descriptor", map[string]string{
			"enonic.yaml":                minimalDescriptor,
			"cms/content-types/a/a.yaml": "",
			"cms/content-types/cms.yaml": "misplaced",
			"cms/cms.yaml.txt":           "wrong extension",
		}, "cms/cms.yaml is required"},
		{"missing descriptor", map[string]string{"cms/cms.yaml": minimalCms}, "enonic.yaml not found"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prjPath := writeProject(t, tc.files)
			jarPath := filepath.Join(t.TempDir(), "app.jar")
			err := writeSchemaJar(prjPath, jarPath, testManifest)
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("expected error containing %q, got %q", tc.want, err.Error())
			}
		})
	}
}

func TestLoadSchemaApp(t *testing.T) {
	descriptorWithName := func(name string) string {
		return "kind: \"Application\"\nname: \"" + name + "\"\n"
	}
	cases := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{"valid", map[string]string{"enonic.yaml": minimalDescriptor, "cms/cms.yaml": minimalCms}, ""},
		{"valid underscore and upper case", map[string]string{"enonic.yaml": descriptorWithName("com.Example.my_app"), "cms/cms.yaml": minimalCms}, ""},
		{"missing name", map[string]string{"enonic.yaml": "kind: \"Application\"\n", "cms/cms.yaml": minimalCms}, "application name is missing"},
		{"dash", map[string]string{"enonic.yaml": descriptorWithName("com-example"), "cms/cms.yaml": minimalCms}, "is not valid"},
		{"leading dot", map[string]string{"enonic.yaml": descriptorWithName(".com"), "cms/cms.yaml": minimalCms}, "is not valid"},
		{"double dot", map[string]string{"enonic.yaml": descriptorWithName("com..x"), "cms/cms.yaml": minimalCms}, "is not valid"},
		{"trailing dot", map[string]string{"enonic.yaml": descriptorWithName("com."), "cms/cms.yaml": minimalCms}, "is not valid"},
		{"too long", map[string]string{"enonic.yaml": descriptorWithName(strings.Repeat("a", 64)), "cms/cms.yaml": minimalCms}, "is too long"},
		{"unknown field", map[string]string{"enonic.yaml": "kind: \"Application\"\nname: \"com.example.myapp\"\ntype: \"Static\"\n", "cms/cms.yaml": minimalCms}, "type"},
		{"missing cms", map[string]string{"enonic.yaml": minimalDescriptor}, "cms/cms.yaml is required"},
		{"no descriptor", map[string]string{"cms/cms.yaml": minimalCms}, "enonic.yaml not found"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prjPath := writeProject(t, tc.files)
			descriptor, err := loadSchemaApp(prjPath)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if descriptor == nil || descriptor.Name == "" {
					t.Fatalf("expected a descriptor with a name, got %+v", descriptor)
				}
				return
			}
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("expected error containing %q, got %q", tc.want, err.Error())
			}
		})
	}
}

func manifestValue(m Manifest, name string) (string, bool) {
	for _, attr := range m {
		if attr.Name == name {
			return attr.Value, true
		}
	}
	return "", false
}

func TestBuildSchemaManifest(t *testing.T) {
	descriptor := &common.AppDescriptor{
		Kind:       "Application",
		Name:       "com.example.myapp",
		Title:      common.LocalizedText{Text: "My App"},
		VendorName: "Enonic AS",
		VendorUrl:  "https://enonic.com",
		Url:        "https://example.com",
	}
	manifest, err := buildSchemaManifest(descriptor)
	if err != nil {
		t.Fatal(err)
	}

	expected := map[string]string{
		"Manifest-Version":    "1.0",
		"Bundle-SymbolicName": "com.example.myapp",
		"Bundle-Name":         "My App",
		"X-Bundle-Type":       "application",
		"X-System-Version":    "[8.1,9)",
		"X-Vendor-Name":       "Enonic AS",
		"X-Vendor-Url":        "https://enonic.com",
		"X-Application-Url":   "https://example.com",
	}
	for name, want := range expected {
		if got, ok := manifestValue(manifest, name); !ok || got != want {
			t.Errorf("expected %s: %q, got %q (present: %v)", name, want, got, ok)
		}
	}
	if _, ok := manifestValue(manifest, "Bundle-Version"); ok {
		t.Error("Bundle-Version must not be written")
	}
	if manifest[0].Name != "Manifest-Version" {
		t.Errorf("Manifest-Version must be the first attribute, got %s", manifest[0].Name)
	}
}

func TestBuildSchemaManifest_OmitsEmptyOptionalHeaders(t *testing.T) {
	manifest, err := buildSchemaManifest(&common.AppDescriptor{Kind: "Application", Name: "com.example.myapp"})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Bundle-Name", "X-Vendor-Name", "X-Vendor-Url", "X-Application-Url"} {
		if _, ok := manifestValue(manifest, name); ok {
			t.Errorf("%s must be omitted when empty", name)
		}
	}
	if _, err := buildSchemaManifest(nil); err == nil {
		t.Error("expected an error for a nil descriptor")
	}
}

func TestSchemaJarPath(t *testing.T) {
	got := schemaJarPath("prj", "com.example.myapp")
	want := filepath.Join("prj", "build", "libs", "myapp.jar")
	if got != want {
		t.Errorf("expected %s, got %s", want, got)
	}
}

func TestSetAppDescriptorName(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			"replace existing name",
			"kind: \"Application\"\nname: \"com.example.old\"\ntitle: \"T\"\n",
			"kind: \"Application\"\nname: \"com.example.new\"\ntitle: \"T\"\n",
		},
		{
			"replace unquoted name with comment",
			"kind: \"Application\"\nname:   com.example.old   # app name\n",
			"kind: \"Application\"\nname: \"com.example.new\"\n",
		},
		{
			"insert after kind",
			"# descriptor\nkind: \"Application\"\ntitle: \"T\"\n",
			"# descriptor\nkind: \"Application\"\nname: \"com.example.new\"\ntitle: \"T\"\n",
		},
		{
			"insert at top without kind",
			"title: \"T\"\n",
			"name: \"com.example.new\"\ntitle: \"T\"\n",
		},
		{
			"insert after document start without kind",
			"---\ntitle: \"T\"\n",
			"---\nname: \"com.example.new\"\ntitle: \"T\"\n",
		},
		{
			"keeps CRLF",
			"kind: \"Application\"\r\ntitle: \"T\"\r\n",
			"kind: \"Application\"\r\nname: \"com.example.new\"\r\ntitle: \"T\"\r\n",
		},
		{
			"indented name is not top level",
			"kind: \"Application\"\nconfig:\n  name: \"x\"\n",
			"kind: \"Application\"\nname: \"com.example.new\"\nconfig:\n  name: \"x\"\n",
		},
		{
			"adds trailing newline",
			"kind: \"Application\"",
			"kind: \"Application\"\nname: \"com.example.new\"\n",
		},
		{
			"keeps BOM",
			"\xEF\xBB\xBFkind: \"Application\"\n",
			"\xEF\xBB\xBFkind: \"Application\"\nname: \"com.example.new\"\n",
		},
		{
			"empty file",
			"",
			"name: \"com.example.new\"\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "enonic.yaml")
			if err := os.WriteFile(file, []byte(tc.in), 0644); err != nil {
				t.Fatal(err)
			}
			if err := setAppDescriptorName(file, "com.example.new"); err != nil {
				t.Fatalf("setAppDescriptorName failed: %v", err)
			}
			got, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Errorf("expected %q, got %q", tc.want, string(got))
			}
		})
	}

	if err := setAppDescriptorName("", "com.example.new"); err == nil {
		t.Error("expected an error for a missing descriptor file")
	}
}
