package passport

import (
	"github.com/golang/glog"
	"labix.org/v2/mgo"
	_ "labix.org/v2/mgo/bson"
	"strings"
)

type OAuth2ValidationResult struct {
	Provider    string
	AccountId   string
	AppId       string
	ProfileData map[string]interface{}
}

type OAuth2Service interface {
	ValidateToken(provider, appId, accessToken string) (*OAuth2ValidationResult, error)
}

type oauth2Impl struct {
	settings Settings
	db       *mgo.Database
	session  *mgo.Session
	apps     *mgo.Collection
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
	impl.apps = impl.db.C("oauth2_apps")
	impl.apps.EnsureIndex(mgo.Index{
		Key:      []string{"provider", "app_id"},
		Unique:   true,
		DropDups: true,
		Sparse:   true,
		Name:     "provider_app_id",
	})
	glog.Infoln("OAuth2 Service MongoDb backend initialized:", impl)
	return impl, nil
}

func (this *oauth2Impl) ValidateToken(provider, appId, accessToken string) (*OAuth2ValidationResult, error) {
	return nil, nil
}
