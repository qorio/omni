package passport

import (
	"github.com/bmizerany/assert"
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

func TestOAuth2FindAppConfigByServiceAndProvider(t *testing.T) {
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
	config, err := service.FindAppConfigByServiceAndProvider("passport", "provider1")
	assert.Equal(t, (*OAuth2AppConfig)(nil), config)
	assert.Equal(t, ERROR_NOT_FOUND, err)

	t.Log("Update the config to make it live and try again.")
	appConfig1.Status = OAuth2AppStatusLive
	err = service.SaveAppConfig(appConfig1)
	assert.Equal(t, nil, err)

	config, err = service.FindAppConfigByServiceAndProvider("passport", "provider1")
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

	called := new(bool)
	profile_data := map[string]interface{}{
		"name":  "test",
		"photo": "http://cdn.com/1234.png",
	}
	// Register the validator
	OAuth2AccessTokenValidators.Register("oauth2.test.com",
		func(config *OAuth2AppConfig, cache OAuth2TokenCache, token string) (r *OAuth2ValidationResult, err error) {
			*called = true
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
	assert.Equal(t, true, *called)
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
	result, err := service.ValidateToken("facebook.com", "769962796379311",
		"CAAK8Ru737K8BAOCuulle6JdrZBB7T3qgZBLl5iNpZB0miSZBygHGwFnQsvBLe3QUG7bLMxBRrsZAn1ZBay5WLNuHDrusL4R3Tpdt3agTmIgKlCxeOCxZBGQlUa8Td4b3m11NMmZBVwDW47Ry357LNS4ZCNVHeLtkOTcGcM0bsO5yICn2ZCQP254ne6ZBnUQBeNKsa6Urrb1Q0xA52IB1wlPK592jM1RAgHEODUZD")
	assert.Equal(t, nil, err)
	assert.NotEqual(t, nil, result)
	assert.NotEqual(t, "", result.ValidatedToken)
	assert.Equal(t, appConfig1.AppId, result.AppId)
}
