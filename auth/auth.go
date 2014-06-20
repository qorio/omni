package auth

import (
	"errors"
	"flag"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"io/ioutil"
	"time"
)

var (
	doAuth = flag.Bool("auth", true, "Turns on authentication")

	InvalidToken error = errors.New("invalid-token")
	ExpiredToken error = errors.New("token-expired")
)

type UUID string

type Settings struct {
	SignKey  []byte
	TTLHours time.Duration
}

type Service struct {
	settings Settings
	GetTime  func() time.Time
}

func ReadPublicKey(filename string) (key []byte, err error) {
	// TODO -- this really isn't doing anything like parsing
	// a proper file format like X.509 or anything.

	key, err = ioutil.ReadFile(filename)
	return
}

func Init(settings Settings) *Service {
	return &Service{
		settings: settings,
		GetTime:  func() time.Time { return time.Now() },
	}
}

// Resolve from token to app key
func (this *Service) GetAppKey(tokenString string) (appKey UUID, err error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) ([]byte, error) {
		return this.settings.SignKey, nil
	})

	if err == nil && token.Valid {
		// Check expiration if there is one
		if expClaim, has := token.Claims["exp"]; has {
			exp, ok := expClaim.(float64)
			if !ok {
				err = InvalidToken
				return
			}
			if this.GetTime().After(time.Unix(int64(exp), 0)) {
				err = ExpiredToken
				return
			}
		}
		appKey = UUID(fmt.Sprintf("%s", token.Claims["appKey"]))
		return
	} else {
		err = InvalidToken
	}
	return
}

// Given the app key, returns the token -- used during signup
func (this *Service) GetAppToken(appKey UUID) (tokenString string, err error) {
	token := jwt.New(jwt.GetSigningMethod("HS256"))
	token.Claims["appKey"] = appKey
	if this.settings.TTLHours > 0 {
		token.Claims["exp"] = time.Now().Add(time.Hour * this.settings.TTLHours).Unix()
	}
	tokenString, err = token.SignedString(this.settings.SignKey)
	return
}
