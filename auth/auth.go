package auth

import (
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"time"
)

type UUID string

type Auth struct {
	signingKey []byte
	TTLHours   time.Duration
	GetTime    func() time.Time
}

var (
	InvalidToken error = errors.New("invalid-token")
	ExpiredToken error = errors.New("token-expired")
)

func NewAuth(key []byte) *Auth {
	return &Auth{
		signingKey: key,
		TTLHours:   72,
		GetTime:    func() time.Time { return time.Now() },
	}
}

// Resolve from token to app key
func (this *Auth) GetAppKey(tokenString string) (appKey UUID, err error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) ([]byte, error) {
		return this.signingKey, nil
	})

	if err == nil && token.Valid {
		// Check expiration
		exp, ok := token.Claims["exp"].(float64)
		if !ok {
			err = InvalidToken
			return
		}

		if this.GetTime().After(time.Unix(int64(exp), 0)) {
			err = ExpiredToken
			return
		}

		appKey = UUID(fmt.Sprintf("%s", token.Claims["appKey"]))
		return
	} else {
		err = InvalidToken
	}
	return
}

// Given the app key, returns the token -- used during signup
func (this *Auth) GetAppToken(appKey UUID) (tokenString string, err error) {
	token := jwt.New(jwt.GetSigningMethod("HS256"))
	// Set some claims
	token.Claims["appKey"] = appKey
	token.Claims["exp"] = time.Now().Add(time.Hour * this.TTLHours).Unix()
	// Sign and get the complete encoded token as a string
	tokenString, err = token.SignedString(this.signingKey)
	return
}
