package lighthouse

import (
	"github.com/bmizerany/assert"
	api "github.com/qorio/api/lighthouse"
	common "github.com/qorio/omni/common"
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

func TestInsertGetAndDelete(t *testing.T) {
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

	/*
		err5 := service.DeleteBeaconProfile(b2.Id)
		assert.Equal(t, nil, err5)

		_, err6 := service.GetBeaconProfile(b2.Id)
		assert.Equal(t, ERROR_NOT_FOUND, err6)
	*/
}
