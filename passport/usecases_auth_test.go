package passport

import (
	"errors"
	"github.com/bmizerany/assert"
	"github.com/drewolson/testflight"
	api "github.com/qorio/api/passport"
	"net/http"
	"testing"
)

var initialize_service_insert_test_user_account = func(t *testing.T, impl *serviceImpl) {
	rootAccount := &api.Account{
		Id: ptr("user1"),
		Primary: &api.Login{
			Email:    ptr("user1@passport"),
			Password: ptr("user1pass"),
		},
		Services: []*api.Service{
			&api.Service{
				Id:        ptr("test"),
				AccountId: ptr("test-root"),
				Scopes: []string{
					api.AuthScopes[api.AccountReadOnly],
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
	wait := start_server(t, "test", "new-user-registration", "/event/new-user-registration", "POST",
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
		initialize_service_insert_test_user_account,
		initialize_service_log),
		func(r *testflight.Requester) {

			t.Log("Authenticate user of service -- test")

			assert.Equal(t, nil, nil)

			authRequest := api.Methods[api.AuthUser].RequestBody().(api.AuthRequest)
			authRequest.Email = ptr("user1@passport")
			authRequest.Password = ptr("user1pass")

			response := r.Post("/api/v1/auth/test", "application/protobuf", string(to_protobuf(&authRequest, t)))
			assert.Equal(t, 200, response.StatusCode)

			authResponse := api.Methods[api.AuthUser].ResponseBody().(api.AuthResponse)
			from_protobuf(&authResponse, response.RawBody, t)
			t.Log("authResponse:", authResponse, "token:", authResponse.GetToken())
			assert.NotEqual(t, "", authResponse.GetToken())

			authService := default_auth(t)
			token, _ := authService.Parse(authResponse.GetToken())

			t.Log("Scopes:", token.GetString("test/@scopes"))
			assert.Equal(t, api.AuthScopes[api.AccountReadOnly], token.GetString("test/@scopes"))
		})
}
