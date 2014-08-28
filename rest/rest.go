package rest

import (
	"code.google.com/p/goprotobuf/proto"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/qorio/api"
	"github.com/qorio/omni/auth"
	omni_http "github.com/qorio/omni/http"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

var (
	ERROR_MISSING_INPUT        = errors.New("error-missing-input")
	ERROR_UNKNOWN_CONTENT_TYPE = errors.New("error-no-content-type")
	ERROR_UNKNOWN_METHOD       = errors.New("error-unknown-method")
)

var (
	json_marshaler = func(contentType string, resp http.ResponseWriter, typed proto.Message) error {
		if buff, err := json.Marshal(typed); err == nil {
			omni_http.SetCORSHeaders(resp)
			resp.Header().Add("Content-Type", contentType)
			resp.Write(buff)
			return nil
		} else {
			return err
		}
	}

	json_unmarshaler = func(body io.ReadCloser, typed proto.Message) error {
		dec := json.NewDecoder(body)
		return dec.Decode(typed)
	}

	proto_marshaler = func(contentType string, resp http.ResponseWriter, typed proto.Message) error {
		if buff, err := proto.Marshal(typed); err == nil {
			omni_http.SetCORSHeaders(resp)
			resp.Header().Add("Content-Type", contentType)
			resp.Write(buff)
			return nil
		} else {
			return err
		}
	}

	proto_unmarshaler = func(body io.ReadCloser, typed proto.Message) error {
		buff, err := ioutil.ReadAll(body)
		if err != nil {
			return err
		}
		return proto.Unmarshal(buff, typed)
	}

	marshalers = map[string]func(string, http.ResponseWriter, proto.Message) error{
		"":                     json_marshaler,
		"application/json":     json_marshaler,
		"application/protobuf": proto_marshaler,
	}

	unmarshalers = map[string]func(io.ReadCloser, proto.Message) error{
		"":                     json_unmarshaler,
		"application/json":     json_unmarshaler,
		"application/protobuf": proto_unmarshaler,
	}
)

type Handler func(http.ResponseWriter, *http.Request)

type ServiceMethodImpl struct {
	Api                  api.MethodSpec // note this is by copy -- so that behavior is deterministic after initialization
	Handler              Handler
	AuthenticatedHandler auth.HttpHandler
	ServiceId            string
}

func SetHandler(m api.MethodSpec, h Handler) *ServiceMethodImpl {
	if m.AuthScope != "" {
		panic(errors.New(fmt.Sprintf("Method %s has oauth scopes but binding to unauthed handler.", m)))
	}
	return &ServiceMethodImpl{
		Api:     m,
		Handler: h,
	}
}

func SetAuthenticatedHandler(serviceId string, m api.MethodSpec, h auth.HttpHandler) *ServiceMethodImpl {
	if m.AuthScope == "" {
		panic(errors.New(fmt.Sprintf("Method %s has no oauth scopes but binding to authenticated handler.", m)))
	}
	return &ServiceMethodImpl{
		Api:                  m,
		AuthenticatedHandler: h,
		ServiceId:            serviceId,
	}
}

type EngineEvent struct {
	Service       string
	ServiceMethod api.ServiceMethod
	Body          interface{}
}

type Engine interface {
	Bind(...*ServiceMethodImpl)
	ServeHTTP(http.ResponseWriter, *http.Request)
	NewAuthToken() *auth.Token
	SignedString(*auth.Token) (string, error)
	GetUrlParameter(*http.Request, string) string
	Unmarshal(*http.Request, proto.Message) error
	Marshal(*http.Request, proto.Message, http.ResponseWriter) error
	HandleError(http.ResponseWriter, *http.Request, string, int) error
	EventChannel() chan<- *EngineEvent
}

type engine struct {
	spec       *api.ServiceMethods
	router     *mux.Router
	auth       auth.Service
	event_chan chan *EngineEvent
	done_chan  chan bool
	webhooks   WebHooksService
}

func NewEngine(spec *api.ServiceMethods, auth auth.Service, webhooks WebHooksService) Engine {
	e := &engine{
		spec:       spec,
		router:     mux.NewRouter(),
		auth:       auth,
		event_chan: make(chan *EngineEvent),
		done_chan:  make(chan bool),
		webhooks:   webhooks,
	}
	return e
}

func (this *engine) NewAuthToken() *auth.Token {
	return this.auth.NewToken()
}

func (this *engine) SignedString(token *auth.Token) (string, error) {
	return this.auth.SignedString(token)
}

func (this *engine) ServeHTTP(resp http.ResponseWriter, request *http.Request) {
	// Also start listening on the event channel for any webhook calls
	go func() {
		for {
			select {

			case message := <-this.event_chan:
				this.do_callback(message)

			case done := <-this.done_chan:
				if done {
					glog.Infoln("REST engine event channel stopped.")
					return
				}
			}
		}
	}()

	this.router.ServeHTTP(resp, request)
	glog.Infoln("Stopping event channel")
	this.done_chan <- true
}

func (this *engine) GetUrlParameter(req *http.Request, key string) string {
	vars := mux.Vars(req)
	if val, has := vars[key]; has {
		return val
	} else if err := req.ParseForm(); err == nil {
		if _, has := req.Form[key]; has {
			return req.Form[key][0]
		}
	}
	return ""
}

func (this *engine) Bind(endpoints ...*ServiceMethodImpl) {
	for i, ep := range endpoints {
		switch {
		case ep.Handler != nil:
			this.router.HandleFunc(ep.Api.UrlRoute, ep.Handler).Methods(ep.Api.HttpMethod)

		case ep.AuthenticatedHandler != nil:
			this.router.HandleFunc(ep.Api.UrlRoute,
				this.auth.RequiresAuth(ep.Api.AuthScope, func(token *auth.Token) []string {
					return strings.Split(token.GetString(ep.ServiceId+"/@scopes"), ",")
				}, ep.AuthenticatedHandler)).Methods(ep.Api.HttpMethod)

		case ep.Handler == nil && ep.AuthenticatedHandler == nil:
			panic(errors.New(fmt.Sprintf("No implementation for REST endpoint[%d]: %s", i, ep)))
		}

		// check the content type
		for _, ct := range ep.Api.ContentTypes {
			if _, has := marshalers[ct]; !has {
				panic(errors.New(fmt.Sprintf("Bad content type: %s", ct)))
			}
			if _, has := unmarshalers[ct]; !has {
				panic(errors.New(fmt.Sprintf("Bad content type: %s", ct)))
			}
		}
	}
}

func (this *engine) Unmarshal(req *http.Request, typed proto.Message) (err error) {
	contentType := req.Header.Get("Content-Type")
	if unmarshaler, has := unmarshalers[contentType]; has {
		return unmarshaler(req.Body, typed)
	} else {
		return ERROR_UNKNOWN_CONTENT_TYPE
	}
}

func (this *engine) Marshal(req *http.Request, typed proto.Message, resp http.ResponseWriter) (err error) {
	contentType := req.Header.Get("Content-Type") // Usually the case when a client POSTs
	if contentType == "" {
		contentType = req.Header.Get("Accept") // Usually the case with GET
	}
	if marshaler, has := marshalers[contentType]; has {
		return marshaler(contentType, resp, typed)
	} else {
		return ERROR_UNKNOWN_CONTENT_TYPE
	}
}

func (this *engine) HandleError(resp http.ResponseWriter, req *http.Request, message string, code int) (err error) {
	resp.WriteHeader(code)
	resp.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", message)))
	return
}

func (this *engine) EventChannel() chan<- *EngineEvent {
	return this.event_chan
}

func (this *engine) do_callback(message *EngineEvent) error {
	if this.webhooks == nil {
		return nil
	}
	//methods := *this.spec
	if m, has := (*this.spec)[message.ServiceMethod]; has {
		if m.CallbackEvent != api.EventKey("") {
			return this.webhooks.Send(message.Service, string(m.CallbackEvent), message.Body, m.CallbackBodyTemplate)
		}
	}
	return ERROR_UNKNOWN_METHOD
}
