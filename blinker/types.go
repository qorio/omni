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
	ERROR_UNKNOWN_IMAGE_FORMAT = errors.New("error-unknown-image-format")
)

type LprJob struct {
	Country   string      `json:"country"`
	Region    string      `json:"region"`
	Id        string      `json:"id"`
	Path      string      `json:"path"`
	RawResult interface{} `json:"raw_result"`
	HasImage  bool        `json:"has_image"`
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
	ListLprJobs() (id []*LprJob, err error)
	RunLprJob(country, region, id string, image io.ReadCloser) (stdout []byte, err error)
	Close()
}
