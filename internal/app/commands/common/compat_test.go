package common

import (
	"flag"
	"testing"

	"github.com/urfave/cli"
)

func newCompatCtx(value string, set bool) *cli.Context {
	fs := flag.NewFlagSet("test", 0)
	fs.String("compat", "", "")
	if set {
		fs.Set("compat", value)
	}
	return cli.NewContext(nil, fs, nil)
}

func TestValidateCompatFlag(t *testing.T) {
	cases := []struct {
		name      string
		value     string
		set       bool
		wantError bool
	}{
		{"nil context", "", false, false},
		{"flag unset", "", false, false},
		{"empty value", "", true, false},
		{"single digit", "7", true, false},
		{"multi digit", "12", true, false},
		{"X.Y", "7.16", true, false},
		{"X.Y multi digit", "10.42", true, false},
		{"X.Y.Z rejected", "7.16.3", true, true},
		{"leading dot rejected", ".7", true, true},
		{"trailing dot rejected", "7.", true, true},
		{"letters rejected", "v7", true, true},
		{"suffix letters rejected", "7x", true, true},
		{"negative rejected", "-7", true, true},
		{"whitespace rejected", " 7", true, true},
		{"non-numeric rejected", "latest", true, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var c *cli.Context
			if tc.name != "nil context" {
				c = newCompatCtx(tc.value, tc.set)
			}
			err := ValidateCompatFlag(c)
			if tc.wantError && err == nil {
				t.Errorf("expected error for value %q, got nil", tc.value)
			}
			if !tc.wantError && err != nil {
				t.Errorf("expected no error for value %q, got %v", tc.value, err)
			}
		})
	}
}

func TestIsCompatMode(t *testing.T) {
	cases := []struct {
		name  string
		value string
		set   bool
		want  bool
	}{
		{"nil context", "", false, false},
		{"flag unset", "", false, false},
		{"empty value", "", true, false},
		{"value 7", "7", true, true},
		{"value 7.15", "7.15", true, true},
		{"value 7.x", "7.x", true, true},
		{"value 8", "8", true, false},
		{"value 8.0", "8.0", true, false},
		{"value latest", "latest", true, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var c *cli.Context
			if tc.name != "nil context" {
				c = newCompatCtx(tc.value, tc.set)
			}
			if got := IsCompatMode(c); got != tc.want {
				t.Errorf("IsCompatMode(%q, set=%v) = %v, want %v", tc.value, tc.set, got, tc.want)
			}
		})
	}
}
