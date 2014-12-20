package auth

import (
	"errors"
	"flag"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"io/ioutil"
	"net/http"
	"time"
)

var (
	doAuth = flag.Bool("auth", true, "Turns on authentication")

	InvalidToken error = errors.New("invalid-token")
	ExpiredToken error = errors.New("token-expired")
)

type UUID string

// Function to override checking of flag.  This is useful for testing
// to turn off auth.
type IsAuthOn func() bool

type CheckScope func(string, []string) bool

type Settings struct {
	SignKey    []byte
	TTLHours   time.Duration
	IsAuthOn   IsAuthOn
	CheckScope CheckScope
}

type HttpHandler func(auth Context, resp http.ResponseWriter, req *http.Request)
type GetScopesFromToken func(*Token) []string

type Service interface {
	NewToken() (token *Token)
	SignedString(token *Token) (tokenString string, err error)
	Parse(tokenString string) (token *Token, err error)
	RequiresAuth(scope string, get_scopes GetScopesFromToken, handler HttpHandler) func(http.ResponseWriter, *http.Request)
}

type serviceImpl struct {
	settings   Settings
	GetTime    func() time.Time
	IsAuthOn   IsAuthOn
	CheckScope CheckScope
	KeyFunc    func(t *jwt.Token) (interface{}, error)
}

type Token struct {
	token *jwt.Token
}

func ReadPublicKey(filename string) (key []byte, err error) {
	// TODO -- this really isn't doing anything like parsing
	// a proper file format like X.509 or anything.

	key, err = ioutil.ReadFile(filename)
	return
}

func Init(settings Settings) *serviceImpl {
	return &serviceImpl{
		settings:   settings,
		GetTime:    func() time.Time { return time.Now() },
		IsAuthOn:   settings.IsAuthOn,
		CheckScope: settings.CheckScope,
		KeyFunc: func(t *jwt.Token) (interface{}, error) {
			return settings.SignKey, nil
		},
	}
}

func (this *serviceImpl) NewToken() (token *Token) {
	token = &Token{token: jwt.New(jwt.GetSigningMethod("HS256"))}
	if this.settings.TTLHours > 0 {
		token.token.Claims["exp"] = time.Now().Add(time.Hour * this.settings.TTLHours).Unix()
	}
	return
}

func (this *serviceImpl) SignedString(token *Token) (tokenString string, err error) {
	tokenString, err = token.token.SignedString(this.settings.SignKey)
	return
}

func (this *serviceImpl) check_token(t *jwt.Token) (*Token, error) {
	if t == nil || !t.Valid {
		return nil, InvalidToken
	}
	// Check expiration if there is one
	if expClaim, has := t.Claims["exp"]; has {
		exp, ok := expClaim.(float64)
		if !ok {
			return nil, InvalidToken
		}
		if this.GetTime().After(time.Unix(int64(exp), 0)) {
			return nil, ExpiredToken
		}
	}
	return &Token{token: t}, nil
}

func (this *serviceImpl) Parse(tokenString string) (token *Token, err error) {
	// t, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
	// 	return this.settings.SignKey, nil
	// })

	t, err := jwt.Parse(tokenString, this.KeyFunc)
	if err != nil {
		return nil, err
	}
	return this.check_token(t)
}

func (this *Token) Add(key string, value interface{}) *Token {
	this.token.Claims[key] = value
	return this
}

func (this *Token) Get(key string) interface{} {
	if v, has := this.token.Claims[key]; has {
		return v
	}
	return nil
}

func (this *Token) GetString(key string) string {
	if v := this.Get(key); v == nil {
		return ""
	} else {
		return fmt.Sprintf("%s", v)
	}
}

func (this *Token) HasKey(key string) bool {
	if _, has := this.token.Claims[key]; has {
		return true
	}
	return false
}
