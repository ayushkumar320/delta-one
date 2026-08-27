// Package token issues and verifies the access tokens used across the
// platform. The auth service issues them; the gateway verifies them.
package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TTL is how long an issued access token stays valid.
const TTL = 24 * time.Hour

const issuer = "delta-one-auth"

// Claims is the payload carried by an access token.
type Claims struct {
	Email string `json:"email"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

// ErrInvalid is returned for any token that fails verification. The reason is
// deliberately not exposed to the caller.
var ErrInvalid = errors.New("invalid token")

// Issue signs an access token for the given user.
func Issue(secret, userID, email, role string) (string, time.Time, error) {
	expiry := time.Now().Add(TTL)
	claims := Claims{
		Email: email,
		Role:  role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expiry),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiry, nil
}

// Verify parses and validates a token. The signing method is pinned to HMAC so
// a token claiming "alg":"none" or an RSA public key cannot be substituted.
func Verify(secret, raw string) (*Claims, error) {
	var claims Claims
	_, err := jwt.ParseWithClaims(raw, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalid
		}
		return []byte(secret), nil
	}, jwt.WithIssuer(issuer), jwt.WithExpirationRequired(), jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, ErrInvalid
	}
	return &claims, nil
}
