package lighthouse

import (
	"github.com/golang/glog"
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
	impl.collection = impl.db.C("beacons")
	// 2d spatial index on beacon's location
	impl.collection.EnsureIndex(mgo.Index{
		Key:      []string{"beacon.location"},
		Unique:   false,
		DropDups: false,
		Name:     "2dsphere",
	})

	impl.collection.EnsureIndex(mgo.Index{
		Key:      []string{"beacon.advertise_info.uuid, beacon.advertise_info.major, beacon.advertise_info.minor"},
		Unique:   true,
		DropDups: true,
		Name:     "ibeacon_advertise_info",
	})

	glog.Infoln("Lighthouse MongoDb backend initialized:", impl)
	return impl, nil
}

func (this *serviceImpl) dropDatabase() (err error) {
	return this.db.DropDatabase()
}

func (this *serviceImpl) SaveBeaconProfile(beacon *BeaconProfile) (err error) {
	uuid, _ := omni_common.NewUUID()
	if beacon.Id == "" {
		beacon.Id = uuid
	}

	changeInfo, err := this.collection.Upsert(bson.M{"id": beacon.Id}, beacon)
	if changeInfo != nil && changeInfo.Updated >= 0 {
		return nil
	}
	return err
}

func (this *serviceImpl) GetBeaconProfile(id string) (beacon *BeaconProfile, err error) {
	result := BeaconProfile{}
	err = this.collection.Find(bson.M{"id": id}).One(&result)
	switch {
	case err == mgo.ErrNotFound:
		return nil, ERROR_NOT_FOUND
	case err != nil:
		return nil, err
	}
	return &result, nil
}

func (this *serviceImpl) DeleteBeaconProfile(id string) (err error) {
	err = this.collection.Remove(bson.M{"id": id})
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
	err = this.collection.Find(bson.M{"uuid": uuid, "major": major, "minor": minor}).One(&result)
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
