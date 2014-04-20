package http

import (
	"encoding/gob"
	"errors"
	"github.com/gorilla/securecookie"
	"net/http"
)

type SecureCookie struct {
	hmacKey      []byte
	encryptKey   []byte
	secureCookie *securecookie.SecureCookie
	Path         string
}

func NewSecureCookie(hmacKey []byte, optionalEncryptionKey []byte) (sc *SecureCookie, err error) {
	if hmacKey == nil {
		return nil, errors.New("requires hmac key")
	}

	var ec []byte = nil
	if optionalEncryptionKey != nil && len(optionalEncryptionKey) > 0 {
		ec = optionalEncryptionKey
	}
	return &SecureCookie{
		hmacKey:      hmacKey,
		encryptKey:   ec,
		secureCookie: securecookie.New(hmacKey, ec)}, nil
}

func (this *SecureCookie) HmacKeyString() string {
	return string(this.hmacKey)
}

func (this *SecureCookie) EncryptKeyString() string {
	return string(this.encryptKey)
}

func (this *SecureCookie) SetCookie(w http.ResponseWriter, key string, value interface{}) (err error) {
	gob.Register(value)
	if encoded, err := this.secureCookie.Encode(key, value); err == nil {
		cookie := &http.Cookie{
			Name:  key,
			Value: encoded,
		}
		if len(this.Path) > 0 {
			cookie.Path = this.Path
		}
		http.SetCookie(w, cookie)
	}
	return
}

func (this *SecureCookie) ReadCookie(r *http.Request, key string, value interface{}) (err error) {
	if cookie, err := r.Cookie(key); err == nil {
		err = this.secureCookie.Decode(key, cookie.Value, value)
	}
	return
}
