package passport

import (
	"errors"
	"github.com/golang/glog"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"strings"
	"time"
)

// Function to validate the access token
type OAuth2AccessTokenValidator func(*OAuth2AppConfig, OAuth2TokenCache, string) (*OAuth2ValidationResult, error)
type accessTokenValidators map[string]OAuth2AccessTokenValidator

func (this accessTokenValidators) Register(provider string, f OAuth2AccessTokenValidator) {
	this[provider] = f
}

func (this accessTokenValidators) Get(provider string) OAuth2AccessTokenValidator {
	if f, exists := this[provider]; exists {
		return f
	} else {
		return nil
	}
}

type OAuth2ProfileFetcher func(*OAuth2AppConfig, string) (*OAuth2Profile, error)
type profileFetchers map[string]OAuth2ProfileFetcher

func (this profileFetchers) Register(provider string, f OAuth2ProfileFetcher) {
	this[provider] = f
}

func (this profileFetchers) Get(provider string) OAuth2ProfileFetcher {
	if f, exists := this[provider]; exists {
		return f
	} else {
		return nil
	}
}

type OAuth2TokenCache interface {
	GetToken() (string, error) // Returns empty string and error if not found
	PutToken(string)
}

type OAuth2AppStatus int

const (
	OAuth2AppStatusLive OAuth2AppStatus = iota
	OAuth2AppStatusTesting
	OAuth2AppStatusDisabled
)

type OAuth2ValidationResult struct {
	Provider       string
	AccountId      string
	AppId          string
	ProfileData    map[string]interface{}
	Timestamp      time.Time
	ValidatedToken string
}

type OAuth2Service interface {
	Close()
	SaveAppConfig(config *OAuth2AppConfig) error

	/// Find the config by provider and app ids
	FindAppConfigByProviderAppId(provider, appId string) (*OAuth2AppConfig, error)

	/// Service is passport service, provider is the oauth provider
	FindAppConfigByServiceAndProvider(serviceId, provider string) (*OAuth2AppConfig, error)

	ValidateToken(provider, appId, accessToken string) (*OAuth2ValidationResult, error)

	FetchProfile(provider, appId, accessToken string) (*OAuth2Profile, error)

	SaveProfile(profile *OAuth2Profile) error

	FindProfileByProviderAccountId(provider, accountId string) (*OAuth2Profile, error)
}

// Stored in the apps collection
type OAuth2AppConfig struct {
	Status     OAuth2AppStatus
	Provider   string
	AppId      string
	AppSecret  string
	ServiceIds []string // Whitelist of services for which this app can be used to register users.
}

// Stored in the profiles collection
type OAuth2Profile struct {
	Timestamp    time.Time
	Provider     string
	AppId        string
	AccountId    string
	ServiceIds   []string // list of services permitted to read this profile
	OriginalData interface{}
}

type oauth2Impl struct {
	settings   Settings
	db         *mgo.Database
	session    *mgo.Session
	appConfigs *mgo.Collection
	profiles   *mgo.Collection
}

func NewOAuth2Service(settings Settings) (*oauth2Impl, error) {

	impl := &oauth2Impl{
		settings: settings,
	}

	var err error
	impl.session, err = mgo.Dial(strings.Join(settings.Mongo.Hosts, ","))
	if err != nil {
		return nil, err
	}
	// Optional. Switch the session to a monotonic behavior.
	impl.session.SetMode(mgo.Monotonic, true)
	impl.session.SetSafe(&mgo.Safe{})

	impl.db = impl.session.DB(settings.Mongo.Db)
	impl.appConfigs = impl.db.C("oauth2_app_configs")
	impl.appConfigs.EnsureIndex(mgo.Index{
		Key:      []string{"status", "provider", "appid"},
		Unique:   true,
		DropDups: true,
		Sparse:   true,
		Name:     "status_provider_app_id",
	})
	impl.appConfigs.EnsureIndex(mgo.Index{
		Key:      []string{"status", "provider", "serviceids"},
		Unique:   true,
		DropDups: true,
		Sparse:   true,
		Name:     "status_provider_service_id",
	})

	impl.profiles = impl.db.C("oauth2_profiles")
	impl.profiles.EnsureIndex(mgo.Index{
		Key:      []string{"provider", "accountid"},
		Unique:   true,
		DropDups: true,
		Sparse:   true,
		Name:     "provider_account_id",
	})
	glog.Infoln("OAuth2 Service MongoDb backend initialized:", impl)
	return impl, nil
}

