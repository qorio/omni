package auth

import (
	"fmt"
	"testing"
)

func TestGetAppToken(t *testing.T) {

	id := UUID("1234")

	auth := NewAuth([]byte("test"))

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

	auth := NewAuth([]byte("test"))
	auth.TTLHours = 0

	token, err := auth.GetAppToken(UUID(id))
	fmt.Println("token=", token, "err=", err)

	// decode
	appKey, err := auth.GetAppKey(token)
	fmt.Println("appkey=", appKey, "err=", err)

	if err != ExpiredToken {
		t.Error("expecting", ExpiredToken, "but got", err)
	}

}
