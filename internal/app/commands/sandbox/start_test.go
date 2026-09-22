package sandbox

import (
	"reflect"
	"testing"
)

// TestRequiredPorts verifies the start-up preflight covers every host port the
// sandbox will bind. A Docker sandbox publishes the JDWP port on the host when
// debugging, so without it `docker run` fails *after* the preflight passed — and
// in detached mode the sandbox is recorded as running regardless (#703).
func TestRequiredPorts(t *testing.T) {
	tests := []struct {
		name     string
		httpPort uint16
		isDocker bool
		debug    bool
		want     []uint16
	}{
		{"distro", 8080, false, false, []uint16{8080, 4848, 2609}},
		{"distro with debug", 8080, false, true, []uint16{8080, 4848, 2609}},
		{"docker", 8080, true, false, []uint16{8080, 4848, 2609}},
		{"docker with debug", 8080, true, true, []uint16{8080, 4848, 2609, 5005}},
		{"docker with debug custom http port", 9090, true, true, []uint16{9090, 4848, 2609, 5005}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := requiredPorts(tt.httpPort, tt.isDocker, tt.debug)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("requiredPorts(%d, %v, %v) = %v, want %v", tt.httpPort, tt.isDocker, tt.debug, got, tt.want)
			}
		})
	}
}
