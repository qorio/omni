package rest

import (
	"code.google.com/p/goprotobuf/proto"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/qorio/omni/api"
	"github.com/qorio/omni/auth"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

var (
	ErrMissingInput       = errors.New("error-missing-input")
	ErrUnknownContentType = errors.New("error-no-content-type")
	ErrUnknownMethod      = errors.New("error-unknown-method")
	ErrIncompatibleType   = errors.New("error-incompatible-type")
)

var (
	json_marshaler = func(contentType string, resp http.ResponseWriter, typed interface{}) error {
		if buff, err := json.Marshal(typed); err == nil {
			resp.Header().Add("Content-Type", contentType)
			resp.Write(buff)
			return nil
		} else {
			return err
		}
	}

	json_unmarshaler = func(body io.ReadCloser, typed interface{}) error {
		dec := json.NewDecoder(body)
		return dec.Decode(typed)
	}

	proto_marshaler = func(contentType string, resp http.ResponseWriter, any interface{}) error {
		typed, ok := any.(proto.Message)
		if !ok {
			return ErrIncompatibleType
		}
		if buff, err := proto.Marshal(typed); err == nil {
			resp.Header().Add("Content-Type", contentType)
			resp.Write(buff)
			return nil
		} else {
			return err
		}
	}

	proto_unmarshaler = func(body io.ReadCloser, any interface{}) error {
		typed, ok := any.(proto.Message)
		if !ok {
			return ErrIncompatibleType
		}
		buff, err := ioutil.ReadAll(body)
		if err != nil {
			return err
		}
		return proto.Unmarshal(buff, typed)
	}

	marshalers = map[string]func(string, http.ResponseWriter, interface{}) error{
		"":                     json_marshaler,
		"application/json":     json_marshaler,
		"application/protobuf": proto_marshaler,
	}

	unmarshalers = map[string]func(io.ReadCloser, interface{}) error{
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
	Domain        string
	Service       string
	ServiceMethod api.ServiceMethod
	Body          interface{}
}

type Engine interface {
	Bind(...*ServiceMethodImpl)
	ServeHTTP(http.ResponseWriter, *http.Request)
	// NewAuthToken() *auth.Token
	// SignedStringForHttpRequest(*auth.Token, *http.Request) (string, error)
	GetUrlParameter(*http.Request, string) string
	Unmarshal(*http.Request, proto.Message) error
	Marshal(*http.Request, proto.Message, http.ResponseWriter) error
	UnmarshalJSON(*http.Request, interface{}) error
	MarshalJSON(*http.Request, interface{}, http.ResponseWriter) error
	HandleError(http.ResponseWriter, *http.Request, string, int) error
	EventChannel() chan<- *EngineEvent
}

type engine struct {
	spec       *api.ServiceMethods
	router     *mux.Router
	auth       auth.Service
	event_chan chan *EngineEvent
	done_chan  chan bool
	webhooks   WebhookManager
}

func NewEngine(spec *api.ServiceMethods, auth auth.Service, webhooks WebhookManager) *engine {
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

func (this *engine) Router() *mux.Router {
	return this.router
}

// func (this *engine) NewAuthToken() *auth.Token {
// 	return this.auth.NewToken()
// }

// func (this *engine) SignedStringForHttpRequest(token *auth.Token, req *http.Request) (string, error) {
// 	return this.auth.SignedStringForHttpRequest(token, req)
// }

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
			this.router.HandleFunc(ep.Api.UrlRoute, ep.Handler).Methods(string(ep.Api.HttpMethod))

		case ep.AuthenticatedHandler != nil:
			this.router.HandleFunc(ep.Api.UrlRoute,
				this.auth.RequiresAuth(ep.Api.AuthScope, func(token *auth.Token) []string {
					return strings.Split(token.GetString(ep.ServiceId+"/@scopes"), ",")
				}, ep.AuthenticatedHandler)).Methods(string(ep.Api.HttpMethod))

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

func content_type_from_request(req *http.Request) (t string) {
	if t = req.Header.Get("Accept"); t == "" {
		return req.Header.Get("Content-Type")
	}
	return
}

func (this *engine) Unmarshal(req *http.Request, typed proto.Message) (err error) {
	contentType := content_type_from_request(req)
	if unmarshaler, has := unmarshalers[contentType]; has {
		return unmarshaler(req.Body, typed)
	} else {
		return ErrUnknownContentType
	}
}

func (this *engine) Marshal(req *http.Request, typed proto.Message, resp http.ResponseWriter) (err error) {
	contentType := content_type_from_request(req)
	if marshaler, has := marshalers[contentType]; has {
		return marshaler(contentType, resp, typed)
	} else {
		return ErrUnknownContentType
	}
}

func (this *engine) UnmarshalJSON(req *http.Request, any interface{}) (err error) {
	contentType := content_type_from_request(req)
	if unmarshaler, has := unmarshalers[contentType]; has {
		return unmarshaler(req.Body, any)
	} else {
		return ErrUnknownContentType
	}
}

func (this *engine) MarshalJSON(req *http.Request, any interface{}, resp http.ResponseWriter) (err error) {
	if buff, err := json.Marshal(any); err == nil {
		resp.Header().Add("Content-Type", "application/json")
		resp.Write(buff)
		return nil
	} else {
		return err
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
	if m, has := (*this.spec)[message.ServiceMethod]; has {
		if m.CallbackEvent != api.EventKey("") {
			return this.webhooks.Send(message.Domain, message.Service, string(m.CallbackEvent), message.Body, m.CallbackBodyTemplate)
		}
	}
	return ErrUnknownMethod
}
