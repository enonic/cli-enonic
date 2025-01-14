package common

import (
	"cli-enonic/internal/app/util"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
	"os"
	"strings"
	"time"
)

type ServiceAccountData struct {
	Algorithm    string `json:"algorithm"`
	Kid          string `json:"kid"`
	Label        string `json:"label"`
	PrincipalKey string `json:"principalKey"`
	PrivateKey   string `json:"privateKey"`
}

func hasJsonExtension(filepath string) bool {
	return strings.ToLower(filepath[len(filepath)-5:]) == ".json"
}

func parseServiceAccountData(credFilePath string) ServiceAccountData {
	if !hasJsonExtension(credFilePath) {
		util.Fatal(errors.New(fmt.Sprintf("Error: %s is not a JSON file", credFilePath)), "")
	}

	fileData, err := os.ReadFile(credFilePath)
	util.Fatal(err, fmt.Sprintf("Error reading JSON file: %v", err))

	var serviceAccountData ServiceAccountData
	if err := json.Unmarshal(fileData, &serviceAccountData); err != nil {
		util.Fatal(err, fmt.Sprintf("Error parsing JSON: %v", err))
	}
	return serviceAccountData
}

func generateServiceAccountJwtToken(credFilePath string) string {
	serviceAccountData := parseServiceAccountData(credFilePath)

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(serviceAccountData.PrivateKey))
	util.Fatal(err, fmt.Sprintf("Error parsing private key: %v", err))

	now := time.Now()

	claims := jwt.MapClaims{
		"sub": serviceAccountData.PrincipalKey,
		"iat": now.Unix(),
		"exp": now.Add(30 * time.Second).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	token.Header["kid"] = serviceAccountData.Kid

	signedToken, err := token.SignedString(privateKey)
	util.Fatal(err, fmt.Sprintf("Error signing token: %v", err))

	return signedToken
}
