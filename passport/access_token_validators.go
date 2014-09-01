package passport

import (
	_ "github.com/golang/glog"
)

var (
	OAuth2AccessTokenValidators = make(accessTokenValidators)
)

func init() {
	OAuth2AccessTokenValidators.Register("facebook.com", facebook_validate_access_token)
}

func facebook_validate_access_token(config *OAuth2AppConfig, token string) (result *OAuth2ValidationResult, err error) {
	return
}
