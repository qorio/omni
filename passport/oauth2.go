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
type OAuth2AccessTokenValidator func(*OAuth2AppConfig, string) (*OAuth2ValidationResult, error)
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

type OAuth2AppStatus int

const (
	OAuth2AppStatusLive OAuth2AppStatus = iota
	OAuth2AppStatusTesting
	OAuth2AppStatusDisabled
)

// Stored in the apps collection
type OAuth2AppConfig struct {
	Status     OAuth2AppStatus
	Provider   string
	AppId      string
	AppSecret  string
	ServiceIds []string // Whitelist of services for which this app can be used to register users.
}

type OAuth2ValidationResult struct {
	Provider    string
	AccountId   string
	AppId       string
	ProfileData map[string]interface{}
	Timestamp   time.Time
}

type OAuth2Service interface {
	Close()
	SaveAppConfig(config *OAuth2AppConfig) error
	FindAppConfigByProviderAppId(provider, appId string) (*OAuth2AppConfig, error)
	ValidateToken(provider, appId, accessToken string) (*OAuth2ValidationResult, error)
}

type oauth2Impl struct {
	settings   Settings
	db         *mgo.Database
	session    *mgo.Session
	appConfigs *mgo.Collection
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

	validationResult, err := validate(config, accessToken)
	if err != nil {
		return
	}

	if validationResult.ProfileData != nil {
		// Save a copy of the profile
	}
	return
}
