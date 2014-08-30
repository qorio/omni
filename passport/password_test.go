package passport

import (
	"code.google.com/p/goprotobuf/proto"
	api "github.com/qorio/api/passport"
	"testing"
)

func TestPasswordHash(t *testing.T) {
	account := &api.Account{
		Primary: &api.Identity{
			Password: proto.String("password"),
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

	pass2 := Password(proto.String("password"))
	if !pass2.MatchAccount(account) {
		t.Error("Expecting password to unlock")
	}

	pass3 := Password(proto.String("badpass"))
	if pass3.MatchAccount(account) {
		t.Error("Expecting password to NOT unlock")
	}

}
