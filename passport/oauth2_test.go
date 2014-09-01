package passport

import (
	"encoding/json"
	"fmt"
	"github.com/bmizerany/assert"
	"io/ioutil"
	"labix.org/v2/mgo/bson"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestNewOAuth2Service(t *testing.T) {
	service, err := NewOAuth2Service(default_settings())
	assert.Equal(t, nil, err)

	defer service.Close()
	t.Log("Started db client", service)
	service.dropDatabase()
}

func TestSaveOAuth2AppConfig(t *testing.T) {
	service, err := NewOAuth2Service(default_settings())
	assert.Equal(t, nil, err)

	defer service.Close()
	t.Log("Started db client", service)

	service.dropDatabase()

	appConfig1 := &OAuth2AppConfig{
		Status:     OAuth2AppStatusDisabled,
		Provider:   "provider1",
		AppId:      "app1",
		AppSecret:  "secret1",
		ServiceIds: []string{"test", "passport"},
	}

	err = service.SaveAppConfig(appConfig1)
	assert.Equal(t, nil, err)

	t.Log("We should not be able to load the config since it is not live")
	config, err := service.FindAppConfigByProviderAppId("provider1", "app1")
	assert.Equal(t, (*OAuth2AppConfig)(nil), config)
	assert.Equal(t, ERROR_NOT_FOUND, err)

	t.Log("Update the config and save")
	appConfig1.Status = OAuth2AppStatusLive
	err = service.SaveAppConfig(appConfig1)
	assert.Equal(t, nil, err)

	config, err = service.FindAppConfigByProviderAppId("provider1", "app1")
	assert.NotEqual(t, (*OAuth2AppConfig)(nil), config)
	assert.Equal(t, OAuth2AppStatusLive, config.Status)
	assert.Equal(t, appConfig1.Provider, config.Provider)
	assert.Equal(t, appConfig1, config)
}

func TestOAuth2GetValidators(t *testing.T) {
	f := OAuth2AccessTokenValidators.Get("facebook.com")
	assert.NotEqual(t, (OAuth2AccessTokenValidator)(nil), f)
	f = OAuth2AccessTokenValidators.Get("no.com")
	assert.Equal(t, (OAuth2AccessTokenValidator)(nil), f)
}

func TestOAuth2ValidateToken(t *testing.T) {
	service, err := NewOAuth2Service(default_settings())
	assert.Equal(t, nil, err)

	defer service.Close()
	t.Log("Started db client", service)

	service.dropDatabase()

	appConfig1 := &OAuth2AppConfig{
		Status:     OAuth2AppStatusLive,
		Provider:   "oauth2.test.com",
		AppId:      "app1",
		AppSecret:  "secret1",
		ServiceIds: []string{"test"},
	}

	err = service.SaveAppConfig(appConfig1)
	assert.Equal(t, nil, err)
	t.Log("Must be able to access a live app config")
	config, err := service.FindAppConfigByProviderAppId("oauth2.test.com", "app1")
	assert.NotEqual(t, (*OAuth2AppConfig)(nil), config)

	test_result := &struct {
		CalledValidator bool
	}{
		false,
	}

	profile_data := map[string]interface{}{
		"name":  "test",
		"photo": "http://cdn.com/1234.png",
	}
	// Register the validator
	OAuth2AccessTokenValidators.Register("oauth2.test.com", func(config *OAuth2AppConfig, token string) (r *OAuth2ValidationResult, err error) {
		test_result.CalledValidator = true
		assert.Equal(t, *appConfig1, *config)
		return &OAuth2ValidationResult{
			Provider:    "oauth2.test.com",
			AppId:       "app1",
			AccountId:   "user1",
			Timestamp:   time.Now(),
			ProfileData: profile_data,
		}, nil
	})

	result, err := service.ValidateToken("oauth2.test.com", "app1", "token")
	assert.Equal(t, nil, err)
	assert.NotEqual(t, nil, result)
	assert.Equal(t, true, test_result.CalledValidator)

	t.Log("We expect a profile to be saved as well.")
	profile, err := service.FindProfileByProviderAccountId("oauth2.test.com", "user1")
	assert.Equal(t, nil, err)

	m := profile.OriginalData.(bson.M)
	assert.Equal(t, profile_data["name"], m["name"])
	assert.Equal(t, profile_data["photo"], m["photo"])
	if profile.ServiceIds[0] != "test" {
		t.Fatal("expects test")
	}
}

func TestOAuth2ValidateTokenFacebook(t *testing.T) {
	service, err := NewOAuth2Service(default_settings())
	assert.Equal(t, nil, err)

	defer service.Close()
	t.Log("Started db client", service)

	service.dropDatabase()

	appConfig1 := &OAuth2AppConfig{
		Status:     OAuth2AppStatusLive,
		Provider:   "facebook.com",
		AppId:      "769962796379311",
		AppSecret:  "7fb04adfc64c6c154eca2395f0191540",
		ServiceIds: []string{"test"},
	}

	err = service.SaveAppConfig(appConfig1)
	assert.Equal(t, nil, err)

	OAuth2AccessTokenValidators.Register("facebook.com",
		func(config *OAuth2AppConfig, token string) (r *OAuth2ValidationResult, err error) {
			// Facebook
			// 1. Get a valid oauth2 token for the app itself before verifying the user access token
			client := &http.Client{}
			url := fmt.Sprintf(
				"https://graph.facebook.com/oauth/access_token?client_secret=%s&client_id=%s&grant_type=client_credentials",
				config.AppSecret, config.AppId)

			get, err := http.NewRequest("GET", url, nil)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := client.Do(get)
			if err != nil {
				t.Fatal(err)
			}
			buff, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			app_token := strings.Split(string(buff), "=")[1]
			t.Log("Got access token for app", app_token)
			assert.NotEqual(t, "", app_token)

			// 2. Now call the debug endpoint to get the user id etc.
			url = fmt.Sprintf(
				"https://graph.facebook.com/debug_token?input_token=%s&access_token=%s",
				token, app_token)
			get, err = http.NewRequest("GET", url, nil)
			if err != nil {
				t.Fatal(err)
			}
			resp, err = client.Do(get)
			if err != nil {
				t.Fatal(err)
			}
			buff, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			data := make(map[string]interface{})
			err = json.Unmarshal(buff, &data)
			if err != nil {
				t.Fatal(err)
			}

			obj := data["data"].(map[string]interface{})

			var profile map[string]interface{}
			if obj["expires_at"].(float64)-float64(time.Now().Unix()) > 0 {
				t.Log("expires", obj["expires_at"].(float64)-float64(time.Now().Unix()))

				// 3. Fetch the me object
				url = fmt.Sprintf("https://graph.facebook.com/v2.1/me?access_token=%s", token)
				get, err = http.NewRequest("GET", url, nil)
				if err != nil {
					t.Fatal(err)
				}
				resp, err = client.Do(get)
				if err != nil {
					t.Fatal(err)
				}
				buff, err = ioutil.ReadAll(resp.Body)
				if err != nil {
					t.Fatal(err)
				}
				profile = make(map[string]interface{})
				err = json.Unmarshal(buff, &profile)
				if err != nil {
					t.Fatal(err)
				}
				t.Log("Got me:", profile)
			}
			return &OAuth2ValidationResult{
				Provider:    "facebook.com",
				AppId:       obj["app_id"].(string),
				AccountId:   obj["user_id"].(string),
				Timestamp:   time.Now(),
				ProfileData: profile,
			}, nil
		})

	result, err := service.ValidateToken("facebook.com", "769962796379311",
		"CAAK8Ru737K8BAOCuulle6JdrZBB7T3qgZBLl5iNpZB0miSZBygHGwFnQsvBLe3QUG7bLMxBRrsZAn1ZBay5WLNuHDrusL4R3Tpdt3agTmIgKlCxeOCxZBGQlUa8Td4b3m11NMmZBVwDW47Ry357LNS4ZCNVHeLtkOTcGcM0bsO5yICn2ZCQP254ne6ZBnUQBeNKsa6Urrb1Q0xA52IB1wlPK592jM1RAgHEODUZD")
	assert.Equal(t, nil, err)
	assert.NotEqual(t, nil, result)

	if result.ProfileData != nil {
		t.Log("We expect a profile to be saved as well.")
		profile, err := service.FindProfileByProviderAccountId(result.Provider, result.AccountId)
		assert.Equal(t, nil, err)
		if profile.ServiceIds[0] != "test" {
			t.Fatal("expects test")
		}

		m := profile.OriginalData.(bson.M)
		t.Log("Facebook profile", m)
		assert.Equal(t, "Qorio", m["last_name"])
		assert.Equal(t, "TestOne", m["first_name"])
		assert.Equal(t, "male", m["gender"])
	}
}
