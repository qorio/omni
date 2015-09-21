package rest

import (
	"errors"
	"github.com/golang/protobuf/proto"
	"github.com/qorio/omni/api"
	"net/http"
)

var (
	ErrMissingInput                 = errors.New("error-missing-input")
	ErrUnknownContentType           = errors.New("error-no-content-type")
	ErrUnknownMethod                = errors.New("error-unknown-method")
	ErrIncompatibleType             = errors.New("error-incompatible-type")
	ErrNotSupportedUrlParameterType = errors.New("error-not-supported-url-query-param-type")
	ErrNoHttpHeaderSpec             = errors.New("error-no-http-header-spec")
)

type Handler func(http.ResponseWriter, *http.Request)

type EngineEvent struct {
	Domain        string
	Service       string
	ServiceMethod api.ServiceMethod
	Body          interface{}
}

type Engine interface {
	Bind(...*ServiceMethodImpl)
	Handle(string, http.Handler)
	ServeHTTP(http.ResponseWriter, *http.Request)
	GetUrlParameter(*http.Request, string) string
	GetHttpHeaders(*http.Request, api.HttpHeaders) (map[string][]string, error)
	GetUrlQueries(*http.Request, api.UrlQueries) (api.UrlQueries, error)
	Unmarshal(*http.Request, proto.Message) error
	Marshal(*http.Request, proto.Message, http.ResponseWriter) error
	UnmarshalJSON(*http.Request, interface{}) error
	MarshalJSON(*http.Request, interface{}, http.ResponseWriter) error
	HandleError(http.ResponseWriter, *http.Request, string, int) error
	EventChannel() chan<- *EngineEvent
	StreamChannel(contentType, eventType, key string) (*sseChannel, bool)
	BroadcastHttpStream(w http.ResponseWriter, r *http.Request, contentType, eventType, key string, src <-chan interface{}) error
	DirectHttpStream(http.ResponseWriter, *http.Request) (chan<- interface{}, error)
	Stop()
}
