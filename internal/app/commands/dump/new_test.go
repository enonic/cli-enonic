package dump

import (
	"flag"
	"testing"

	"github.com/urfave/cli"
)

func newCreateCtx(compat string, archive, skipVersions bool, maxAge, maxVersions string) *cli.Context {
	fs := flag.NewFlagSet("test", 0)
	fs.String("compat", "", "")
	fs.Bool("archive", false, "")
	fs.Bool("skip-versions", false, "")
	fs.String("max-version-age", "", "")
	fs.String("max-versions", "", "")
	if compat != "" {
		fs.Set("compat", compat)
	}
	if archive {
		fs.Set("archive", "true")
	}
	if skipVersions {
		fs.Set("skip-versions", "true")
	}
	if maxAge != "" {
		fs.Set("max-version-age", maxAge)
	}
	if maxVersions != "" {
		fs.Set("max-versions", maxVersions)
	}
	return cli.NewContext(nil, fs, nil)
}

func TestBuildNewParams_Defaults(t *testing.T) {
	c := newCreateCtx("", false, false, "", "")
	params := buildNewParams(c, "mydump")

	if params["name"] != "mydump" {
		t.Errorf("expected name=mydump, got %v", params["name"])
	}
	if params["includeVersions"] != true {
		t.Errorf("expected includeVersions=true by default, got %v", params["includeVersions"])
	}
	if _, ok := params["archive"]; ok {
		t.Errorf("expected no archive param in XP8 mode")
	}
	if _, ok := params["maxAge"]; ok {
		t.Errorf("expected no maxAge param")
	}
	if _, ok := params["maxVersions"]; ok {
		t.Errorf("expected no maxVersions param")
	}
}

func TestBuildNewParams_XP8IgnoresArchive(t *testing.T) {
	// XP8 dump create endpoint does not accept the archive param;
	// even when --archive is set, it must not be sent unless compat mode is enabled.
	c := newCreateCtx("", true, false, "", "")
	params := buildNewParams(c, "mydump")

	if _, ok := params["archive"]; ok {
		t.Errorf("expected archive param to be suppressed without --compat, got %v", params["archive"])
	}
}

func TestBuildNewParams_CompatWithArchive(t *testing.T) {
	c := newCreateCtx("7", true, false, "", "")
	params := buildNewParams(c, "mydump")

	if params["archive"] != true {
		t.Errorf("expected archive=true in compat mode, got %v", params["archive"])
	}
}

func TestBuildNewParams_CompatWithoutArchive(t *testing.T) {
	c := newCreateCtx("7", false, false, "", "")
	params := buildNewParams(c, "mydump")

	if _, ok := params["archive"]; ok {
		t.Errorf("expected no archive param when --archive not set, even in compat mode")
	}
}

func TestBuildNewParams_SkipVersions(t *testing.T) {
	c := newCreateCtx("", false, true, "", "")
	params := buildNewParams(c, "mydump")

	if params["includeVersions"] != false {
		t.Errorf("expected includeVersions=false when --skip-versions is set, got %v", params["includeVersions"])
	}
}

func TestBuildNewParams_MaxAgeAndVersions(t *testing.T) {
	c := newCreateCtx("", false, false, "30", "5")
	params := buildNewParams(c, "mydump")

	if params["maxAge"] != "30" {
		t.Errorf("expected maxAge=30, got %v", params["maxAge"])
	}
	if params["maxVersions"] != "5" {
		t.Errorf("expected maxVersions=5, got %v", params["maxVersions"])
	}
}

func TestBuildNewParams_CompatPrefixVariants(t *testing.T) {
	c := newCreateCtx("7.15.3", true, false, "", "")
	params := buildNewParams(c, "mydump")

	if params["archive"] != true {
		t.Errorf("expected compat mode for value '7.15.3', archive param missing")
	}
}
