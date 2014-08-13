package lighthouse

import (
	"code.google.com/p/go-uuid/uuid"
	"errors"
	api "github.com/qorio/api/lighthouse"
	passport "github.com/qorio/api/passport"
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
	Id            uuid.UUID            `json:"id"`
	Beacon        *api.Beacon          `json:"beacon"`
	ChangeHistory []*api.DeviceProfile `json:"change_history"`
	Acl           []*Acl               `json:"acl"`
}

type UserProfile struct {
	Id      uuid.UUID       `json:"id"`
	Login   *passport.Login `json:"login"`
	IsAdmin bool            `json:"admin_user"` // admin users are partners that distribute beacons to end users.
}

type Service interface {
	RegisterUser(*UserProfile) error
	GetUserProfile(uuid.UUID) (*UserProfile, error)
	SaveBeaconProfile(*BeaconProfile) error
	GetBeaconProfile(uuid.UUID) (*BeaconProfile, error)
	DeleteBeaconProfile(string) error
	FindBeaconProfileByUUIDMajorMinor([]byte, int, int) (*BeaconProfile, error)
	Close()
}

type BeaconPost struct {
	Id       string
	Beacons  []string
	Audience []UserUrl
	// post
	// short url
}
