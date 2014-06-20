package auth

import (
	"flag"
	"fmt"
	"testing"
	"time"
)

var (
	authKeyFile = flag.String("auth_public_key_file", "", "Auth public key file")
)

func TestGetAppToken(t *testing.T) {

	id := UUID("1234")

	auth := Init(Settings{SignKey: []byte("test"), TTLHours: 0})

	token, err := auth.GetAppToken(UUID(id))
	fmt.Println("token=", token, "err=", err)

	// decode
	appKey, err := auth.GetAppKey(token)
	fmt.Println("appkey=", appKey, "err=", err)

	if appKey != id {
		t.Error("expecting", id, "but got", appKey)
	}
}

func TestGetAppTokenAuthRsaKey(t *testing.T) {

	key, e := ReadPublicKey(*authKeyFile)
	if e != nil {
		t.Fatal(e)
	}

	id := UUID("1234")

	auth := Init(Settings{SignKey: key, TTLHours: 0})

	token, err := auth.GetAppToken(UUID(id))
	fmt.Println("token=", token, "err=", err)

	// decode
	appKey, err := auth.GetAppKey(token)
	fmt.Println("appkey=", appKey, "err=", err)

	if appKey != id {
		t.Error("expecting", id, "but got", appKey)
	}
}

func TestTokenExpiration(t *testing.T) {

	id := UUID("1234")

	auth := Init(Settings{SignKey: []byte("test"), TTLHours: 1})
	auth.GetTime = func() time.Time {
		return time.Now().Add(time.Hour * 2)
	}

	token, err := auth.GetAppToken(UUID(id))
	fmt.Println("token=", token, "err=", err)

	// decode
	appKey, err := auth.GetAppKey(token)
	fmt.Println("appkey=", appKey, "err=", err)

	if err != ExpiredToken {
		t.Error("expecting", ExpiredToken, "but got", err)
	}

}
