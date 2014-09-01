package passport

import (
	"github.com/bmizerany/assert"
	"labix.org/v2/mgo/bson"
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
