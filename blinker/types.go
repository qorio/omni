package blinker

import (
	"errors"
	"io"
)

// see passport.proto

var (
	ERROR_MISSING_INPUT        = errors.New("error-missing-input")
	ERROR_NOT_FOUND            = errors.New("account-not-found")
	ERROR_UNKNOWN_CONTENT_TYPE = errors.New("error-no-content-type")
)

type AlprCommand struct {
	Country string
	Region  string
	Path    string
}

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
	GetImage(country, region, id string) (bytes io.ReadCloser, size int64, err error)
	ExecAlpr(country, region, id string, image io.ReadCloser) (stdout []byte, err error)
	Close()
}
