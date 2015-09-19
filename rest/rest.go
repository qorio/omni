package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/mux"
	"github.com/qorio/omni/api"
	"github.com/qorio/omni/auth"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

type nht int

const (
	no_header nht = 1
)

var (
	string_marshaler = func(contentType string, resp http.ResponseWriter, typed interface{}, noHeader ...nht) error {
		if str, ok := typed.(*string); ok {
			if len(noHeader) == 0 {
				resp.Header().Add("Content-Type", contentType)
			}
			resp.Write([]byte(*str))
			return nil
		} else {
			return errors.New("wrong-type-expects-str-ptr")
		}
	}

	string_unmarshaler = func(body io.ReadCloser, typed interface{}) error {
		if _, ok := typed.(*string); !ok {
			return errors.New("wrong-type-expects-str-ptr")
		}
		if buff, err := ioutil.ReadAll(body); err == nil {
			ptr := typed.(*string)
			*ptr = string(buff)
			return nil
		} else {
			return err
		}
	}

	json_marshaler = func(contentType string, resp http.ResponseWriter, typed interface{}, noHeader ...nht) error {
		if buff, err := json.Marshal(typed); err == nil {
			if len(noHeader) == 0 {
				resp.Header().Add("Content-Type", contentType)
			}
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

	json_logging_unmarshaler = func(body io.ReadCloser, typed interface{}) error {
		buff, err := ioutil.ReadAll(body)
		if err != nil {
			return err
		}
		glog.V(100).Infoln("Unmarshal [", string(buff), "]")
		return json.Unmarshal(buff, typed)
	}

	proto_marshaler = func(contentType string, resp http.ResponseWriter, any interface{}, noHeader ...nht) error {
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

	marshalers = map[string]func(string, http.ResponseWriter, interface{}, ...nht) error{
		"":                     json_marshaler,
		"application/json":     json_marshaler,
		"application/protobuf": proto_marshaler,
		"text/plain":           string_marshaler,
		"text/html":            nil,
	}

	unmarshalers = map[string]func(io.ReadCloser, interface{}) error{
		"":                     json_logging_unmarshaler,
		"application/json":     json_logging_unmarshaler,
		"application/protobuf": proto_unmarshaler,
		"text/plain":           string_unmarshaler,
		"text/html":            nil,
	}
)

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

type engine struct {
	spec        *api.ServiceMethods
	router      *mux.Router
	auth        auth.Service
	event_chan  chan *EngineEvent
	done_chan   chan bool
	webhooks    WebhookManager
	sseChannels map[string]*sseChannel
	lock        sync.Mutex
}

func NewEngine(spec *api.ServiceMethods, auth auth.Service, webhooks WebhookManager) *engine {
	e := &engine{
		spec:        spec,
		router:      mux.NewRouter(),
		auth:        auth,
		event_chan:  make(chan *EngineEvent),
		done_chan:   make(chan bool),
		webhooks:    webhooks,
		sseChannels: make(map[string]*sseChannel),
	}
	return e
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

func (this *engine) GetHttpHeaders(req *http.Request, m api.HttpHeaders) (map[string][]string, error) {
	if m == nil {
		return nil, ErrNoHttpHeaderSpec
	}
	q := make(map[string][]string)
	for k, h := range m {
		if l, ok := req.Header[h]; ok {
			// Really strange -- you can have a 1 element list with value that's actually comma-delimited.
			if len(l) == 1 {
				q[k] = strings.Split(l[0], ", ")
			} else {
				q[k] = l
			}
		}
	}
	return q, nil
}

func (this *engine) GetUrlQueries(req *http.Request, m api.UrlQueries) (api.UrlQueries, error) {
	result := make(api.UrlQueries)
	for key, default_value := range m {
		actual := this.GetUrlParameter(req, key)
		if actual != "" {
			// Check the type and do conversion
			switch reflect.TypeOf(default_value).Kind() {
			case reflect.Bool:
				if v, err := strconv.ParseBool(actual); err != nil {
					return nil, err
				} else {
					result[key] = v
				}
			case reflect.String:
				result[key] = actual
			case reflect.Int:
				if v, err := strconv.Atoi(actual); err != nil {
					return nil, err
				} else {
					result[key] = v
				}
			case reflect.Float32:
				if v, err := strconv.ParseFloat(actual, 32); err != nil {
					return nil, err
				} else {
					result[key] = v
				}
			case reflect.Float64:
				if v, err := strconv.ParseFloat(actual, 64); err != nil {
					return nil, err
				} else {
					result[key] = v
				}
			default:
				return nil, ErrNotSupportedUrlParameterType
			}

		} else {
			result[key] = default_value
		}
	}
	return result, nil
}

func (this *engine) Handle(path string, handler http.Handler) {
	this.router.Handle(path, handler)
}

func (this *engine) Bind(endpoints ...*ServiceMethodImpl) {
	for i, ep := range endpoints {
		switch {
		case ep.Handler != nil:
			this.router.HandleFunc(ep.Api.UrlRoute, ep.Handler).Methods(string(ep.Api.HttpMethod))

		case ep.AuthenticatedHandler != nil:
			h := this.router.HandleFunc(ep.Api.UrlRoute,
				this.auth.RequiresAuth(ep.Api.AuthScope, func(token *auth.Token) []string {
					return strings.Split(token.GetString(ep.ServiceId+"/@scopes"), ",")
				}, ep.AuthenticatedHandler))
			if ep.Api.HttpMethod != "" {
				h.Methods(string(ep.Api.HttpMethod))
			}
			if len(ep.Api.HttpMethods) > 0 {
				s := []string{}
				for _, m := range ep.Api.HttpMethods {
					s = append(s, string(m))
				}
				h.Methods(s...)
			}

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

func JSONContentType(req *http.Request) bool {
	return "application/json" == content_type_for_request(req)
}

func GetContentType(req *http.Request) *string {
	if req == nil {
		return nil
	} else {
		t := content_type_for_request(req)
		return &t
	}
}

func content_type_for_request(req *http.Request) string {
	t := "application/json"

	if req.Method == "POST" || req.Method == "PUT" {
		t = req.Header.Get("Content-Type")
	}
	switch t {
	case "*/*":
		return "application/json"
	case "":
		return "application/json"
	default:
		return t
	}
}

func content_type_for_response(req *http.Request) string {
	t := req.Header.Get("Accept")
	switch t {
	case "*/*":
		return "application/json"
	case "":
		return content_type_for_request(req) // use the same content type as the request if no accept header
	default:
		return t
	}
}

var ErrorRenderer = func(resp http.ResponseWriter, req *http.Request, message string, code int) (err error) {
	// First look for accept content type in the header
	ct := content_type_for_response(req)
	switch ct {
	case "application/json":
		resp.Write([]byte(fmt.Sprintf("{ \"error\": \"%s\" }", message)))
		return
	case "application/protobuf":
		return
	default:
		resp.Write([]byte(fmt.Sprintf("<html><body>Error: %s </body></html>", message)))
		return
	}
}

func (this *engine) Unmarshal(req *http.Request, typed proto.Message) (err error) {
	contentType := content_type_for_request(req)
	if unmarshaler, has := unmarshalers[contentType]; has {
		return unmarshaler(req.Body, typed)
	} else {
		return ErrUnknownContentType
	}
}

func (this *engine) Marshal(req *http.Request, typed proto.Message, resp http.ResponseWriter) (err error) {
	contentType := content_type_for_response(req)
	if marshaler, has := marshalers[contentType]; has {
		return marshaler(contentType, resp, typed)
	} else {
		return ErrUnknownContentType
	}
}

func (this *engine) UnmarshalJSON(req *http.Request, any interface{}) (err error) {
	contentType := content_type_for_request(req)
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
	if len(message) > 0 {
		resp.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", message)))
	}
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
