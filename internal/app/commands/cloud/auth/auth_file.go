package auth

import (
	"cli-enonic/internal/app/commands/cloud/util"
	"encoding/gob"
	"fmt"
	"io"
	"path/filepath"
)

// Logout user
func Logout() error {
	if f, err := authFile(); err != nil {
		return err
	} else {
		return util.RemoveFile(f)
	}
}

// Save token to authfile
func saveTokens(t *tokens) error {
	f, err := authFile()
	if err != nil {
		return err
	}

	util.RemoveFile(f)
	return util.WriteFile(f, 0600, func(w io.Writer) error {
		enc := gob.NewEncoder(w)
		return enc.Encode(t)
	})
}

// Load tokens from authfile
func loadTokens() (*tokens, error) {
	f, err := authFile()
	if err != nil {
		return nil, err
	}

	var t tokens
	return &t, util.ReadFile(f, func(r io.Reader) error {
		dec := gob.NewDecoder(r)
		return dec.Decode(&t)
	})
}

// Get auth file location
func authFile() (string, error) {
	if dir, err := util.CloudConfigFolder(); err != nil {
		return "", fmt.Errorf("could not find auth file: %v", err)
	} else {
		return filepath.Join(dir, "auth"), nil
	}
}
