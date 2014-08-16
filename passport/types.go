package passport

import (
	"code.google.com/p/go-uuid/uuid"
	"errors"
	api "github.com/qorio/api/passport"
	"net/http"
)

// see passport.proto

var (
	ERROR_MISSING_INPUT        = errors.New("error-missing-input")
	ERROR_NOT_FOUND            = errors.New("not-found")
	ERROR_UNKNOWN_CONTENT_TYPE = errors.New("error-no-content-type")
)

type DbSettings struct {
	Hosts []string
	Db    string
}

type Settings struct {

	// Settings for mongo db
	Mongo DbSettings

	// Function that takes the http request and determine the application id
	// The default is to take the request's URL host, e.g. qor.io or shorty.qor.io
	ResolveServiceId func(req *http.Request) string
}

type Service interface {
	FindAccountByEmail(email string) (account *api.Account, err error)
	FindAccountByPhone(phone string) (account *api.Account, err error)
	FindAccountByUsername(username string) (account *api.Account, err error)
	SaveAccount(account *api.Account) (err error)
	GetAccount(id uuid.UUID) (account *api.Account, err error)
	DeleteAccount(id uuid.UUID) (err error)
	Close()
}
