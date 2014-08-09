package http

import (
	"code.google.com/p/goprotobuf/proto"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/qorio/omni/auth"
	"io/ioutil"
	"net/http"
)

var (
	ERROR_MISSING_INPUT        = errors.New("error-missing-input")
	ERROR_UNKNOWN_CONTENT_TYPE = errors.New("error-no-content-type")
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
	Bind([]*ServiceMethod)
	ServeHTTP(http.ResponseWriter, *http.Request)
	NewAuthToken() *auth.Token
	SignedString(*auth.Token) (string, error)
	ServiceMethod(string) *ServiceMethod
	Unmarshal(*http.Request, proto.Message) error
	Marshal(*http.Request, proto.Message, http.ResponseWriter) error
	RenderJsonError(http.ResponseWriter, *http.Request, string, int) error
}

type engine struct {
	router  *mux.Router
	auth    *auth.Service
	methods map[string]*ServiceMethod
}

func NewEngine(auth *auth.Service) Engine {
	return &engine{
		router:  mux.NewRouter(),
		auth:    auth,
		methods: make(map[string]*ServiceMethod),
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

func (this *engine) ServiceMethod(key string) *ServiceMethod {
	glog.Infoln("KEY = ", key)

	if v, has := this.methods[key]; has {
		return v
	} else {
		panic(errors.New(fmt.Sprintf("Mismatched key: %s", key)))
	}
}

func (this *engine) Bind(endpoints []*ServiceMethod) {
	for i, ep := range endpoints {
		switch {
		case ep.Handler != nil:
			this.router.HandleFunc(ep.UrlRoute, ep.Handler).Methods(ep.HttpMethod).Name(ep.Name)
			this.methods[ep.Name] = ep

		case ep.AuthenticatedHandler != nil:
			this.router.HandleFunc(ep.UrlRoute, this.auth.RequiresAuth(ep.AuthenticatedHandler)).Methods(ep.HttpMethod).Name(ep.Name)
			this.methods[ep.Name] = ep

		case ep.Handler == nil && ep.AuthenticatedHandler == nil:
			panic(errors.New(fmt.Sprintf("No implementation for REST endpoint[%d]: %s", i, ep)))
		}
	}
}

func (this *engine) Unmarshal(req *http.Request, typed proto.Message) (err error) {
	contentType := req.Header.Get("Content-Type")
	body := req.Body
	switch {
	case contentType == "application/json" || contentType == "":
		dec := json.NewDecoder(body)
		return dec.Decode(typed)
	case contentType == "application/protobuf":
		buff, err := ioutil.ReadAll(body)
		if err != nil {
			return err
		}
		return proto.Unmarshal(buff, typed)
	default:
		return ERROR_UNKNOWN_CONTENT_TYPE
	}
}

func (this *engine) Marshal(req *http.Request, typed proto.Message, resp http.ResponseWriter) (err error) {
	contentType := req.Header.Get("Content-Type")
	var buff []byte
	switch {
	case contentType == "application/json" || contentType == "":
		if buff, err = json.Marshal(typed); err == nil {
		}
	case contentType == "application/protobuf":
		buff, err = proto.Marshal(typed)
	default:
		err = ERROR_UNKNOWN_CONTENT_TYPE
	}

	if err == nil {
		SetCORSHeaders(resp)
		resp.Header().Add("Content-Type", contentType)
		resp.Write(buff)
	}
	return
}

func (this *engine) RenderJsonError(resp http.ResponseWriter, req *http.Request, message string, code int) (err error) {
	resp.WriteHeader(code)
	resp.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", message)))
	return
}
