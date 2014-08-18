package passport

import (
	"code.google.com/p/goprotobuf/proto"
	"encoding/json"
	"flag"
	"github.com/bmizerany/assert"
	"github.com/gorilla/mux"
	omni_auth "github.com/qorio/omni/auth"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

var (
	authKeyFile = flag.String("auth_public_key_file", "", "Auth public key file")
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
	// buff, err := ioutil.ReadAll(src)
	// if err != nil {
	// 	t.Error(err)
	// }

	if err := proto.Unmarshal(buff, o); err != nil {
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
	key, err := omni_auth.ReadPublicKey(*authKeyFile)
	if err != nil {
		t.Log("Cannot read public key file", *authKeyFile)
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

type serviceImplInit func(*testing.T, *serviceImpl)

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

	endpoint, err := NewApiEndPoint(default_settings(), auth, service, service)
	if err != nil {
		t.Log("Error starting endpoint:", err)
		t.Fatal(err)
	}
	return endpoint
}

func start_server(t *testing.T, addr, route, method string, handler func(resp http.ResponseWriter, req *http.Request) error) (wait func(int) error) {
	done := make(chan bool)
	r := mux.NewRouter()
	var err error
	r.HandleFunc(route, func(resp http.ResponseWriter, req *http.Request) {
		defer func() {
			done <- true
		}()
		err = handler(resp, req)
	}).Methods(method)
	go func() {
		http.ListenAndServe(addr, r)
	}()
	return func(seconds int) error {
		time.AfterFunc(time.Duration(seconds)*time.Second, func() { done <- true })
		<-done
		return err
	}
}
