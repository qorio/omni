package auth

import (
	"fmt"
	"testing"
	"time"
)

func TestGetAppToken(t *testing.T) {

	id := UUID("1234")

	auth := NewService([]byte("test"))
	auth.TTLHours = 0 // no expiration

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

	auth := NewService(key)
	auth.TTLHours = 0 // no expiration

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

	auth := NewService([]byte("test"))
	auth.TTLHours = 1
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
