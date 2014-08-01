package lighthouse

import (
	"errors"
)

// see passport.proto

var (
	ERROR_MISSING_INPUT        = errors.New("error-missing-input")
	ERROR_NOT_FOUND            = errors.New("account-not-found")
	ERROR_UNKNOWN_CONTENT_TYPE = errors.New("error-no-content-type")
)

type FsSettings struct {
	RootDir string
}

type DbSettings struct {
	Hosts []string
	Db    string
}

type Settings struct {
	FsSettings FsSettings
}

type Service interface {
	Close()
}
