package auth

import (
	"flag"
	"reflect"
	"testing"
	"time"
)

var (
	authKeyFile = flag.String("auth_public_key_file", "", "Auth public key file")
)

func TestNewToken(t *testing.T) {

	SignKey := func() []byte { return []byte("test") }
	VerifyKey := func() []byte { return []byte("test") }

	auth := Init(Settings{TTLHours: 0})

	token := auth.NewToken()
	token.Add("foo1", "foo1").Add("foo2", "foo2").Add("count", 2)

	signedString, err := auth.SignedString(token, SignKey)
	if err != nil {
		t.Error(err)
	}

	t.Log("token=", signedString)
	parsed, err := auth.Parse(signedString, VerifyKey)
	if err != nil {
		t.Error(err)
	}

	if !parsed.HasKey("count") {
		t.Error("Should have count")
	}

	if float64(2) != parsed.Get("count") {
		t.Error("Should be 2", parsed.Get("count"),
			reflect.TypeOf(parsed.Get("count")))
	}

	if "foo1" != parsed.Get("foo1") {
		t.Error("Should be foo1", parsed.Get("foo1"))
	}

}

func TestTokenExpiration(t *testing.T) {

	SignKey := func() []byte { return []byte("test") }
	VerifyKey := func() []byte { return []byte("test") }
	auth := Init(Settings{TTLHours: 1})
	auth.GetTime = func() time.Time {
		return time.Now().Add(time.Hour * 2)
	}

	token := auth.NewToken()
	encoded, err := auth.SignedString(token, SignKey)

	// decode
	_, err = auth.Parse(encoded, VerifyKey)

	if err != ExpiredToken {
		t.Error("expecting", ExpiredToken, "but got", err)
	}
}

func TestGetAppTokenAuthRsaKey(t *testing.T) {

	key, e := ReadPublicKey(*authKeyFile)
	if e != nil {
		t.Fatal(e)
	}

	SignKey := func() []byte { return key }
	VerifyKey := func() []byte { return key }
	id := UUID("1234")
	auth := Init(Settings{TTLHours: 0})

	token := auth.NewToken().Add("appKey", id)
	encoded, err := auth.SignedString(token, SignKey)

	// decode
	parsed, err := auth.Parse(encoded, VerifyKey)
	if err != nil {
		t.Error(err)
	}

	appKey := parsed.GetString("appKey")
	t.Log("appkey=", appKey)

	if UUID(appKey) != id {
		t.Error("expecting", id, "but got", appKey)
	}
}
