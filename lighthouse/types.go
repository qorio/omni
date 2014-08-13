package lighthouse

import (
	"errors"
	api "github.com/qorio/api/lighthouse"
)

var (
	ERROR_NOT_FOUND = errors.New("beacon-not-found")
)

type DbSettings struct {
	Hosts []string
	Db    string
}

type Settings struct {
	Mongo DbSettings
}

type UserUrl string

type Acl struct {
	Id    string    `json:"id"`
	Name  string    `json:"name"`
	Users []UserUrl `json:"users"`
}

type BeaconProfile struct {
	Id            string               `json:"id"`
	Beacon        *api.Beacon          `json:"beacon"`
	ChangeHistory []*api.DeviceProfile `json:"change_history"`
	Acl           []*Acl               `json:"acl"`
}

type Service interface {
	SaveBeacon(*BeaconProfile) error
	GetBeacon(string) (*BeaconProfile, error)
	DeleteBeacon(string) error
	FindBeaconByUUIDMajorMinor([]byte, int, int) (*BeaconProfile, error)
	Close()
}

type BeaconPost struct {
	Id       string
	Beacons  []string
	Audience []UserUrl
	// post
	// short url
}
