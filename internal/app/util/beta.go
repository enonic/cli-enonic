package util

import (
	"os"
	"path/filepath"
)

const BETA_ENV_VAR = "ENONIC_CLI_BETA"
const BETA_MESSAGE = "This is a beta feature. Set ENONIC_CLI_BETA=true to use it."
const SNAP_ENV_VAR = "SNAP_USER_COMMON"

// IsBeta reports whether beta features are enabled with ENONIC_CLI_BETA=true. The variable is read on
// every call
func IsBeta() bool {
	return os.Getenv(BETA_ENV_VAR) == "true"
}

// GetInEnonicDir is a path in the directory where the CLI keeps its data, the snap's common directory
// when running as a snap
func GetInEnonicDir(children ...string) string {
	var joinArgs []string
	if GetCurrentOs() == "linux" {
		if snapCommon, snapExists := os.LookupEnv(SNAP_ENV_VAR); snapExists {
			joinArgs = []string{snapCommon, "dot-enonic"}
		}
	}
	if joinArgs == nil {
		joinArgs = []string{GetEnonicHome()}
	}
	return filepath.Join(append(joinArgs, children...)...)
}
