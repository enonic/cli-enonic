package dump

import (
	"flag"
	"testing"

	"github.com/urfave/cli"
)

func newLoadCtx(compat string, archive, upgrade bool) *cli.Context {
	fs := flag.NewFlagSet("test", 0)
	fs.String("compat", "", "")
	fs.Bool("archive", false, "")
	fs.Bool("upgrade", false, "")
	if compat != "" {
		fs.Set("compat", compat)
	}
	if archive {
		fs.Set("archive", "true")
	}
	if upgrade {
		fs.Set("upgrade", "true")
	}
	return cli.NewContext(nil, fs, nil)
}

func TestNormalizeName(t *testing.T) {
	cases := []struct {
		input    string
		wantName string
		wantZip  bool
	}{
		{"mydump", "mydump", false},
		{"mydump.zip", "mydump", true},
		{"my.dump", "my.dump", false},
		{"my.dump.zip", "my.dump", true},
		{"", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			gotName, gotZip := normalizeName(tc.input)
			if gotName != tc.wantName || gotZip != tc.wantZip {
				t.Errorf("normalizeName(%q) = (%q, %v), want (%q, %v)",
					tc.input, gotName, gotZip, tc.wantName, tc.wantZip)
			}
		})
	}
}

func TestBuildLoadParams_XP8Default(t *testing.T) {
	// Without compat mode, archive flag and .zip suffix detection must be ignored.
	c := newLoadCtx("", true, false)
	params := buildLoadParams(c, "mydump.zip")

	if params["name"] != "mydump" {
		t.Errorf("expected name=mydump, got %v", params["name"])
	}
	if _, ok := params["archive"]; ok {
		t.Errorf("expected no archive param in XP8 mode, got %v", params["archive"])
	}
	if _, ok := params["upgrade"]; ok {
		t.Errorf("expected no upgrade param when flag is false")
	}
}

func TestBuildLoadParams_CompatExplicitArchive(t *testing.T) {
	c := newLoadCtx("7", true, false)
	params := buildLoadParams(c, "mydump")

	if params["name"] != "mydump" {
		t.Errorf("expected name=mydump, got %v", params["name"])
	}
	if params["archive"] != true {
		t.Errorf("expected archive=true in compat mode with --archive, got %v", params["archive"])
	}
}

func TestBuildLoadParams_CompatZipAutoDetect(t *testing.T) {
	// In compat mode, a .zip suffix should auto-set archive=true.
	c := newLoadCtx("7", false, false)
	params := buildLoadParams(c, "mydump.zip")

	if params["name"] != "mydump" {
		t.Errorf("expected name=mydump (stripped), got %v", params["name"])
	}
	if params["archive"] != true {
		t.Errorf("expected archive=true from .zip auto-detect in compat mode, got %v", params["archive"])
	}
}

func TestBuildLoadParams_CompatNoArchive(t *testing.T) {
	c := newLoadCtx("7", false, false)
	params := buildLoadParams(c, "mydump")

	if _, ok := params["archive"]; ok {
		t.Errorf("expected no archive param when neither --archive nor .zip set")
	}
}

func TestBuildLoadParams_Upgrade(t *testing.T) {
	c := newLoadCtx("", false, true)
	params := buildLoadParams(c, "mydump")

	if params["upgrade"] != true {
		t.Errorf("expected upgrade=true, got %v", params["upgrade"])
	}
}

func TestBuildLoadParams_CompatPrefixVariants(t *testing.T) {
	// Any compat value starting with "7" should enable compat behavior.
	c := newLoadCtx("7.15", true, false)
	params := buildLoadParams(c, "mydump")

	if params["archive"] != true {
		t.Errorf("expected compat mode for value '7.15', archive param missing")
	}
}
