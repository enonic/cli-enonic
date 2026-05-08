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
			actual, err := expandToAbsoluteURL(tt.repo, true)
			if err != nil {
				t.Errorf("expandToAbsoluteURL(%q, true) returned error: %v", tt.repo, err)
				return
			}
			if actual != tt.expected {
				t.Errorf("expandToAbsoluteURL(%q, true) = %q, want %q", tt.repo, actual, tt.expected)
			}
		})
	}
}

func TestExpandToAbsoluteURLWithoutGuessingShortUrls(t *testing.T) {
	_, err := expandToAbsoluteURL("mycompany/myrepo", false)
	if err == nil {
		t.Errorf("expandToAbsoluteURL(%q, false) expected error, got nil", "mycompany/myrepo")
	}
}

func TestExpandToAbsoluteURLInvalidRepo(t *testing.T) {
	repo := "too/many/slashes/here"
	_, err := expandToAbsoluteURL(repo, true)
	if err == nil {
		t.Errorf("expandToAbsoluteURL(%q, true) expected error, got nil", repo)
	}
}
