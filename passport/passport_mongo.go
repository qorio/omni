package passport

import (
	"code.google.com/p/go-uuid/uuid"
	"github.com/golang/glog"
	api "github.com/qorio/api/passport"
	omni_common "github.com/qorio/omni/common"
	omni_rest "github.com/qorio/omni/rest"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"strings"
)

type serviceImpl struct {
	settings Settings
	db       *mgo.Database
	session  *mgo.Session
	accounts *mgo.Collection
	webhooks *mgo.Collection
}

type mgo_webhook struct {
	Id      bson.ObjectId `bson:"_id"`
	Service string
	Map     omni_rest.EventKeyUrlMap
}

func NewService(settings Settings) (*serviceImpl, error) {

	impl := &serviceImpl{
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
	impl.accounts = impl.db.C("accounts")
	// 2d spatial index on primary login's location
	impl.accounts.EnsureIndex(mgo.Index{
		Key:      []string{"primary.location"},
		Unique:   false,
		DropDups: false,
		Sparse:   true,
		Name:     "2dsphere",
	})

	impl.accounts.EnsureIndex(mgo.Index{
		Key:      []string{"primary.phone"},
		Unique:   true,
		DropDups: true,
		Sparse:   true,
		Name:     "primary.phone",
	})

	impl.accounts.EnsureIndex(mgo.Index{
		Key:      []string{"primary.email"},
		Unique:   true,
		DropDups: true,
		Sparse:   true,
		Name:     "primary.email",
	})

	impl.accounts.EnsureIndex(mgo.Index{
		Key:      []string{"primary.username"},
		Unique:   true,
		DropDups: true,
		Sparse:   true,
		Name:     "primary.username",
	})

	// This is for configuration of services like callback/webhooks
	impl.webhooks = impl.db.C("webhooks")
	impl.webhooks.EnsureIndex(mgo.Index{
		Key:      []string{"service"},
		Unique:   true,
		DropDups: true,
		Sparse:   true,
		Name:     "webhooks_service",
	})

	if count, err := impl.webhooks.Count(); err == nil && count == 0 {
		for service, ekum := range DefaultWebHooks {
			err := impl.webhooks.Insert(&mgo_webhook{
				Id:      bson.NewObjectId(),
				Service: service,
				Map:     ekum,
			})
			if err != nil {
				panic(err)
			}
		}

	} else if err != nil {
		panic(err)
	}

	glog.Infoln("Passport MongoDb backend initialized:", impl)
	return impl, nil
}

func (this *serviceImpl) dropDatabase() (err error) {
	return this.db.DropDatabase()
}

func (this *serviceImpl) FindAccountByEmail(email string) (account *api.Account, err error) {
	result := api.Account{}
	err = this.accounts.Find(bson.M{"primary.email": email}).One(&result)
	switch {
	case err == mgo.ErrNotFound:
		return nil, ERROR_NOT_FOUND
	case err != nil:
		return nil, err
	}
	return &result, nil
}

func (this *serviceImpl) FindAccountByPhone(phone string) (account *api.Account, err error) {
	result := api.Account{}
	err = this.accounts.Find(bson.M{"primary.phone": phone}).One(&result)
	switch {
	case err == mgo.ErrNotFound:
		return nil, ERROR_NOT_FOUND
	case err != nil:
		return nil, err
	}
	return &result, nil
}

func (this *serviceImpl) FindAccountByUsername(username string) (account *api.Account, err error) {
	result := api.Account{}
	err = this.accounts.Find(bson.M{"primary.username": username}).One(&result)
	switch {
	case err == mgo.ErrNotFound:
		return nil, ERROR_NOT_FOUND
	case err != nil:
		return nil, err
	}
	return &result, nil
}

func (this *serviceImpl) SaveAccount(account *api.Account) (err error) {
	if account.GetId() == "" {
		uuid := omni_common.NewUUID().String()
		account.Id = &uuid
	}
	if account.GetPrimary().GetId() == "" {
		uuid2 := omni_common.NewUUID().String()
		account.GetPrimary().Id = &uuid2
	}
	// To work around the problem with Go's default "" value for string causing problem
	// with unique indexes (email, phone), we assign some uuid to fill up the fields so
	// that the fields that have unique indexes are never empty strings
	if account.GetPrimary().GetPhone() == "" {
		uuid3 := omni_common.NewUUID().String()
		account.GetPrimary().Phone = &uuid3
	}
	if account.GetPrimary().GetEmail() == "" {
		uuid4 := omni_common.NewUUID().String()
		account.GetPrimary().Email = &uuid4
	}

	changeInfo, err := this.accounts.Upsert(bson.M{"id": account.GetId()}, account)
	if changeInfo != nil && changeInfo.Updated >= 0 {
		return nil
	}
	return err
}

func (this *serviceImpl) GetAccount(id uuid.UUID) (account *api.Account, err error) {
	result := api.Account{}
	err = this.accounts.Find(bson.M{"id": id.String()}).One(&result)
	switch {
	case err == mgo.ErrNotFound:
		return nil, ERROR_NOT_FOUND
	case err != nil:
		return nil, err
	}
	return &result, nil
}

func (this *serviceImpl) DeleteAccount(id uuid.UUID) (err error) {
	err = this.accounts.Remove(bson.M{"id": id.String()})
	switch {
	case err == mgo.ErrNotFound:
		return nil
	case err != nil:
		return err
	}
	return nil
}

func (this *serviceImpl) Close() {
	this.session.Close()
	glog.Infoln("Session closed", this.session)
}

func (this *serviceImpl) Send(serviceKey, eventKey string, message interface{}, templateString string) error {

	result := &mgo_webhook{}

	err := this.webhooks.Find(bson.M{"service": serviceKey}).One(result)
	switch {
	case err == mgo.ErrNotFound:
		return ERROR_NOT_FOUND
	case err != nil:
		return err
	}

	if webhook, has := result.Map[eventKey]; has {
		webhook.Send(message, templateString)
	}

	return nil

}
