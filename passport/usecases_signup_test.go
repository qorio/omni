package passport

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/bmizerany/assert"
	"github.com/drewolson/testflight"
	api "github.com/qorio/api/passport"
	omni_rest "github.com/qorio/omni/rest"
	"math/rand"
	"net/http"
	"testing"
	"time"
)

var initialize_service_insert_passport_user_accounts = func(t *testing.T, impl *serviceImpl) {
	user2 := &api.Account{
		Id: ptr("api1"),
		Primary: &api.Login{
			Email:    ptr("api1@passport"),
			Password: ptr("api1pass"),
		},
		Services: []*api.Service{
			&api.Service{
				Id:        ptr("passport"),
				AccountId: ptr("passport-user-2"),
				Scopes: []string{
					api.AuthScopes[api.AccountUpdate],
					api.AuthScopes[api.AccountReadOnly],
				},
			},
		},
	}
	user3 := &api.Account{
		Id: ptr("api2"),
		Primary: &api.Login{
			Email:    ptr("api2@passport"),
			Password: ptr("api2pass"),
		},
		Services: []*api.Service{
			&api.Service{
				Id:        ptr("passport"),
				AccountId: ptr("passport-user-3"),
				Scopes: []string{
					api.AuthScopes[api.AccountReadOnly],
				},
			},
		},
	}

	Password(user2.Primary.Password).Hash().Update()
	Password(user3.Primary.Password).Hash().Update()

	err := impl.SaveAccount(user2)
	err = impl.SaveAccount(user3)

	if err != nil {
		t.Fatal(err)
	}
}

func TestRegisterUser(t *testing.T) {

	// first create a new callback /webhook record
	rand.Seed(time.Now().Unix())
	port := fmt.Sprintf(":%d", rand.Int()%10000+10000)

	t.Log("Creating webhook listener at", port)

	webhook := default_service(t)
	err := webhook.RegisterWebHooks("test", omni_rest.EventKeyUrlMap{
		"new-user-registration": omni_rest.WebHook{
			Url: "http://localhost" + port + "/event/new-user-registration",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	var newAccountId = ptr("")
	wait := start_server(t, port, "/event/new-user-registration", "POST",
		func(resp http.ResponseWriter, req *http.Request) error {
			t.Log("The webhook got called:", req.Body)
			// check header
			if _, has := req.Header[omni_rest.WebHookHmacHeader]; !has {
				return errors.New("no hmac header")
			}
			// parse
			v := from_json(make(map[string]interface{}), req.Body, t).(map[string]interface{})

			t.Log("Received message", v)
			if id, has := v["id"]; !has {
				return errors.New("no id property")
			} else {
				*newAccountId = id.(string)
			}
			return nil
		})

	authSettings := default_auth_settings(t)
	authSettings.CheckScope = func(methodScope string, grantedScopes []string) bool {
		return true
	}

	newUser := struct {
		Email    string
		Password string
	}{
		"test@test.com",
		"testpass",
	}

	var registerResult *api.Login = nil

	testflight.WithServer(endpoint(t, authSettings, default_settings(),
		initialize_service_insert_passport_user_accounts),

		func(r *testflight.Requester) {

			t.Log("Authenticate user of service -- test")

			assert.Equal(t, nil, nil)

			authRequest := api.Methods[api.AuthUser].RequestBody().(api.AuthRequest)
			authRequest.Email = ptr("api1@passport")
			authRequest.Password = ptr("api1pass")

			// Sign in to all the services -- including passport
			response := r.Post("/api/v1/auth", "application/protobuf", string(to_protobuf(&authRequest, t)))
			assert.Equal(t, 200, response.StatusCode)

			authResponse := api.Methods[api.AuthUser].ResponseBody().(api.AuthResponse)
			from_protobuf(&authResponse, response.RawBody, t)
			authService := default_auth(t)
			token, _ := authService.Parse(authResponse.GetToken())
			t.Log("Scopes:", token.GetString("@passport/scopes"))

			// Create the login object for signing up
			login := api.Methods[api.RegisterUser].RequestBody().(api.Login)
			login.Email = &newUser.Email
			login.Password = &newUser.Password

			apiCall, err := http.NewRequest("POST", "/api/v1/register/test", bytes.NewBuffer(to_protobuf(&login, t)))
			assert.Equal(t, nil, err)
			apiCall.Header.Add("Content-Type", "application/protobuf")
			apiCall.Header.Add("Authorization", "Bearer "+authResponse.GetToken())
			apiResponse := r.Do(apiCall)

			assert.Equal(t, 200, apiResponse.StatusCode)

			o := api.Methods[api.RegisterUser].ResponseBody().(api.Login)
			from_protobuf(&o, apiResponse.RawBody, t)
			registerResult = &o
			t.Log("Got login", registerResult.String())
			assert.Equal(t, newUser.Email, registerResult.GetEmail())
			assert.Equal(t, "", registerResult.GetPassword())
			assert.Equal(t, "", registerResult.GetPhone())

			err = wait(2)
			assert.Equal(t, nil, err)
		})

	t.Log("Got new account id", *newAccountId)
	if *newAccountId == "" {
		t.Fatal("Did not get the new account id")
	}

	t.Log("Now get the account where id = ", *newAccountId)

	assert.Equal(t, *newAccountId, registerResult.GetId())

	testflight.WithServer(endpoint(t, authSettings, default_settings(),
		initialize_service_insert_passport_user_accounts),

		func(r *testflight.Requester) {

			t.Log("Authenticate user of service -- test")

			assert.Equal(t, nil, nil)

			authRequest := api.Methods[api.AuthUser].RequestBody().(api.AuthRequest)
			authRequest.Email = ptr("api1@passport")
			authRequest.Password = ptr("api1pass")

			// Sign in to all the services -- including passport
			response := r.Post("/api/v1/auth", "application/protobuf", string(to_protobuf(&authRequest, t)))
			authResponse := api.Methods[api.AuthUser].ResponseBody().(api.AuthResponse)
			from_protobuf(&authResponse, response.RawBody, t)

			apiCall, err := http.NewRequest("GET", "/api/v1/account/"+*newAccountId, nil)
			assert.Equal(t, nil, err)
			apiCall.Header.Add("Accept", "application/protobuf")
			apiCall.Header.Add("Authorization", "Bearer "+authResponse.GetToken())
			apiResponse := r.Do(apiCall)

			assert.Equal(t, 200, apiResponse.StatusCode)

			a := api.Methods[api.FetchAccount].ResponseBody().(api.Account)
			from_protobuf(&a, apiResponse.RawBody, t)
			t.Log("Got account", a)
			assert.Equal(t, *newAccountId, a.GetId())
			assert.Equal(t, newUser.Email, a.Primary.GetEmail())
			assert.Equal(t, "", a.Primary.GetUsername())
			assert.Equal(t, "", a.Primary.GetPassword())

		})

}