func (this *oauth2Impl) dropDatabase() (err error) {
	return this.db.DropDatabase()
}

func (this *oauth2Impl) Close() {
	this.session.Close()
	glog.Infoln("Session closed", this.session)
}

func (this *oauth2Impl) SaveAppConfig(config *OAuth2AppConfig) error {
	changeInfo, err := this.appConfigs.Upsert(bson.M{
		"status":   config.Status,
		"provider": config.Provider,
		"appid":    config.AppId,
	}, config)
	if changeInfo != nil && changeInfo.Updated >= 0 {
		return nil
	}
	return err
}

func (this *oauth2Impl) SaveProfile(profile *OAuth2Profile) error {
	changeInfo, err := this.profiles.Upsert(bson.M{
		"provider":  profile.Provider,
		"accountid": profile.AccountId,
	}, profile)
	if changeInfo != nil && changeInfo.Updated >= 0 {
		return nil
	}
	return err
}

func (this *oauth2Impl) FindProfileByProviderAccountId(provider, accountId string) (*OAuth2Profile, error) {
	result := OAuth2Profile{}
	err := this.profiles.Find(bson.M{
		"provider":  provider,
		"accountid": accountId}).One(&result)
	switch {
	case err == mgo.ErrNotFound:
		return nil, ERROR_NOT_FOUND
	case err != nil:
		return nil, err
	}
	return &result, nil
}

func (this *oauth2Impl) FindAppConfigByProviderAppId(provider, appId string) (*OAuth2AppConfig, error) {
	result := OAuth2AppConfig{}
	err := this.appConfigs.Find(bson.M{
		"status":   OAuth2AppStatusLive,
		"provider": provider,
		"appid":    appId}).One(&result)
	switch {
	case err == mgo.ErrNotFound:
		return nil, ERROR_NOT_FOUND
	case err != nil:
		return nil, err
	}
	return &result, nil
}

func (this *oauth2Impl) FindAppConfigByServiceAndProvider(serviceId, provider string) (*OAuth2AppConfig, error) {
	result := OAuth2AppConfig{}
	err := this.appConfigs.Find(bson.M{
		"status":     OAuth2AppStatusLive,
		"provider":   provider,
		"serviceids": serviceId}).One(&result)
	switch {
	case err == mgo.ErrNotFound:
		return nil, ERROR_NOT_FOUND
	case err != nil:
		return nil, err
	}
	return &result, nil
}

// Implements TokenCache
type tokenCache struct {
	appConfig *OAuth2AppConfig
	impl      *oauth2Impl
}

var appAccessTokenMap = make(map[string]string)

func (this *tokenCache) GetToken() (token string, err error) {
	key := this.appConfig.Provider + this.appConfig.AppId
	if token, has := appAccessTokenMap[key]; has {
		return token, nil
	}
	return "", ERROR_NOT_FOUND
}

func (this *tokenCache) PutToken(token string) {
	key := this.appConfig.Provider + this.appConfig.AppId
	appAccessTokenMap[key] = token
}

func (this *oauth2Impl) ValidateToken(provider, appId, accessToken string) (result *OAuth2ValidationResult, err error) {
	config, err := this.FindAppConfigByProviderAppId(provider, appId)

	if err != nil {
		return
	}

	// Look up the validator
	validate := OAuth2AccessTokenValidators.Get(config.Provider)
	if validate == nil {
		err = errors.New("unknown-provider-cannot-validate-token")
	}

	result, err = validate(config, &tokenCache{appConfig: config, impl: this}, accessToken)
	if err != nil {
		return
	}
	if result == nil {
		err = errors.New("internal-error-bad-validator")
		return
	}

	if result.AppId != config.AppId {
		// Something is wrong. The token is from another app registered
		// with the OAuth provider.
		err = errors.New("token-not-granted-for-app")
		return
	}
	return
}

func (this *oauth2Impl) FetchProfile(provider, appId, accessToken string) (*OAuth2Profile, error) {
	config, err := this.FindAppConfigByProviderAppId(provider, appId)

	if err != nil {
		return nil, err
	}

	// Look up the profile fetcher
	fetch := OAuth2ProfileFetchers.Get(config.Provider)
	if fetch == nil {
		err = errors.New("unknown-provider-cannot-fetch-profile")
		return nil, err
	}

	result, err := fetch(config, accessToken)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, errors.New("internal-error-bad-fetcher")
	}
	return result, nil
}
