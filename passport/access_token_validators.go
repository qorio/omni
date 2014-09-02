package passport

import (
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/golang/glog"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

var (
	OAuth2AccessTokenValidators = make(accessTokenValidators)
)

func init() {
	OAuth2AccessTokenValidators.Register("facebook.com", facebook_validate_access_token)
}

func facebook_validate_access_token(config *OAuth2AppConfig, cache OAuth2TokenCache, token string) (result *OAuth2ValidationResult, err error) {
	// Facebook
	// 1. Get a valid oauth2 token for the app itself before verifying the user access token
	app_token, err := cache.GetToken()
	if err != nil {
		app_token, err = facebook_get_app_token(config)
		if err != nil {
			return
		} else {
			go func() {
				cache.PutToken(app_token)
			}()
		}
	}
	// 2. Now call the debug endpoint to get the user id etc.
	result, err = facebook_debug_token(token, app_token)
	return
}

func facebook_get_app_token(config *OAuth2AppConfig) (token string, err error) {
	client := &http.Client{}
	url := fmt.Sprintf(
		"https://graph.facebook.com/oauth/access_token?client_secret=%s&client_id=%s&grant_type=client_credentials",
		config.AppSecret, config.AppId)
	resp, err := client.Get(url)
	if err != nil {
		return
	}
	buff, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	p := strings.Split(string(buff), "=")
	if len(p) > 1 {
		token = p[1]
		return
	} else {
		token = p[0]
		return
	}
}

func facebook_debug_token(token, appToken string) (result *OAuth2ValidationResult, err error) {
	client := &http.Client{}
	url := fmt.Sprintf(
		"https://graph.facebook.com/debug_token?input_token=%s&access_token=%s",
		token, appToken)
	resp, err := client.Get(url)
	if err != nil {
		return
	}
	buff, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	data := make(map[string]interface{})
	err = json.Unmarshal(buff, &data)
	if err != nil {
		return
	}
	// Parse the response, example:
	// {
	//     "data": {
	//         "app_id": "769962796379311",
	//         "application": "QLTest",
	//         "expires_at": 1414657567,
	//         "is_valid": true,
	//         "issued_at": 1409473567,
	//         "metadata": {
	//             "sso": "ios"
	//         },
	//         "scopes": [
	//             "public_profile"
	//         ],
	//         "user_id": "208267802676736"
	//     }
	// }
	obj := data["data"].(map[string]interface{})

	if expiry, ok := obj["expires_at"].(float64); ok {
		if expiry-float64(time.Now().Unix()) <= 0 {
			return nil, errors.New("expired-token")
		}
	}

	if app_id, ok := obj["app_id"].(string); ok {
		if user_id, ok := obj["user_id"].(string); ok {
			return &OAuth2ValidationResult{
				Provider:       "facebook.com",
				AppId:          app_id,
				AccountId:      user_id,
				ValidatedToken: token,
				Timestamp:      time.Now(),
			}, nil
		}
	}
	return nil, errors.New("error-parsing-fb-response")
}
