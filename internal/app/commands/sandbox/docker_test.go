package sandbox

import (
	"fmt"
	"testing"
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
