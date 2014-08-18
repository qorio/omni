package passport

import (
	"errors"
	"github.com/bmizerany/assert"
	"github.com/drewolson/testflight"
	api "github.com/qorio/api/passport"
	// omni_common "github.com/qorio/omni/common"
	//omni_rest "github.com/qorio/omni/rest"
	"net/http"
	"testing"
)

var initialize_service_insert_root_account = func(t *testing.T, impl *serviceImpl) {
	rootAccount := &api.Account{
		Id: ptr("root"),
		Primary: &api.Login{
			Email:    ptr("root@passport"),
			Password: ptr("rootpass"),
		},
		Services: []*api.Service{
			&api.Service{
				Id:        ptr("test"),
				AccountId: ptr("test-root"),
				Scopes: []string{
					api.AuthScopes[api.ManageAccount],
				},
			},
		},
	}

	Password(rootAccount.Primary.Password).Hash().Update()

	err := impl.SaveAccount(rootAccount)
	if err != nil {
		t.Fatal(err)
	}
}

var initialize_service_log = func(t *testing.T, impl *serviceImpl) {
	t.Log("Initialized service")
}

func TestNoUnaunthenticatedRegistrationCall(t *testing.T) {
	wait := start_server(t, ":9999", "/event/new-user-registration", "POST",
		func(resp http.ResponseWriter, req *http.Request) error {
			return errors.New("This should not be called because request isn't authenticated.")
		})

	testflight.WithServer(default_endpoint(t), func(r *testflight.Requester) {

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

func TestAuthenticateUser(t *testing.T) {
	authSettings := default_auth_settings(t)
	authSettings.CheckScope = func(methodScope string, grantedScopes []string) bool {
		return true
	}

	testflight.WithServer(endpoint(t, authSettings, default_settings(),
		initialize_service_insert_root_account,
		initialize_service_log),
		func(r *testflight.Requester) {

			t.Log("Authenticate root user")

			assert.Equal(t, nil, nil)

			authRequest := api.Methods[api.AuthUser].RequestBody().(api.AuthRequest)
			authRequest.Email = ptr("root@passport")
			authRequest.Password = ptr("rootpass")

			response := r.Post("/api/v1/auth/test", "application/protobuf", string(to_protobuf(&authRequest, t)))
			t.Log("Got response", response)
			assert.Equal(t, 200, response.StatusCode)
		})
}
