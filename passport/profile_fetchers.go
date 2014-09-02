package passport

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

var (
	OAuth2ProfileFetchers = make(profileFetchers)
)

func init() {
	OAuth2ProfileFetchers.Register("facebook.com", facebook_fetch_profile)
}

type facebook_object map[string]interface{}

func facebook_fetch_profile(config *OAuth2AppConfig, token string) (*OAuth2Profile, error) {
	me, err := facebook_get_me(token)
	if err != nil {
		return nil, err
	}

	if v, has := me["id"]; has {
		if user_id, ok := v.(string); ok {
			return &OAuth2Profile{
				Timestamp:    time.Now(),
				Provider:     config.Provider,
				AppId:        config.AppId,
				AccountId:    user_id,
				ServiceIds:   config.ServiceIds,
				OriginalData: me,
			}, nil
		}
	}
	return nil, errors.New("cannot-parse-profile-response")
}

func facebook_get_me(token string) (me facebook_object, err error) {
	client := &http.Client{}
	url := fmt.Sprintf("https://graph.facebook.com/v2.1/me?access_token=%s", token)
	resp, err := client.Get(url)
	buff, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	me = make(facebook_object)
	err = json.Unmarshal(buff, &me)
	return
}
