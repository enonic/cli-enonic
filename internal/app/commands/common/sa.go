package common

import (
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"
	"log"
	"os"
	"path/filepath"
	"time"
)

type ServiceAccountData struct {
	Algorithm    string `json:"algorithm"`
	Kid          string `json:"kid"`
	Label        string `json:"label"`
	PrincipalKey string `json:"principalKey"`
	PrivateKey   string `json:"privateKey"`
}

func ServiceAccountConfigFolder() (string, error) {
	folder := GetInEnonicDir("sa")
	return folder, os.MkdirAll(folder, os.ModeDir|os.ModePerm)
}

func loadServiceAccountFile(filename string) (string, error) {
	path, err := serviceAccountFile(filename)
	if err != nil {
		return "", errors.New(fmt.Sprintf("could not find '%s' file: %v", filename, err))
	}

	fileData, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Error reading JSON file: %v", err)
	}

	var serviceAccountData ServiceAccountData
	if err := json.Unmarshal(fileData, &serviceAccountData); err != nil {
		log.Fatalf("Error parsing JSON: %v", err)
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(serviceAccountData.PrivateKey))
	if err != nil {
		log.Fatalf("Error parsing private key: %v", err)
	}

	claims := jwt.MapClaims{
		"sub": serviceAccountData.PrincipalKey,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(30 * time.Second).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	token.Header["kid"] = serviceAccountData.Kid

	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		log.Fatalf("Error signing token: %v", err)
	}

	return signedToken, nil
}

func serviceAccountFile(filename string) (string, error) {
	if dir, err := ServiceAccountConfigFolder(); err != nil {
		return "", fmt.Errorf("could not find '%s' file: %v", filename, err)
	} else {
		return filepath.Join(dir, filename), nil
	}
}
