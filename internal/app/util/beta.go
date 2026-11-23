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

// WriteFileAtomic replaces the file with a temporary file written next to it, so a concurrent reader never
// sees partial content. The directory is created with mode 0700 if missing
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
