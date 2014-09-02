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

	account.Primary.Password = nil
	account.Primary.Oauth2AccessToken = nil

	return account
}

/// Do we have enough information in the account object to find it
/// in our database?
func is_account_findable(account *api.Account) bool {
	if account.Primary == nil {
		return false
	}
	if account.Primary.Email != nil || account.Primary.Username != nil || account.Primary.Phone != nil {
		return true
	}
	if account.Primary.Oauth2AccountId != nil {
		return true
	}
	return false
}
