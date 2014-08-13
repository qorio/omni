package passport

import (
	"code.google.com/p/go-uuid/uuid"
	"github.com/golang/glog"
	api "github.com/qorio/api/passport"
	omni_common "github.com/qorio/omni/common"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"strings"
)

type serviceImpl struct {
	settings   Settings
	db         *mgo.Database
	session    *mgo.Session
	collection *mgo.Collection
}

func NewService(settings Settings) (Service, error) {

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

	impl.db = impl.session.DB(settings.Mongo.Db)
	impl.collection = impl.db.C("accounts")
	// 2d spatial index on primary login's location
	impl.collection.EnsureIndex(mgo.Index{
		Key:      []string{"primary.location"},
		Unique:   false,
		DropDups: false,
		Name:     "2dsphere",
	})

	impl.collection.EnsureIndex(mgo.Index{
		Key:      []string{"primary.phone"},
		Unique:   true,
		DropDups: true,
		Name:     "primary.phone",
	})

	impl.collection.EnsureIndex(mgo.Index{
		Key:      []string{"primary.email"},
		Unique:   true,
		DropDups: true,
		Name:     "primary.email",
	})

	glog.Infoln("Passport MongoDb backend initialized:", impl)
	return impl, nil
}

func (this *serviceImpl) dropDatabase() (err error) {
	return this.db.DropDatabase()
}

func (this *serviceImpl) FindAccountByEmail(email string) (account *api.Account, err error) {
	result := api.Account{}
	err = this.collection.Find(bson.M{"primary.email": email}).One(&result)
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
	err = this.collection.Find(bson.M{"primary.phone": phone}).One(&result)
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

	changeInfo, err := this.collection.Upsert(bson.M{"id": account.GetId()}, account)
	if changeInfo != nil && changeInfo.Updated >= 0 {
		return nil
	}
	return err
}

func (this *serviceImpl) GetAccount(id uuid.UUID) (account *api.Account, err error) {
	result := api.Account{}
	err = this.collection.Find(bson.M{"id": id.String()}).One(&result)
	switch {
	case err == mgo.ErrNotFound:
		return nil, ERROR_NOT_FOUND
	case err != nil:
		return nil, err
	}
	return &result, nil
}

func (this *serviceImpl) DeleteAccount(id uuid.UUID) (err error) {
	err = this.collection.Remove(bson.M{"id": id.String()})
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
