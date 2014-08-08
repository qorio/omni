package http

import (
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/qorio/omni/auth"
	"net/http"
)

type ObjectFactory func() interface{}
type Handler func(http.ResponseWriter, *http.Request)

type ServiceMethod struct {
	Doc                  string
	Name                 string
	UrlRoute             string
	HttpMethod           string
	ContentTypes         []string
	RequestBody          ObjectFactory
	ResponseBody         ObjectFactory
	Handler              Handler
	AuthenticatedHandler auth.HttpHandler
}

func Publish(endpoints ...*ServiceMethod) []*ServiceMethod {
	return endpoints
}

type Engine interface {
	Bind(endpoints []*ServiceMethod)
	ServeHTTP(resp http.ResponseWriter, request *http.Request)
	NewAuthToken() *auth.Token
	SignedString(*auth.Token) (string, error)
}

type engine struct {
	router *mux.Router
	auth   *auth.Service
}

func NewEngine(auth *auth.Service) Engine {
	return &engine{
		router: mux.NewRouter(),
		auth:   auth,
	}
}

func (this *engine) NewAuthToken() *auth.Token {
	return this.auth.NewToken()
}

func (this *engine) SignedString(token *auth.Token) (string, error) {
	return this.auth.SignedString(token)
}

func (this *engine) ServeHTTP(resp http.ResponseWriter, request *http.Request) {
	this.router.ServeHTTP(resp, request)
}

func (this *engine) Bind(endpoints []*ServiceMethod) {
	for i, ep := range endpoints {
		switch {
		case ep.Handler != nil:
			this.router.HandleFunc(ep.UrlRoute, ep.Handler).Methods(ep.HttpMethod).Name(ep.Name)

		case ep.AuthenticatedHandler != nil:
			this.router.HandleFunc(ep.UrlRoute, this.auth.RequiresAuth(ep.AuthenticatedHandler)).Methods(ep.HttpMethod).Name(ep.Name)

		case ep.Handler == nil && ep.AuthenticatedHandler == nil:
			panic(errors.New(fmt.Sprintf("No implementation for REST endpoint[%d]: %s", i, ep)))
		}
	}
}
