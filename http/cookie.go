package http

import (
	"bytes"
	"encoding/base64"
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

type Cookies interface {
	Set(key string, value interface{}) (err error)
	Get(key string, value interface{}) (err error)
	SetPlain(key string, value interface{}) (err error)
	GetPlain(key string, value interface{}) (err error)
	SetPlainString(key, value string) (err error)
	GetPlainString(key string) (value string, err error)
}

type wrappedCookie struct {
	secureCookie *SecureCookie
	httpRequest  *http.Request
	httpResponse http.ResponseWriter
	cache        map[string]*bytes.Buffer
}

func NewCookieHandler(secureCookie *SecureCookie, resp http.ResponseWriter, request *http.Request) *wrappedCookie {
	return &wrappedCookie{
		secureCookie: secureCookie,
		httpRequest:  request,
		httpResponse: resp,
		cache:        make(map[string]*bytes.Buffer),
	}
}

func (this *wrappedCookie) Set(key string, value interface{}) (err error) {
	err = this.secureCookie.SetCookie(this.httpResponse, key, value, true)
	if err == nil {
		var buff bytes.Buffer
		enc := gob.NewEncoder(&buff)
		if err2 := enc.Encode(value); err2 == nil {
			this.cache[key] = &buff
		}
	}
	return err
}

func (this *wrappedCookie) SetPlainString(key, value string) (err error) {
	http.SetCookie(this.httpResponse, &http.Cookie{
		Name:  key,
		Value: value,
	})
	if err == nil {
		this.cache[key] = bytes.NewBufferString(value)
	}
	return err
}

func (this *wrappedCookie) SetPlain(key string, value interface{}) (err error) {
	err = this.secureCookie.SetCookie(this.httpResponse, key, value, false)
	if err == nil {
		var buff bytes.Buffer
		enc := gob.NewEncoder(&buff)
		if err2 := enc.Encode(value); err2 == nil {
			this.cache[key] = &buff
		}
	}
	return err
}

func DecodePlain(raw string, value interface{}) (err error) {
	if buff, err := base64.StdEncoding.DecodeString(raw); err == nil {
		b := bytes.NewBuffer(buff)
		dec := gob.NewDecoder(b)
		return dec.Decode(value)
	}
	return
}

func (this *wrappedCookie) Get(key string, value interface{}) (err error) {
	// return cached value if exists, this is because we know that
	// it will be set when sending http response anyway.
	if v, ok := this.cache[key]; ok {
		dec := gob.NewDecoder(v)
		return dec.Decode(value)
	}
	return this.secureCookie.ReadCookie(this.httpRequest, key, value, true)
}

func (this *wrappedCookie) GetPlain(key string, value interface{}) (err error) {
	// return cached value if exists, this is because we know that
	// it will be set when sending http response anyway.
	if v, ok := this.cache[key]; ok {
		dec := gob.NewDecoder(v)
		return dec.Decode(value)
	}
	return this.secureCookie.ReadCookie(this.httpRequest, key, value, false)
}

func (this *wrappedCookie) GetPlainString(key string) (value string, err error) {
	// return cached value if exists, this is because we know that
	// it will be set when sending http response anyway.
	if v, ok := this.cache[key]; ok {
		value = v.String()
	}
	cookie, err := this.httpRequest.Cookie(key)
	if err == nil {
		value = cookie.Value
	}
	return
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

func (this *SecureCookie) SetCookie(w http.ResponseWriter, key string, value interface{}, encrypt bool) (err error) {
	gob.Register(value)

	cookieVal := ""
	if encrypt {
		if encoded, err := this.secureCookie.Encode(key, value); err == nil {
			cookieVal = encoded
		}
	} else {
		var buff bytes.Buffer
		enc := gob.NewEncoder(&buff)
		if err2 := enc.Encode(value); err2 == nil {
			cookieVal = base64.StdEncoding.EncodeToString(buff.Bytes())
		}
	}
	cookie := &http.Cookie{
		Name:  key,
		Value: cookieVal,
	}
	if len(this.Path) > 0 {
		cookie.Path = this.Path
	}
	http.SetCookie(w, cookie)
	return
}

func (this *SecureCookie) ReadCookie(r *http.Request, key string, value interface{}, encrypt bool) (err error) {
	if cookie, err := r.Cookie(key); err == nil {
		if encrypt {
			err = this.secureCookie.Decode(key, cookie.Value, value)
		} else {
			return DecodePlain(cookie.Value, value)
		}
	}
	return
}
