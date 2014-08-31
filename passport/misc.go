package passport

import (
	"errors"
	api "github.com/qorio/api/passport"
)

func validate_login(login *api.Identity) error {
	switch {
	case login.Password != nil:
		if login.Phone == nil && login.Email == nil && login.Username == nil {
			return errors.New("missing-identifier")
		}
		if login.Password == nil {
			return errors.New("missing-password")
		}
	case login.Oauth2AccessToken != nil:
	}
	return nil
}

// For removing sensitive information before sending back to client
func sanitize(account *api.Account) *api.Account {
	if account.Primary == nil {
		return account
	}

	login := account.Primary

	if login.GetEmail() == account.GetId() {
		login.Email = nil
	}
	if login.GetPhone() == account.GetId() {
		login.Phone = nil
	}
	if login.GetUsername() == account.GetId() {
		login.Username = nil
	}
	if login.GetOauth2AccountId() == account.GetId() {
		login.Oauth2AccountId = nil
	}
	login.Password = nil
	return account
}
