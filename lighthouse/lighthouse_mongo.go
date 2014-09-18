package lighthouse

import (
	"code.google.com/p/go-uuid/uuid"
	"github.com/golang/glog"
	"github.com/qorio/api/passport"
	"github.com/qorio/omni/common"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"strings"
)

type serviceImpl struct {
	settings           Settings
	db                 *mgo.Database
	session            *mgo.Session
	beacons_collection *mgo.Collection
	users_collection   *mgo.Collection
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
	impl.beacons_collection = impl.db.C("beacons")
	// 2d spatial index on beacon's location
	impl.beacons_collection.EnsureIndex(mgo.Index{
		Key:      []string{"beacon.location"},
		Unique:   false,
		DropDups: false,
		Sparse:   true,
		Name:     "2dsphere",
	})

	impl.beacons_collection.EnsureIndex(mgo.Index{
		Key:      []string{"beacon.advertise_info.uuid, beacon.advertise_info.major, beacon.advertise_info.minor"},
		Unique:   true,
		DropDups: true,
		Sparse:   true,
		Name:     "ibeacon_advertise_info",
	})

	impl.users_collection = impl.db.C("users")

	glog.Infoln("Lighthouse MongoDb backend initialized:", impl)
	return impl, nil
}

func (this *serviceImpl) dropDatabase() (err error) {
	return this.db.DropDatabase()
}

func (this *serviceImpl) RegisterUser(l *passport.Identity) (u *UserProfile, err error) {
	return this.registerUser(l, false)
}

func (this *serviceImpl) RegisterAdminUser(l *passport.Identity) (u *UserProfile, err error) {
	return this.registerUser(l, true)
}

func (this *serviceImpl) registerUser(l *passport.Identity, admin bool) (u *UserProfile, err error) {
	userProfile := &UserProfile{
		Id:      common.NewUUID(),
		Login:   l,
		IsAdmin: admin,
	}

	changeInfo, err := this.users_collection.Upsert(bson.M{"id": userProfile.Id}, userProfile)
	if changeInfo != nil && changeInfo.Updated >= 0 {
		return userProfile, err
	}
	return nil, err
}

func (this *serviceImpl) GetUserProfile(id uuid.UUID) (*UserProfile, error) {
	result := UserProfile{}
	err := this.users_collection.Find(bson.M{"id": id}).One(&result)
	switch {
	case err == mgo.ErrNotFound:
		return nil, ERROR_NOT_FOUND
	case err != nil:
		return nil, err
	}
	return &result, nil
}

func (this *serviceImpl) SaveBeaconProfile(beacon *BeaconProfile) (err error) {
	if beacon.Id == nil {
		beacon.Id = common.NewUUID()
	}
	changeInfo, err := this.beacons_collection.Upsert(bson.M{"id": beacon.Id}, beacon)
	glog.Infoln("upsert beacon", beacon, changeInfo, err)
	if changeInfo != nil && changeInfo.Updated >= 0 {
		return nil
	}
	return err
}

func (this *serviceImpl) GetBeaconProfile(id uuid.UUID) (beacon *BeaconProfile, err error) {
	result := BeaconProfile{}
	err = this.beacons_collection.Find(bson.M{"id": id}).One(&result)
	switch {
	case err == mgo.ErrNotFound:
		return nil, ERROR_NOT_FOUND
	case err != nil:
		return nil, err
	}
	return &result, nil
}

func (this *serviceImpl) DeleteBeaconProfile(id uuid.UUID) (err error) {
	err = this.beacons_collection.Remove(bson.M{"id": id})
	switch {
	case err == mgo.ErrNotFound:
		return nil
	case err != nil:
		return err
	}
	return nil
}

func (this *serviceImpl) FindBeaconProfileByUUIDMajorMinor(uuid []byte, major, minor int) (beacon *BeaconProfile, err error) {
	result := BeaconProfile{}
	err = this.beacons_collection.Find(bson.M{"uuid": uuid, "major": major, "minor": minor}).One(&result)
	switch {
	case err == mgo.ErrNotFound:
		return nil, ERROR_NOT_FOUND
	case err != nil:
		return nil, err
	}
	return &result, nil
}

func (this *serviceImpl) Close() {
	this.session.Close()
	glog.Infoln("Session closed", this.session)
}
