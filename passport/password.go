package passport

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	api "github.com/qorio/api/passport"
)

type password struct {
	str  **string
	hash string
}

func Password(str *string) *password {
	return &password{
		str: &str,
	}
}

func (this *password) MatchAccount(account *api.Account) bool {
	if *this.str != nil {
		this.hash = hmacSha256String(**this.str)
		return account.Primary.GetPassword() == this.hash
	}
	return account.Primary.Password == nil
}

func (this *password) Hash() *password {
	if *this.str != nil {
		this.hash = hmacSha256String(**this.str)
		**this.str = this.hash
	}
	return this
}

func (this *password) Update() {
	if *this.str != nil {
		**this.str = this.hash
	}
}

var secret = []byte("this needs to stay constant and never change; we only use this for hmac")

func hmacSha256(input []byte) (h []byte) {
	mac := hmac.New(sha256.New, secret)
	mac.Write(input)
	h = mac.Sum(nil)
	return
}

func hmacSha256String(input string) (h string) {
	buff := hmacSha256([]byte(input))
	return base64.StdEncoding.EncodeToString(buff)
}
