package dump

import (
	"flag"
	"io"
	"net/http"
	"testing"

	"github.com/urfave/cli"
)

func dumpCtx() *cli.Context {
	fs := flag.NewFlagSet("test", 0)
	fs.String("compat", "", "")
	fs.Bool("upgrade", false, "")
	fs.Bool("archive", false, "")
	fs.Bool("skip-versions", false, "")
	fs.String("max-version-age", "", "")
	fs.String("max-versions", "", "")
	fs.Set("upgrade", "true")
	return cli.NewContext(nil, fs, nil)
}

func TestDumpRequests_ServerApi(t *testing.T) {
	isolateEnonicHome(t)
	c := dumpCtx()

	for want, req := range map[string]*http.Request{
		"/server:dump/":               createNewRequest(c, "mydump", true),
		"/server:dump/mydump/load":    createLoadRequest(c, "mydump.zip", true),
		"/server:dump/mydump/upgrade": createUpgradeRequest(c, "mydump.zip", true),
	} {
		if req.Method != "POST" || req.URL.Path != want {
			t.Errorf("got %s %s, want POST %s", req.Method, req.URL.Path, want)
		}
	}

	if body, _ := io.ReadAll(createUpgradeRequest(c, "mydump", true).Body); len(body) != 0 {
		t.Errorf("upgrade names the dump by the path only, got %s", body)
	}
}
