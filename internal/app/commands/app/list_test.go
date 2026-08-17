package app

import (
	"strings"
	"testing"
)

const listEventStream = `id: 018f-1
event: list
data: {"applications":[{"displayName":"Superhero","key":"com.enonic.app.superhero","local":false,"maxSystemVersion":"8","minSystemVersion":"7.7.0","modifiedTime":"2026-08-17T10:00:00.000Z","state":"started","url":"https://market.enonic.com/","vendorName":"Enonic AS","vendorUrl":"https://enonic.com","version":"2.0.5"},{"key":"com.enonic.app.local","local":true,"modifiedTime":"2026-08-17T11:00:00.000Z","state":"stopped","version":"1.0.0-SNAPSHOT"}]}

`

func TestReadApplicationList(t *testing.T) {
	result, err := readApplicationList(strings.NewReader(listEventStream))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Applications) != 2 {
		t.Fatalf("expected 2 applications, got %d", len(result.Applications))
	}

	superhero := result.Applications[0]
	if superhero.Key != "com.enonic.app.superhero" {
		t.Errorf("unexpected key: %q", superhero.Key)
	}
	if superhero.DisplayName != "Superhero" {
		t.Errorf("unexpected display name: %q", superhero.DisplayName)
	}
	if superhero.Version != "2.0.5" {
		t.Errorf("unexpected version: %q", superhero.Version)
	}
	if superhero.State != "started" {
		t.Errorf("unexpected state: %q", superhero.State)
	}
	if superhero.Local {
		t.Error("expected superhero not to be local")
	}
	if superhero.MinSystemVersion != "7.7.0" || superhero.MaxSystemVersion != "8" {
		t.Errorf("unexpected system versions: %q - %q", superhero.MinSystemVersion, superhero.MaxSystemVersion)
	}
	if superhero.VendorName != "Enonic AS" || superhero.VendorUrl != "https://enonic.com" {
		t.Errorf("unexpected vendor: %q %q", superhero.VendorName, superhero.VendorUrl)
	}
	if superhero.ModifiedTime.IsZero() {
		t.Error("expected modified time to be parsed")
	}

	local := result.Applications[1]
	if !local.Local || local.State != "stopped" || local.DisplayName != "" {
		t.Errorf("unexpected local application: %+v", local)
	}
}

func TestReadApplicationListSkipsOtherEvents(t *testing.T) {
	stream := "event: state\ndata: {\"key\":\"com.enonic.app.superhero\"}\n\n" + listEventStream

	result, err := readApplicationList(strings.NewReader(stream))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Applications) != 2 {
		t.Errorf("expected 2 applications, got %d", len(result.Applications))
	}
}

func TestReadApplicationListEmpty(t *testing.T) {
	result, err := readApplicationList(strings.NewReader("event: list\ndata: {\"applications\":[]}\n\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Applications) != 0 {
		t.Errorf("expected no applications, got %d", len(result.Applications))
	}
}

func TestReadApplicationListWithoutListEvent(t *testing.T) {
	_, err := readApplicationList(strings.NewReader("event: state\ndata: {}\n\n"))
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "list") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestReadApplicationListInvalidJson(t *testing.T) {
	if _, err := readApplicationList(strings.NewReader("event: list\ndata: not json\n\n")); err == nil {
		t.Fatal("expected an error")
	}
}

// XP 7 may send explicit nulls where XP 8 omits the field
func TestReadApplicationListWithNullFields(t *testing.T) {
	stream := `event: list
data: {"applications":[{"key":"com.enonic.app.local","version":"1.0.0","state":"stopped","local":true,"displayName":null,"url":null,"vendorName":null,"vendorUrl":null,"minSystemVersion":null,"maxSystemVersion":null,"modifiedTime":null}]}

`
	result, err := readApplicationList(strings.NewReader(stream))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Applications) != 1 {
		t.Fatalf("expected 1 application, got %d", len(result.Applications))
	}
	app := result.Applications[0]
	if app.Key != "com.enonic.app.local" || app.DisplayName != "" || !app.ModifiedTime.IsZero() {
		t.Errorf("unexpected application: %+v", app)
	}
}

func TestPrintApplications(t *testing.T) {
	result, err := readApplicationList(strings.NewReader(listEventStream))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var out strings.Builder
	printApplications(&out, result.Applications)

	expected := "KEY                        NAME        VERSION          STATE     LOCAL\n" +
		"com.enonic.app.superhero   Superhero   2.0.5            started   \n" +
		"com.enonic.app.local                   1.0.0-SNAPSHOT   stopped   yes\n"

	if out.String() != expected {
		t.Errorf("unexpected table:\n%s\nwant:\n%s", out.String(), expected)
	}
}

func TestPrintApplicationsEmpty(t *testing.T) {
	var out strings.Builder
	printApplications(&out, nil)

	if out.String() != "" {
		t.Errorf("expected nothing on stdout, got %q", out.String())
	}
}
