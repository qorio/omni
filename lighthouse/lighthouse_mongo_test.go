package lighthouse

import (
	"github.com/bmizerany/assert"
	api "github.com/qorio/api/lighthouse"
	"github.com/qorio/api/passport"
	"github.com/qorio/omni/common"
	"testing"
	"time"
)

func default_settings() Settings {
	return Settings{
		Mongo: DbSettings{
			Hosts: []string{"localhost"},
			Db:    "lighthouse_test",
		},
	}
}

var beacon = struct {
	DeviceId         string
	InstallTimestamp float64
	Lat              float64
	Lng              float64
	Uuid             []byte
	Major            int32
	Minor            int32
}{
	"device-id-1",
	float64(time.Now().Unix()),
	45.,
	17.,
	[]byte{},
	1000,
	2000,
}

func test_beacon() *BeaconProfile {
	beacon.Uuid = common.NewUUID()
	return &BeaconProfile{
		Beacon: &api.Beacon{
			HardwareId: &beacon.DeviceId,
			AdvertiseInfo: &api.BeaconAdvertisement{
				Ibeacon: &api.IBeacon{
					Uuid:  beacon.Uuid,
					Major: &beacon.Major,
					Minor: &beacon.Minor,
				},
			},
			InstalledTimestamp: &beacon.InstallTimestamp,
			Location: &api.Location{
				Lon: &beacon.Lng,
				Lat: &beacon.Lat,
			},
		},
	}
}

func TestNewService(t *testing.T) {
	service, err := NewService(default_settings())
	assert.Equal(t, nil, err)

	defer service.Close()
	t.Log("Started db client", service)
}

func TestRegisterUserAndGetUser(t *testing.T) {

	// Actual workflow of registering a user:
	// 1. Mobile client posts Login to passport
	// 2. Passport create new account if necessary
	// 3. On success, a callback URL is posted.  This calls lighthouse
	// 4. Lighthouse creates new UserProfile object
	// 5. Lighthouse updates passport with service information, e.g. adds a service

	// Consider adding new endpoint to passport like /v1/{service}/register where {service}
	// will map to lighthouse, and a specific callback url (webhook) on successful registration

	service, err := NewService(default_settings())
	assert.Equal(t, nil, err)

	defer service.Close()
	t.Log("Started db client", service)

	up, err := service.RegisterUser(&passport.Identity{
		Email:    ptr("foo@bar.com"),
		Password: ptr("password"),
		Username: ptr("foo"),
	})
	assert.Equal(t, nil, err)
	assert.NotEqual(t, nil, up)
	assert.NotEqual(t, nil, up.Id)

	up2, err := service.GetUserProfile(up.Id)
	assert.Equal(t, nil, err)
	assert.Equal(t, up.Login.String(), up2.Login.String())
	assert.Equal(t, up.toJSON(), up2.toJSON())
}

func TestInsertGetAndDeleteBeaconProfile(t *testing.T) {
	service, err := NewService(default_settings())
	assert.Equal(t, nil, err)

	impl := service.(*serviceImpl)
	impl.dropDatabase()

	defer service.Close()
	t.Log("Started db client", service)

	b := test_beacon()
	err = service.SaveBeaconProfile(b)
	assert.Equal(t, nil, err)

	t.Log("id=", b.Id)

	b2, err2 := service.GetBeaconProfile(b.Id)
	assert.Equal(t, nil, err2)
	t.Log("b2", b2)

	err5 := service.DeleteBeaconProfile(b2.Id)
	assert.Equal(t, nil, err5)

	_, err6 := service.GetBeaconProfile(b2.Id)
	assert.Equal(t, ERROR_NOT_FOUND, err6)
}
