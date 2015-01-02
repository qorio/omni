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

	ErrNoSignKey   = errors.New("no-sign-key")
	ErrNoVerifyKey = errors.New("no-verify-key")
)

type UUID string

// Function to override checking of flag.  This is useful for testing
// to turn off auth.
type IsAuthOn func() bool

type CheckScope func(string, []string) bool

type SignKey func() []byte
type VerifyKey func() []byte

// Sign keys and verifcation keys are both function of the input which is the http request.
type Settings struct {
	TTLHours                 time.Duration
	IsAuthOn                 IsAuthOn
	CheckScope               CheckScope
	SignKeyFromHttpRequest   func(*http.Request) []byte
	VerifyKeyFromHttpRequest func(*http.Request) []byte
}

type HttpHandler func(auth Context, resp http.ResponseWriter, req *http.Request)
type GetScopesFromToken func(*Token) []string

type Service interface {
	NewToken() (token *Token)
	SignedStringForHttpRequest(token *Token, req *http.Request) (tokenString string, err error)
	SignedString(token *Token, f SignKey) (tokenString string, err error)
	Parse(tokenString string, f VerifyKey) (token *Token, err error)
	ParseForHttpRequest(tokenString string, req *http.Request) (token *Token, err error)
	RequiresAuth(scope string, get_scopes GetScopesFromToken, handler HttpHandler) func(http.ResponseWriter, *http.Request)
}

type serviceImpl struct {
	settings   Settings
	GetTime    func() time.Time
	IsAuthOn   IsAuthOn
	CheckScope CheckScope
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
	}
}

func (this *serviceImpl) NewToken() (token *Token) {
	token = &Token{token: jwt.New(jwt.GetSigningMethod("HS256"))}
	if this.settings.TTLHours > 0 {
		token.token.Claims["exp"] = time.Now().Add(time.Hour * this.settings.TTLHours).Unix()
	}
	return
}

func (this *serviceImpl) SignedString(token *Token, f SignKey) (tokenString string, err error) {
	if f == nil {
		return "", ErrNoSignKey
	}
	tokenString, err = token.token.SignedString(f())
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

func (this *serviceImpl) Parse(tokenString string, f VerifyKey) (token *Token, err error) {
	if f == nil {
		return nil, ErrNoVerifyKey
	}
	t, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return f(), nil
	})
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
