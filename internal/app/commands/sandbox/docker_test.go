package sandbox

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestIsDockerDistro verifies docker distro detection
func TestIsDockerDistro(t *testing.T) {
	tests := []struct {
		distro   string
		expected bool
	}{
		{"docker:enonic/xp:7.13.4-sdk", true},
		{"docker:enonic/xp:latest", true},
		{"docker:my-custom/image:1.0", true},
		{"enonic-xp-linux-sdk-7.13.4", false},
		{"enonic-xp-linux-arm64-sdk-7.13.4", false},
		{"", false},
		{"docker", false},
		{"docker:", true},
	}

	for _, tt := range tests {
		t.Run(tt.distro, func(t *testing.T) {
			result := IsDockerDistro(tt.distro)
			if result != tt.expected {
				t.Errorf("IsDockerDistro(%q) = %v, want %v", tt.distro, result, tt.expected)
			}
		})
	}
}

// TestGetDockerImageName verifies extraction of docker image name from distro
func TestGetDockerImageName(t *testing.T) {
	tests := []struct {
		distro   string
		expected string
	}{
		{"docker:enonic/xp:7.13.4-sdk", "enonic/xp:7.13.4-sdk"},
		{"docker:enonic/xp:latest", "enonic/xp:latest"},
		{"docker:my-registry.com/image:1.0", "my-registry.com/image:1.0"},
	}

	for _, tt := range tests {
		t.Run(tt.distro, func(t *testing.T) {
			result := GetDockerImageName(tt.distro)
			if result != tt.expected {
				t.Errorf("GetDockerImageName(%q) = %q, want %q", tt.distro, result, tt.expected)
			}
		})
	}
}

// TestFormatDockerDistro verifies formatting a docker image as a distro name
func TestFormatDockerDistro(t *testing.T) {
	tests := []struct {
		imageName string
		expected  string
	}{
		{"enonic/xp:7.13.4-sdk", "docker:enonic/xp:7.13.4-sdk"},
		{"enonic/xp:latest", "docker:enonic/xp:latest"},
	}

	for _, tt := range tests {
		t.Run(tt.imageName, func(t *testing.T) {
			result := FormatDockerDistro(tt.imageName)
			if result != tt.expected {
				t.Errorf("FormatDockerDistro(%q) = %q, want %q", tt.imageName, result, tt.expected)
			}
		})
	}
}

// TestGetDockerContainerName verifies container name generation from sandbox name
func TestGetDockerContainerName(t *testing.T) {
	tests := []struct {
		sandboxName string
		expected    string
	}{
		{"MySandbox", "enonic-sandbox-mysandbox"},
		{"Sandbox1", "enonic-sandbox-sandbox1"},
		{"my_sandbox", "enonic-sandbox-my_sandbox"},
		// Defensive sanitization: characters docker disallows in container names
		// must be replaced with `_` so the produced name always matches
		// `[a-zA-Z0-9][a-zA-Z0-9_.-]*`.
		{"My Sandbox", "enonic-sandbox-my_sandbox"},
		{"my/sandbox", "enonic-sandbox-my_sandbox"},
		{"my:sandbox", "enonic-sandbox-my_sandbox"},
		{"my+sandbox!", "enonic-sandbox-my_sandbox_"},
		{"box.1-test", "enonic-sandbox-box.1-test"},
	}

	for _, tt := range tests {
		t.Run(tt.sandboxName, func(t *testing.T) {
			result := GetDockerContainerName(tt.sandboxName)
			if result != tt.expected {
				t.Errorf("GetDockerContainerName(%q) = %q, want %q", tt.sandboxName, result, tt.expected)
			}
		})
	}
}

// TestFormatSandboxDisplay verifies sandbox display formatting for both docker and distro sandboxes
func TestFormatSandboxDisplay(t *testing.T) {
	osWithArch := "linux"

	tests := []struct {
		box      *Sandbox
		expected string
	}{
		{
			box:      &Sandbox{Name: "MyBox", Distro: "docker:enonic/xp:7.13.4-sdk"},
			expected: "MyBox (docker:enonic/xp:7.13.4-sdk)",
		},
		{
			box:      &Sandbox{Name: "MyBox", Distro: "docker:enonic/xp:latest"},
			expected: "MyBox (docker:enonic/xp:latest)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.box.Distro, func(t *testing.T) {
			result := formatSandboxDisplay(tt.box, osWithArch)
			if result != tt.expected {
				t.Errorf("formatSandboxDisplay(%v, %q) = %q, want %q", tt.box, osWithArch, result, tt.expected)
			}
		})
	}
}

// TestDockerDistroRoundTrip verifies that a docker distro name survives a format/parse round trip
func TestDockerDistroRoundTrip(t *testing.T) {
	imageNames := []string{
		"enonic/xp:7.13.4-sdk",
		"enonic/xp:latest",
		"my-registry.com:5000/enonic/xp:7.13.4",
	}

	for _, imageName := range imageNames {
		t.Run(imageName, func(t *testing.T) {
			distro := FormatDockerDistro(imageName)
			if !IsDockerDistro(distro) {
				t.Errorf("FormatDockerDistro(%q) = %q, IsDockerDistro returned false", imageName, distro)
			}
			result := GetDockerImageName(distro)
			if result != imageName {
				t.Errorf("Round trip for %q: got %q", imageName, result)
			}
		})
	}
}

// TestDockerContainerPrefix verifies the container name prefix
func TestDockerContainerPrefix(t *testing.T) {
	name := GetDockerContainerName("Test")
	expected := fmt.Sprintf("%stest", DOCKER_CONTAINER_PREFIX)
	if name != expected {
		t.Errorf("GetDockerContainerName(\"Test\") = %q, want %q", name, expected)
	}
}

// TestFetchDockerTagsParsesAndPrefixes verifies the tag list is parsed from the
// Docker Hub response shape and each entry is prefixed with the image name.
func TestFetchDockerTagsParsesAndPrefixes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"count":3,"results":[
			{"name":"7.13.4-sdk"},
			{"name":""},
			{"name":"7.14.0-sdk"}
		]}`)
	}))
	defer srv.Close()

	tags, err := fetchDockerTagsFromURL(srv.URL, time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"enonic/xp:7.13.4-sdk", "enonic/xp:7.14.0-sdk"}
	if len(tags) != len(want) {
		t.Fatalf("got %d tags %v, want %d %v", len(tags), tags, len(want), want)
	}
	for i, tag := range tags {
		if tag != want[i] {
			t.Errorf("tag[%d] = %q, want %q", i, tag, want[i])
		}
	}
}

// TestFetchDockerTagsTimesOut verifies a stalled server is abandoned within the
// supplied timeout instead of hanging the wizard.
func TestFetchDockerTagsTimesOut(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Hold the response open until the client gives up.
		time.Sleep(2 * time.Second)
	}))
	defer srv.Close()

	start := time.Now()
	_, err := fetchDockerTagsFromURL(srv.URL, 100*time.Millisecond)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if elapsed > time.Second {
		t.Errorf("fetch took %v, expected <1s with a 100ms timeout", elapsed)
	}
}

// TestFetchDockerTagsBadJSON verifies parse failures surface as a clear error.
func TestFetchDockerTagsBadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "not json")
	}))
	defer srv.Close()

	if _, err := fetchDockerTagsFromURL(srv.URL, time.Second); err == nil {
		t.Fatal("expected parse error, got nil")
	}
}
