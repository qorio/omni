package http

import (
	"bytes"
	"code.google.com/p/goprotobuf/proto"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/qorio/api"
	"github.com/qorio/omni/auth"
	"io"
	"io/ioutil"
	"net/http"
	"text/template"
)

var (
	ERROR_MISSING_INPUT        = errors.New("error-missing-input")
	ERROR_UNKNOWN_CONTENT_TYPE = errors.New("error-no-content-type")
)

var marshalers = map[string]func(string, http.ResponseWriter, proto.Message) error{
	"application/json": func(contentType string, resp http.ResponseWriter, typed proto.Message) error {
		if buff, err := json.Marshal(typed); err == nil {
			SetCORSHeaders(resp)
			resp.Header().Add("Content-Type", contentType)
			resp.Write(buff)
			return nil
		} else {
			return err
		}
	},
	"": func(contentType string, resp http.ResponseWriter, typed proto.Message) error {
		if buff, err := json.Marshal(typed); err == nil {
			SetCORSHeaders(resp)
			resp.Header().Add("Content-Type", contentType)
			resp.Write(buff)
			return nil
		} else {
			return err
		}
	},
	"application/protobuf": func(contentType string, resp http.ResponseWriter, typed proto.Message) error {
		if buff, err := proto.Marshal(typed); err == nil {
			SetCORSHeaders(resp)
			resp.Header().Add("Content-Type", contentType)
			resp.Write(buff)
			return nil
		} else {
			return err
		}
	},
}

var unmarshalers = map[string]func(io.ReadCloser, proto.Message) error{
	"application/json": func(body io.ReadCloser, typed proto.Message) error {
		dec := json.NewDecoder(body)
		return dec.Decode(typed)
	},
	"": func(body io.ReadCloser, typed proto.Message) error {
		dec := json.NewDecoder(body)
		return dec.Decode(typed)
	},
	"application/protobuf": func(body io.ReadCloser, typed proto.Message) error {
		buff, err := ioutil.ReadAll(body)
		if err != nil {
			return err
		}
		return proto.Unmarshal(buff, typed)
	},
}

type Handler func(http.ResponseWriter, *http.Request)

type ServiceMethodImpl struct {
	Api                  api.MethodSpec // note this is by copy -- so that behavior is deterministic after initialization
	Handler              Handler
	AuthenticatedHandler auth.HttpHandler
}

func SetHandler(m api.MethodSpec, h Handler) *ServiceMethodImpl {
	if m.RequiresAuth {
		panic(errors.New("Method " + m.Name + " requires auth; binding to unauthed handler."))
	}
	return &ServiceMethodImpl{
		Api:     m,
		Handler: h,
	}
}

func SetAuthenticatedHandler(m api.MethodSpec, h auth.HttpHandler) *ServiceMethodImpl {
	if !m.RequiresAuth {
		panic(errors.New("Method " + m.Name + " requires no auth; binding to authed handler."))
	}
	return &ServiceMethodImpl{
		Api:                  m,
		AuthenticatedHandler: h,
	}
}

type EngineEvent struct {
	ServiceMethod api.ServiceMethod
	Body          interface{}
}

type Engine interface {
	Bind(...*ServiceMethodImpl)
	ServeHTTP(http.ResponseWriter, *http.Request)
	NewAuthToken() *auth.Token
	SignedString(*auth.Token) (string, error)
	ServiceMethod(string) *ServiceMethodImpl
	GetUrlParameter(*http.Request, string) string
	Unmarshal(*http.Request, proto.Message) error
	Marshal(*http.Request, proto.Message, http.ResponseWriter) error
	HandleError(http.ResponseWriter, *http.Request, string, int) error
	EventChannel() chan<- *EngineEvent
}

type engine struct {
	spec       *api.ServiceMethods
	router     *mux.Router
	auth       *auth.Service
	methods    map[string]*ServiceMethodImpl
	event_chan chan *EngineEvent
	done_chan  chan bool
}

func NewEngine(spec *api.ServiceMethods, auth *auth.Service) Engine {
	return &engine{
		spec:       spec,
		router:     mux.NewRouter(),
		auth:       auth,
		methods:    make(map[string]*ServiceMethodImpl),
		event_chan: make(chan *EngineEvent),
		done_chan:  make(chan bool),
	}
}

func (this *engine) NewAuthToken() *auth.Token {
	return this.auth.NewToken()
}

func (this *engine) SignedString(token *auth.Token) (string, error) {
	return this.auth.SignedString(token)
}

func (this *engine) do_callback(message *EngineEvent) {
	methods := *this.spec
	if m, has := methods[message.ServiceMethod]; has {
		if m.CallbackEvent != api.EventKey("") {
			go func() {
				glog.Infoln("Sending", message, "spec", m)

				// TODO - cache the compiled templates
				var buffer bytes.Buffer
				if m.CallbackBodyTemplate != "" {
					t := template.Must(template.New(m.Name).Parse(m.CallbackBodyTemplate))
					err := t.Execute(&buffer, message.Body)
					if err != nil {
						glog.Warningln("Cannot build payload for event", message)
						return
					}
				}
				// Determine where to send the event.
				url := ""
				client := &http.Client{}
				post, err := http.NewRequest("POST", url, &buffer)
				post.Header.Add("x-passport-hmac", "TO DO: compute a HMAC here")
				resp, err := client.Do(post)
				if err != nil {
					glog.Warningln("Cannot deliver callback to", url)
				} else {
					glog.Infoln("Sent callback to ", url, "response=", resp)
				}
			}()
		}
	}
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

func (this *engine) ServiceMethod(key string) *ServiceMethodImpl {
	if v, has := this.methods[key]; has {
		return v
	} else {
		panic(errors.New(fmt.Sprintf("Mismatched key: %s", key)))
	}
}

func (this *engine) GetUrlParameter(req *http.Request, key string) string {
	vars := mux.Vars(req)
	if val, has := vars[key]; has {
		return val
	} else if err := req.ParseForm(); err == nil {
		return req.Form[key][0]
	}
	return ""
}

func (this *engine) Bind(endpoints ...*ServiceMethodImpl) {
	for i, ep := range endpoints {
		switch {
		case ep.Handler != nil:
			this.router.HandleFunc(ep.Api.UrlRoute, ep.Handler).Methods(ep.Api.HttpMethod).Name(ep.Api.Name)
			this.methods[ep.Api.Name] = ep

		case ep.AuthenticatedHandler != nil:
			this.router.HandleFunc(ep.Api.UrlRoute, this.auth.RequiresAuth(ep.AuthenticatedHandler)).Methods(ep.Api.HttpMethod).Name(ep.Api.Name)
			this.methods[ep.Api.Name] = ep

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
	contentType := req.Header.Get("Content-Type")
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
