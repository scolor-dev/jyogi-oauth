package oauth

import (
	"crypto/sha256"
	"encoding/base64"
	"regexp"
)

var codeVerifierRegex = regexp.MustCompile(`^[A-Za-z0-9\-._~]{43,128}$`)

func VerifyPKCE(codeVerifier, codeChallenge, method string) bool {
	if method != "S256" {
		return false
	}
	if !codeVerifierRegex.MatchString(codeVerifier) {
		return false
	}
	h := sha256.Sum256([]byte(codeVerifier))
	computed := base64.RawURLEncoding.EncodeToString(h[:])
	return computed == codeChallenge
}
