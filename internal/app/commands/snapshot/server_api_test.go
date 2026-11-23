package snapshot

import (
	"flag"
	"testing"

	"github.com/urfave/cli"
)

func TestCreateNewRequest_ServerApi(t *testing.T) {
	isolateEnonicHome(t)
	req := createNewRequest(newSnapCtx("", "com.enonic.cms.default"), true)

	if req.Method != "POST" || req.URL.Path != "/server:snapshot/" {
		t.Fatalf("got %s %s", req.Method, req.URL.Path)
	}
	if params := decodeJSONBody(t, req.Body); params["repositoryId"] != "com.enonic.cms.default" {
		t.Errorf("unexpected payload %v", params)
	}
}

func TestCreateRestoreRequest_ServerApi(t *testing.T) {
	isolateEnonicHome(t)
	byName := createRestoreRequest(newRestoreCtx("", "my-snap", "", false, true), true)
	if byName.URL.Path != "/server:snapshot/my-snap/restore" {
		t.Errorf("got %s", byName.URL.Path)
	}
	if params := decodeJSONBody(t, byName.Body); params["force"] != true {
		t.Errorf("unexpected payload %v", params)
	}

	latest := createRestoreRequest(newRestoreCtx("", "", "", true, false), true)
	if latest.URL.Path != "/server:snapshot/restore" {
		t.Errorf("got %s", latest.URL.Path)
	}
	if params := decodeJSONBody(t, latest.Body); params["latest"] != true {
		t.Errorf("unexpected payload %v", params)
	}
}

func TestCreateDeleteRequest_ServerApi(t *testing.T) {
	isolateEnonicHome(t)
	c := cli.NewContext(nil, flag.NewFlagSet("test", 0), nil)

	byName := createDeleteRequest(c, "my-snap", "", true)
	if byName.Method != "DELETE" || byName.URL.Path != "/server:snapshot/my-snap" || byName.URL.RawQuery != "" {
		t.Errorf("got %s %s?%s", byName.Method, byName.URL.Path, byName.URL.RawQuery)
	}

	before := createDeleteRequest(c, "", "1 Nov 26", true)
	if before.Method != "DELETE" || before.URL.Path != "/server:snapshot/" || before.URL.Query().Get("before") != "2026-11-01T00:00:00Z" {
		t.Errorf("got %s %s?%s", before.Method, before.URL.Path, before.URL.RawQuery)
	}
}
