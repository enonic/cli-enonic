package util

import "testing"

func TestIsBeta(t *testing.T) {
	tests := map[string]bool{"true": true, "": false, "1": false, "TRUE": false, "yes": false}
	for value, want := range tests {
		t.Setenv(BETA_ENV_VAR, value)
		if got := IsBeta(); got != want {
			t.Errorf("IsBeta() with %q = %v, want %v", value, got, want)
		}
	}
}
