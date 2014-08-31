package passport

import (
	"code.google.com/p/goprotobuf/proto"
	"encoding/json"
	"fmt"
	"github.com/bmizerany/assert"
	"github.com/gorilla/mux"
	omni_auth "github.com/qorio/omni/auth"
	omni_rest "github.com/qorio/omni/rest"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"testing"
	"time"
)

func from_json(o interface{}, src io.Reader, t *testing.T) interface{} {
	err := json.NewDecoder(src).Decode(&o)
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func to_json(o interface{}, t *testing.T) []byte {
	data, err := json.Marshal(o)
	if err != nil {
		t.Error(err)
	}
	return data
}

func to_protobuf(o proto.Message, t *testing.T) []byte {
	data, err := proto.Marshal(o)
	if err != nil {
		t.Error(err)
	}
	return data
}

func from_protobuf(o proto.Message, buff []byte, t *testing.T) interface{} {
	if err := proto.Unmarshal(buff, o); err != nil {
		t.Log("Protobuf bytes", string(buff))
		t.Error(err)
	}
	return o
}

func check_error_response_reason(t *testing.T, body string, expected string) {
	authResponse := from_json(make(map[string]interface{}), strings.NewReader(body), t).(map[string]interface{})
	reason, has := authResponse["error"]
	assert.Equal(t, true, has)
	assert.Equal(t, expected, reason)
}

func default_settings() Settings {
	return Settings{
		Mongo: DbSettings{
			Hosts: []string{"localhost"},
			Db:    "passport_test",
		},
	}
}

func default_auth_settings(t *testing.T) omni_auth.Settings {
	key, err := omni_auth.ReadPublicKey(*AuthKeyFileFlag)
	if err != nil {
		t.Log("Cannot read public key file", *AuthKeyFileFlag)
		t.Fatal(err)
	}
	return omni_auth.Settings{
		SignKey:  key,
		TTLHours: time.Duration(1),
	}
}

func default_auth(t *testing.T) omni_auth.Service {
	return omni_auth.Init(default_auth_settings(t))
}

func default_endpoint(t *testing.T) *EndPoint {
	return endpoint(t, default_auth_settings(t), default_settings(), nil)
}

func default_service(t *testing.T) *serviceImpl {
	service, err := NewService(default_settings())
	if err != nil {
		t.Fatal(err)
	}
	return service
}

type serviceImplInit func(*testing.T, *serviceImpl)
type oauth2ImplInit func(*testing.T, *oauth2Impl)

func test_service(t *testing.T, authSettings omni_auth.Settings, s Settings, serviceInits ...serviceImplInit) *serviceImpl {
	service, err := NewService(s)

	if err != nil {
		t.Fatal(err)
	}

	for _, serviceInit := range serviceInits {
		if serviceInit != nil {
			serviceInit(t, service)
		}
	}
	return service
}

func test_oauth2(t *testing.T, s Settings, o2Inits ...oauth2ImplInit) *oauth2Impl {
	service, err := NewOAuth2Service(s)

	if err != nil {
		t.Fatal(err)
	}

	for _, serviceInit := range o2Inits {
		if serviceInit != nil {
			serviceInit(t, service)
		}
	}
	return service
}

func endpoint(t *testing.T, authSettings omni_auth.Settings, s Settings, serviceInits ...serviceImplInit) *EndPoint {

	auth := omni_auth.Init(authSettings)
	service, err := NewService(s)

	if err != nil {
		t.Fatal(err)
	}

	for _, serviceInit := range serviceInits {
		if serviceInit != nil {
			serviceInit(t, service)
		}
	}

	endpoint, err := NewApiEndPoint(default_settings(), auth, service, nil, service)
	if err != nil {
		t.Log("Error starting endpoint:", err)
		t.Fatal(err)
	}
	return endpoint
}

var test_port string

func get_port() string {
	for {
		port := fmt.Sprintf(":%d", rand.Int()%10000+10000)
		if port != test_port {
			test_port = port
			break
		}
	}
	return test_port
}

func start_passport(t *testing.T, handler http.Handler) struct{ URL string } {
	p := get_port()
	go func() {
		t.Log("Passport running at port", p)
		err := http.ListenAndServe(p, handler)
		t.Log("ERROR", err)
		if err != nil {
			t.Fatal(err)
		} else {
			select {}
		}
	}()
	return struct{ URL string }{"http://127.0.0.1" + p}
}

func start_server(t *testing.T, service, event, route, method string,
	handler func(resp http.ResponseWriter, req *http.Request) error) (wait func(int) error) {

	p := get_port()
	t.Log("Creating webhook listener for service", service, ",event=", event, "at", p)

	webhook := default_service(t)
	err := webhook.RegisterWebHooks(service, omni_rest.EventKeyUrlMap{
		event: omni_rest.WebHook{
			Url: "http://localhost" + p + route,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Start the server
	done := make(chan bool)
	r := mux.NewRouter()
	r.HandleFunc(route, func(resp http.ResponseWriter, req *http.Request) {
		defer func() {
			done <- true
		}()
		err = handler(resp, req)
	}).Methods(method)
	go func() {
		err := http.ListenAndServe(test_port, r)
		t.Log("ERROR", err)
	}()
	return func(seconds int) error {
		time.AfterFunc(time.Duration(seconds)*time.Second, func() { done <- true })
		<-done
		return err
	}
}
