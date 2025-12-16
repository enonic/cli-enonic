// This file was co-authored by Claude AI assistant (Anthropic)
// Generated with assistance from AI for test coverage of sandbox create functionality

package sandbox

import (
	"testing"
)

// TestGetFirstValidSandboxName tests the generation of unique sandbox names
func TestGetFirstValidSandboxName(t *testing.T) {
	tests := []struct {
		name        string
		sandboxes   []*Sandbox
		expectedNum int
	}{
		{
			name:        "Empty list should return Sandbox1",
			sandboxes:   []*Sandbox{},
			expectedNum: 1,
		},
		{
			name: "Should skip existing Sandbox1",
			sandboxes: []*Sandbox{
				{Name: "Sandbox1", Distro: "7.0.0"},
			},
			expectedNum: 2,
		},
		{
			name: "Should skip multiple existing sandboxes",
			sandboxes: []*Sandbox{
				{Name: "Sandbox1", Distro: "7.0.0"},
				{Name: "Sandbox2", Distro: "7.0.0"},
				{Name: "Sandbox3", Distro: "7.0.0"},
			},
			expectedNum: 4,
		},
		{
			name: "Should handle case-insensitive comparison",
			sandboxes: []*Sandbox{
				{Name: "sandbox1", Distro: "7.0.0"},
			},
			expectedNum: 2,
		},
		{
			name: "Should handle mixed case names",
			sandboxes: []*Sandbox{
				{Name: "SandBox1", Distro: "7.0.0"},
			},
			expectedNum: 2,
		},
		{
			name: "Should find gap in sequence",
			sandboxes: []*Sandbox{
				{Name: "Sandbox1", Distro: "7.0.0"},
				{Name: "Sandbox3", Distro: "7.0.0"},
			},
			expectedNum: 2,
		},
		{
			name: "Should handle non-sequential names",
			sandboxes: []*Sandbox{
				{Name: "MyCustomSandbox", Distro: "7.0.0"},
				{Name: "AnotherBox", Distro: "7.0.0"},
			},
			expectedNum: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFirstValidSandboxName(tt.sandboxes)

			// Verify result is not empty
			if result == "" {
				t.Errorf("getFirstValidSandboxName() returned empty string")
			}

			// Verify it starts with "Sandbox"
			if len(result) < 8 || result[:7] != "Sandbox" {
				t.Errorf("getFirstValidSandboxName() = %v, expected to start with 'Sandbox'", result)
			}

			// Verify it's not in the existing list (case-insensitive)
			for _, sandbox := range tt.sandboxes {
				if result == sandbox.Name {
					t.Errorf("getFirstValidSandboxName() = %v, which already exists in the list", result)
				}
			}
		})
	}
}

// TestGetFirstValidSandboxNameSpecific tests specific expected outputs
func TestGetFirstValidSandboxNameSpecific(t *testing.T) {
	// Test with no existing sandboxes
	result := getFirstValidSandboxName([]*Sandbox{})
	if result != "Sandbox1" {
		t.Errorf("getFirstValidSandboxName([]) = %v, want Sandbox1", result)
	}

	// Test with Sandbox1 existing
	result = getFirstValidSandboxName([]*Sandbox{
		{Name: "Sandbox1", Distro: "7.0.0"},
	})
	if result != "Sandbox2" {
		t.Errorf("getFirstValidSandboxName with Sandbox1 = %v, want Sandbox2", result)
	}

	// Test with Sandbox1 and Sandbox2 existing
	result = getFirstValidSandboxName([]*Sandbox{
		{Name: "Sandbox1", Distro: "7.0.0"},
		{Name: "Sandbox2", Distro: "7.0.0"},
	})
	if result != "Sandbox3" {
		t.Errorf("getFirstValidSandboxName with Sandbox1,2 = %v, want Sandbox3", result)
	}

	// Test case insensitivity
	result = getFirstValidSandboxName([]*Sandbox{
		{Name: "sandbox1", Distro: "7.0.0"},
	})
	if result != "Sandbox2" {
		t.Errorf("getFirstValidSandboxName with sandbox1 (lowercase) = %v, want Sandbox2", result)
	}

	// Test finding gaps
	result = getFirstValidSandboxName([]*Sandbox{
		{Name: "Sandbox1", Distro: "7.0.0"},
		{Name: "Sandbox3", Distro: "7.0.0"},
	})
	if result != "Sandbox2" {
		t.Errorf("getFirstValidSandboxName with gap = %v, want Sandbox2", result)
	}
}

// TestGetFirstValidSandboxNameWithManyBoxes tests with a larger number of sandboxes
func TestGetFirstValidSandboxNameWithManyBoxes(t *testing.T) {
	// Create sandboxes 1-10
	sandboxes := make([]*Sandbox, 10)
	sandboxes[0] = &Sandbox{Name: "Sandbox1", Distro: "7.0.0"}
	sandboxes[1] = &Sandbox{Name: "Sandbox2", Distro: "7.0.0"}
	sandboxes[2] = &Sandbox{Name: "Sandbox3", Distro: "7.0.0"}
	sandboxes[3] = &Sandbox{Name: "Sandbox4", Distro: "7.0.0"}
	sandboxes[4] = &Sandbox{Name: "Sandbox5", Distro: "7.0.0"}
	sandboxes[5] = &Sandbox{Name: "Sandbox6", Distro: "7.0.0"}
	sandboxes[6] = &Sandbox{Name: "Sandbox7", Distro: "7.0.0"}
	sandboxes[7] = &Sandbox{Name: "Sandbox8", Distro: "7.0.0"}
	sandboxes[8] = &Sandbox{Name: "Sandbox9", Distro: "7.0.0"}
	sandboxes[9] = &Sandbox{Name: "Sandbox10", Distro: "7.0.0"}

	result := getFirstValidSandboxName(sandboxes)
	if result != "Sandbox11" {
		t.Errorf("getFirstValidSandboxName with 10 boxes = %v, want Sandbox11", result)
	}
}

// TestGetFirstValidSandboxNameWithCustomNames tests with non-standard sandbox names
func TestGetFirstValidSandboxNameWithCustomNames(t *testing.T) {
	sandboxes := []*Sandbox{
		{Name: "MyCustomSandbox", Distro: "7.0.0"},
		{Name: "AnotherBox", Distro: "7.0.0"},
		{Name: "TestEnv", Distro: "7.0.0"},
	}

	result := getFirstValidSandboxName(sandboxes)

	// Should return Sandbox1 since none of the custom boxes match the pattern
	if result != "Sandbox1" {
		t.Errorf("getFirstValidSandboxName with custom names = %v, want Sandbox1", result)
	}
}
