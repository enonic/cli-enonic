package env

import (
	"cli-enonic/internal/app/commands/remote"
	"testing"
)

func activeEnv(t *testing.T) (string, int) {
	t.Helper()
	data, err := remote.ReadEnvironments()
	if err != nil {
		t.Fatal(err)
	}
	return data.Active, len(data.Environments)
}

func TestActivateAddsCloudInstanceByName(t *testing.T) {
	setup(t)
	t.Setenv(CLOUD_DOMAIN_VAR, "enonic-dev.cloud")

	target, err := Activate("acme-shop-prod")
	if err != nil {
		t.Fatal(err)
	}
	if target.Name != "acme-shop-prod" || target.Url.String() != "https://acme-shop-prod-mgmt.enonic-dev.cloud" {
		t.Fatalf("unexpected target %+v", target)
	}
	if active, count := activeEnv(t); active != "acme-shop-prod" || count != 1 {
		t.Fatalf("active %q of %d", active, count)
	}
}

func TestActivateAddsUrlUnderHostLabel(t *testing.T) {
	setup(t)

	target, err := Activate("https://acme-mgmt.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if target.Name != "acme" || target.Url.String() != "https://acme-mgmt.example.com" {
		t.Fatalf("unexpected target %+v", target)
	}
}

func TestActivateUsesExistingEnvironment(t *testing.T) {
	setup(t)
	if err := run(t, "add", "prod", "https://prod.example/"); err != nil {
		t.Fatal(err)
	}

	for _, arg := range []string{"prod", "https://prod.example", "https://prod.example/"} {
		target, err := Activate(arg)
		if err != nil {
			t.Fatalf("%s: %v", arg, err)
		}
		if target.Name != "prod" {
			t.Fatalf("%s: unexpected target %+v", arg, target)
		}
		if active, count := activeEnv(t); active != "prod" || count != 1 {
			t.Fatalf("%s: active %q of %d", arg, active, count)
		}
	}
}

func TestActivateLocal(t *testing.T) {
	setup(t)
	if _, err := Activate("https://prod.example"); err != nil {
		t.Fatal(err)
	}

	for _, arg := range []string{"local", "http://localhost:4848"} {
		target, err := Activate(arg)
		if err != nil {
			t.Fatalf("%s: %v", arg, err)
		}
		if active, _ := activeEnv(t); target.Name != remote.LOCAL_ENV || active != "" {
			t.Fatalf("%s: target %+v, active %q", arg, target, active)
		}
	}
}

func TestActivateRefusesTakenName(t *testing.T) {
	setup(t)
	if err := run(t, "add", "acme", "https://acme.example.org"); err != nil {
		t.Fatal(err)
	}

	if _, err := Activate("https://acme-mgmt.example.com"); err == nil {
		t.Fatal("expected an error for a name taken by another url")
	}
	if active, count := activeEnv(t); active != "" || count != 1 {
		t.Fatalf("active %q of %d", active, count)
	}
}

func TestActivateRefusesInvalidName(t *testing.T) {
	setup(t)
	for _, arg := range []string{"Acme", "url-acme", "ftp://acme.example"} {
		if _, err := Activate(arg); err == nil {
			t.Fatalf("%s: expected an error", arg)
		}
	}
	if _, count := activeEnv(t); count != 0 {
		t.Fatalf("added %d environments", count)
	}
}
