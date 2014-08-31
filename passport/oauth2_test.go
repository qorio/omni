package passport

import (
	_ "code.google.com/p/goprotobuf/proto"
	"github.com/bmizerany/assert"
	"testing"
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
	f = OAuth2AccessTokenValidators.Get("test")
	assert.NotEqual(t, (OAuth2AccessTokenValidator)(nil), f)
	f = OAuth2AccessTokenValidators.Get("no.com")
	assert.Equal(t, (OAuth2AccessTokenValidator)(nil), f)
}
