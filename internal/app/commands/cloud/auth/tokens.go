package auth

import (
	"encoding/gob"
	"io"
	"os"
	"path/filepath"
	"time"

	util "github.com/enonic/cli-enonic/internal/app/commands/cloud/util"
	"github.com/enonic/cli-enonic/internal/app/commands/common"
)

type tokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    int64
}

func (t tokens) isExpired() bool {
	return time.Now().Add(time.Minute*5).Unix() > t.ExpiresAt
}

func authFile() string {
	return filepath.Join(common.GetEnonicDir(), "auth")
}

func authFileExists() bool {
	info, err := os.Stat(authFile())
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func Logout() error {
	return util.RemoveFile(authFile())
}

func IsLoggedIn() bool {
	_, err := GetAccessToken()
	return err == nil
}

func GetAccessToken() (string, error) {
	t, err := loadTokens()
	if err != nil {
		return "", err
	}
	if t.isExpired() {
		t, err := oAuthRefreshTokens(t)
		if err != nil {
			return "", err
		}
		err = saveTokens(t)
		if err != nil {
			return "", err
		}
	}
	return t.AccessToken, nil
}

func saveTokens(t *tokens) error {
	f := authFile()
	util.RemoveFile(f)

	return util.WriteFile(f, 0600, func(w io.Writer) error {
		enc := gob.NewEncoder(w)
		return enc.Encode(t)
	})
}

func loadTokens() (*tokens, error) {
	f := authFile()

	var t tokens
	err := util.ReadFile(f, func(r io.Reader) error {
		dec := gob.NewDecoder(r)
		return dec.Decode(&t)
	})

	return &t, err
}
