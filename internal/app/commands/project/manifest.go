package project

import (
	"cli-enonic/internal/app/commands/common"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"
)

const MANIFEST_PATH = "META-INF/MANIFEST.MF"

// JAR specification: no line may be longer than 72 bytes (not characters), continuation lines start with a space.
const manifestMaxLineBytes = 72

var manifestAttrNameRegex = regexp.MustCompile(`^[A-Za-z0-9_-]{1,70}$`)

var manifestValueSanitizer = strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ")

type ManifestAttr struct {
	Name  string
	Value string
}

type Manifest []ManifestAttr

func (m Manifest) Write(w io.Writer) error {
	for _, attr := range m {
		if err := writeManifestAttr(w, attr.Name, attr.Value); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, "\r\n")
	return err
}

func writeManifestAttr(w io.Writer, name, value string) error {
	if !manifestAttrNameRegex.MatchString(name) {
		return fmt.Errorf("invalid manifest attribute name '%s'", name)
	}
	value = strings.TrimSpace(manifestValueSanitizer.Replace(value))

	var buf strings.Builder
	lineBytes := 0
	for _, r := range name + ": " + value {
		runeBytes := utf8.RuneLen(r)
		if lineBytes+runeBytes > manifestMaxLineBytes {
			buf.WriteString("\r\n ")
			lineBytes = 1
		}
		buf.WriteRune(r)
		lineBytes += runeBytes
	}
	buf.WriteString("\r\n")

	_, err := io.WriteString(w, buf.String())
	return err
}

// buildStaticManifest creates the OSGi manifest for a Static application, mirroring the headers
// written by the com.enonic.xp.app gradle plugin.
func buildStaticManifest(appName string, descriptor *common.AppDescriptor) (Manifest, error) {
	systemVersion, err := common.SystemVersionRange(common.STATIC_APP_XP_VERSION)
	if err != nil {
		return nil, err
	}

	// Bundle-Version is omitted: Static applications have no version, and both OSGi and XP
	// default to 0.0.0 when the header is absent
	manifest := Manifest{
		{"Manifest-Version", "1.0"},
		{"Bundle-ManifestVersion", "2"},
		{"Bundle-SymbolicName", appName},
	}
	if descriptor != nil && descriptor.Title.Text != "" {
		manifest = append(manifest, ManifestAttr{"Bundle-Name", descriptor.Title.Text})
	}
	manifest = append(manifest,
		ManifestAttr{"X-Bundle-Type", "application"},
		ManifestAttr{"X-System-Version", systemVersion},
	)
	if descriptor != nil {
		optional := []ManifestAttr{
			{"X-Vendor-Name", descriptor.VendorName},
			{"X-Vendor-Url", descriptor.VendorUrl},
			{"X-Application-Url", descriptor.Url},
		}
		for _, attr := range optional {
			if attr.Value != "" {
				manifest = append(manifest, attr)
			}
		}
	}
	manifest = append(manifest, ManifestAttr{"Created-By", "Enonic CLI"})

	return manifest, nil
}
