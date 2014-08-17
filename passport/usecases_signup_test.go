package passport

import (
	"errors"
	"github.com/bmizerany/assert"
	"github.com/drewolson/testflight"
	api "github.com/qorio/api/passport"
	omni_rest "github.com/qorio/omni/rest"
	"net/http"
	"testing"
)

func TestNoUnaunthenticatedRegistrationCall(t *testing.T) {
	wait := start_server(t, ":9999", "/event/new-user-registration", "POST",
		func(resp http.ResponseWriter, req *http.Request) error {
			return errors.New("This should not be called because request isn't authenticated.")
		})

	testflight.WithServer(endpoint(t), func(r *testflight.Requester) {

		t.Log("Testing user registration without authentication token")

		assert.Equal(t, nil, nil)

		login := api.Methods[api.RegisterUser].RequestBody().(api.Login)

		response := r.Post("/api/v1/register/test", "application/protobuf", string(to_protobuf(&login, t)))
		t.Log("Got response", response.Body)
		assert.Equal(t, 401, response.StatusCode)
	})

	err := wait(1)
	assert.Equal(t, nil, err)
}

func TestRegisterUser(t *testing.T) {
	wait := start_server(t, ":9999", "/event/new-user-registration", "POST",
		func(resp http.ResponseWriter, req *http.Request) error {
			// check header
			if _, has := req.Header[omni_rest.WebHookHmacHeader]; !has {
				return errors.New("no hmac header")
			}
			// parse
			v := from_json(make(map[string]string), req.Body, t).(map[string]string)
			if _, has := v["id"]; !has {
				return errors.New("no id property")
			}
			return nil
		})

	testflight.WithServer(endpoint(t), func(r *testflight.Requester) {

		t.Log("Testing user registration without authentication token")

		assert.Equal(t, nil, nil)

		login := api.Methods[api.RegisterUser].RequestBody().(api.Login)

		response := r.Post("/api/v1/register/test", "application/protobuf", string(to_protobuf(&login, t)))
		t.Log("Got response", response)
		assert.Equal(t, 401, response.StatusCode)
	})

	err := wait(2)
	assert.Equal(t, nil, err)
}
