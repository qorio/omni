package passport

import (
	api "github.com/qorio/api/passport"
	"testing"
)

func TestPasswordHash(t *testing.T) {

	account := &api.Account{
		Primary: &api.Login{
			Password: ptr("password"),
		},
	}

	prev := account.Primary.GetPassword()
	pass := Password(account.Primary.Password)
	pass.Hash().Update()

	after := account.Primary.GetPassword()

	t.Log("before", prev, "after", after)
	if prev == after {
		t.Error("Expecting a hashed value:", after)
	}

	pass2 := Password(ptr("password"))
	if !pass2.MatchAccount(account) {
		t.Error("Expecting password to unlock")
	}

	pass3 := Password(ptr("badpass"))
	if pass3.MatchAccount(account) {
		t.Error("Expecting password to NOT unlock")
	}

}
