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
