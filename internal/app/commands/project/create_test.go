package project

import "testing"

func TestExpandToAbsoluteURLWithShortGitHubRepo(t *testing.T) {
	tests := []struct {
		name     string
		repo     string
		expected string
	}{
		{
			name:     "organization and repo",
			repo:     "mycompany/myrepo",
			expected: "https://github.com/mycompany/myrepo.git",
		},
		{
			name:     "organization and repo with git suffix",
			repo:     "mycompany/myrepo.git",
			expected: "https://github.com/mycompany/myrepo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := expandToAbsoluteURl(tt.repo, true)
			if err != nil {
				t.Fatalf("expandToAbsoluteURl(%q, true) returned error: %v", tt.repo, err)
			}
			if actual != tt.expected {
				t.Fatalf("expandToAbsoluteURl(%q, true) = %q, want %q", tt.repo, actual, tt.expected)
			}
		})
	}
}
