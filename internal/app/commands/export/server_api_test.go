package export

import "testing"

func TestCreateNewRequest_ServerApi(t *testing.T) {
	isolateEnonicHome(t)
	req := createNewRequest(newCtx(t, "", nil), true)

	if req.Method != "POST" || req.URL.Path != "/server:export/" {
		t.Fatalf("got %s %s", req.Method, req.URL.Path)
	}
	assertKeys(t, decodeJSONBody(t, req.Body), []string{"exportName", "sourceRepoPath"})
}

// the export is named by the path: an exportName in the body would be rejected as an unknown property
func TestCreateLoadRequest_ServerApi(t *testing.T) {
	isolateEnonicHome(t)
	req := createLoadRequest(newCtx(t, "", map[string]bool{"skip-permissions": true}), true)

	if req.Method != "POST" || req.URL.Path != "/server:export/myExport/load" {
		t.Fatalf("got %s %s", req.Method, req.URL.Path)
	}
	params := decodeJSONBody(t, req.Body)
	assertKeys(t, params, []string{"targetRepoPath", "importWithIds", "importWithPermissions"})
	if params["importWithIds"] != true || params["importWithPermissions"] != false {
		t.Errorf("unexpected payload %v", params)
	}
}
