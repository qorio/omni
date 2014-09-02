package passport

import (
	"bytes"
	"code.google.com/p/goprotobuf/proto"
	"errors"
	"fmt"
	"github.com/bmizerany/assert"
	"github.com/drewolson/testflight"
	_ "github.com/golang/glog"
	api "github.com/qorio/api/passport"
	omni_common "github.com/qorio/omni/common"
	omni_rest "github.com/qorio/omni/rest"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

var (
	partner = struct {
		Email    string
		Password string
	}{
		"partner-test@passport",
		"partner-testpass",
	}
	apiUser = struct {
		Email    string
		Password string
	}{
		"api1@passport",
		"api1pass",
	}
)

var initialize_service_insert_passport_user_accounts = func(t *testing.T, impl *serviceImpl) {

	// This is the SDK api user -- it is able to register new user and read account information.
	user2 := &api.Account{
		Id: proto.String("api1"),
		Primary: &api.Identity{
			Email:    proto.String("api1@passport"),
			Password: proto.String("api1pass"),
		},
		Services: []*api.Service{
			&api.Service{
				Id:        proto.String("passport"),
				AccountId: proto.String("passport-user-2"),
				Scopes: []string{
					api.AuthScopes[api.RegisterNewUser],
					api.AuthScopes[api.AccountReadOnly],
				},
			},
		},
	}

	// This is the parnter system which delegates auth to passport. It receives notifications from
	// passport and can update passport's service accounts after its local account / use objects have
	// been created.
	user3 := &api.Account{
		Id: proto.String("partner-test"),
		Primary: &api.Identity{
			Email:    proto.String("partner-test@passport"),
			Password: proto.String("partner-testpass"),
		},
		Services: []*api.Service{
			&api.Service{
				Id:        proto.String("passport"),
				AccountId: proto.String("passport-user-3"),
				Scopes: []string{
					api.AuthScopes[api.AccountUpdate],
					api.AuthScopes[api.AccountReadOnly],
					api.AuthScopes[api.AccessProfile],
				},
			},
		},
	}

	Password(user2.Primary.Password).Hash().Update()
	Password(user3.Primary.Password).Hash().Update()

	err := impl.SaveAccount(user2)
	if err != nil {
		t.Fatal(err)
	}
	err = impl.SaveAccount(user3)
	if err != nil {
		t.Fatal(err)
	}
}

// Authenticates against the endpoint and returns the access token
func authenticate(t *testing.T, r *testflight.Requester, email, password, host *string) (token string) {
	authRequest := api.Methods[api.AuthUser].RequestBody().(api.Identity)
	authRequest.Email = email
	authRequest.Password = password
	token = authenticate2(t, r, &authRequest, host)
	return
}

func authenticate2(t *testing.T, r *testflight.Requester, authRequest *api.Identity, host *string) (token string) {
	url := "/api/v1/auth"
	authResponse := api.Methods[api.AuthUser].ResponseBody().(api.AuthResponse)
	if r != nil {
		// Sign in to all the services -- including passport
		response := r.Post(url, "application/protobuf", string(to_protobuf(authRequest, t)))
		assert.Equal(t, 200, response.StatusCode)
		from_protobuf(&authResponse, response.RawBody, t)
	} else {
		client := &http.Client{}
		url = *host + url
		post, err := http.NewRequest("POST", url, bytes.NewBuffer(to_protobuf(authRequest, t)))
		post.Header.Add("Content-Type", "application/protobuf")
		if err != nil {
			t.Fatal(err)
		}
		resp, err := client.Do(post)
		if err != nil {
			t.Fatal(err)
		}
		buff, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		from_protobuf(&authResponse, buff, t)
		assert.Equal(t, 200, resp.StatusCode)
	}
	t.Log("AUTH", "url=", url, "req=", authRequest.String(), "resp=", authResponse.String())
	return authResponse.GetToken()
}

func new_user() struct {
	Email    string
	Password string
} {
	return struct {
		Email    string
		Password string
	}{
		omni_common.NewUUID().String() + "-test@test.com",
		"testpass",
	}
}

// Tests minimal functionality where on new user registration, a callback is made to the other service.
// The other service will receive an actual user id.  This user id can then be used to look up the
// user account information in the passport system.
func TestRegisterUser(t *testing.T) {

	newUser := new_user()
	var newAccountId = proto.String("")
	wait := start_server(t, "test", "new-user-registration", "/event/new-user-registration", "POST",
		func(resp http.ResponseWriter, req *http.Request) error {
			t.Log("The webhook got called:", req.Body)
			// check header
			if _, has := req.Header[omni_rest.WebHookHmacHeader]; !has {
				return errors.New("no hmac header")
			}
			// parse
			v := from_json(make(map[string]interface{}), req.Body, t).(map[string]interface{})

			t.Log("Received message", v, "host=", req.Host)
			if id, has := v["id"]; !has {
				return errors.New("no id property")
			} else {
				*newAccountId = id.(string)
			}
			return nil
		})

	var registerResult *api.Identity = nil
	testflight.WithServer(endpoint(t, default_auth_settings(t), default_settings(),
		initialize_service_insert_passport_user_accounts),

		func(r *testflight.Requester) {

			t.Log("Authenticate user of service -- test")

			authToken := authenticate(t, r, &apiUser.Email, &apiUser.Password, nil)

			// Create the login object for signing up
			login := api.Methods[api.RegisterUser].RequestBody().(api.Identity)
			login.Email = &newUser.Email
			login.Password = &newUser.Password

			apiCall, err := http.NewRequest("POST", "/api/v1/register/test", bytes.NewBuffer(to_protobuf(&login, t)))
			assert.Equal(t, nil, err)
			apiCall.Header.Add("Content-Type", "application/protobuf")
			apiCall.Header.Add("Authorization", "Bearer "+authToken)
			apiResponse := r.Do(apiCall)

			assert.Equal(t, 200, apiResponse.StatusCode)

			o := api.Methods[api.RegisterUser].ResponseBody().(api.Identity)
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

	testflight.WithServer(endpoint(t, default_auth_settings(t), default_settings(),
		initialize_service_insert_passport_user_accounts),

		func(r *testflight.Requester) {

			authToken := authenticate(t, r, &apiUser.Email, &apiUser.Password, nil)

			apiCall, err := http.NewRequest("GET", "/api/v1/account/"+*newAccountId, nil)
			assert.Equal(t, nil, err)
			apiCall.Header.Add("Accept", "application/protobuf")
			apiCall.Header.Add("Authorization", "Bearer "+authToken)
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

/// Tests the case where the partner service receives a webhook call to indicate that a user has registered,
/// which then triggers the partner system to create a corresponding user account, if necessary.  The partner
/// system then updates passport with the application account which passport will maintain the association of
/// a global single signon user with the foregin account id as part of its service records.
func TestRegisterUserWithUpdateServiceAccountFromPartnerService(t *testing.T) {

	newUser := new_user()
	passport := start_passport(t, endpoint(t, default_auth_settings(t), default_settings(),
		initialize_service_insert_passport_user_accounts))
	//passport.Start()
	//defer passport.Close()

	t.Log("Started passport at", passport.URL)

	test_account_data := struct {
		NicknameKey          string
		NicknameValue        string
		ScopesForTestService []string
	}{
		"nickname",
		"The Hulk",
		[]string{"test_permission1", "test_permission2", "test_permission3"},
	}

	var newAccountId = proto.String("")
	wait := start_server(t, "test", "new-user-registration", "/event/new-user-registration", "POST",
		func(resp http.ResponseWriter, req *http.Request) error {
			t.Log("The webhook got called:", req.Body)
			// check header
			if _, has := req.Header[omni_rest.WebHookHmacHeader]; !has {
				return errors.New("no hmac header")
			}
			// parse
			v := from_json(make(map[string]interface{}), req.Body, t).(map[string]interface{})

			t.Log("Received message", v, "host=", req.Host, "remote=", req.RemoteAddr, "passport=", passport.URL)
			if id, has := v["id"]; !has {
				return errors.New("no id property")
			} else {
				*newAccountId = id.(string)
			}

			// Here we create any necessary account or user objects in our system.
			// After it's done, we make a call back to passport to update the service record.
			service := api.Methods[api.AddOrUpdateAccountService].RequestBody().(api.Service)
			service.Id = proto.String("test")
			service.Status = proto.String("created")
			service.AccountId = proto.String("test-service-account-" + *newAccountId)
			service.Scopes = test_account_data.ScopesForTestService
			// Add some interesting data that we want to have passport manage for us and include
			// in auth token.
			attr_type := api.Attribute_STRING
			embed := true
			service.Attributes = []*api.Attribute{
				&api.Attribute{
					Type:         &attr_type,
					EmbedInToken: &embed,
					Key:          &test_account_data.NicknameKey,
					StringValue:  &test_account_data.NicknameValue,
				},
			}

			// In production, this port number is well known, while during test random port is assigned.
			authToken := authenticate(t, nil, &partner.Email, &partner.Password, &passport.URL)
			t.Log("Got auth token:", authToken)
			if authToken == "" {
				t.Fatal(errors.New("no auth token"))
			}

			// Call passport to update the service record:
			url := fmt.Sprintf("%s/api/v1/account/%s/services", passport.URL, *newAccountId)
			client := &http.Client{}
			post, err := http.NewRequest("POST", url, bytes.NewBuffer(to_protobuf(&service, t)))
			post.Header.Add("Content-Type", "application/protobuf")
			post.Header.Add("Authorization", "Bearer "+authToken)
			postResponse, err := client.Do(post)
			t.Log("updated service account:", postResponse, err)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, 200, postResponse.StatusCode)
			return nil
		})

	authService := default_auth(t)

	var registerResult *api.Identity = nil
	t.Log("Authenticate api call to passport")
	authToken := authenticate(t, nil, &apiUser.Email, &apiUser.Password, &passport.URL)

	// Create the login object for signing up
	login := api.Methods[api.RegisterUser].RequestBody().(api.Identity)
	login.Email = &newUser.Email
	login.Password = &newUser.Password

	t.Log("Register new user for service 'test'")
	r := &http.Client{}
	apiCall, err := http.NewRequest("POST", passport.URL+"/api/v1/register/test", bytes.NewBuffer(to_protobuf(&login, t)))
	assert.Equal(t, nil, err)
	apiCall.Header.Add("Content-Type", "application/protobuf")
	apiCall.Header.Add("Authorization", "Bearer "+authToken)
	apiResponse, err := r.Do(apiCall)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 200, apiResponse.StatusCode)
	o := api.Methods[api.RegisterUser].ResponseBody().(api.Identity)
	buff, _ := ioutil.ReadAll(apiResponse.Body)
	from_protobuf(&o, buff, t)
	registerResult = &o
	t.Log("Got login", registerResult.String())
	assert.NotEqual(t, "", registerResult.GetId())
	assert.Equal(t, newUser.Email, registerResult.GetEmail())
	assert.Equal(t, login.GetEmail(), registerResult.GetEmail())
	assert.Equal(t, "", registerResult.GetPassword()) // Password is cleared
	assert.Equal(t, "", registerResult.GetPhone())

	t.Log("Wait for partner service 'test' to finish creating user in its system")
	err = wait(2)
	assert.Equal(t, nil, err)

	// Read user account data
	t.Log("Got new account id", *newAccountId)
	if *newAccountId == "" {
		t.Fatal("Did not get the new account id")
	}

	t.Log("Now get the account where id = ", *newAccountId)

	assert.Equal(t, *newAccountId, registerResult.GetId())
	authToken = authenticate(t, nil, &apiUser.Email, &apiUser.Password, &passport.URL)

	apiCall, err = http.NewRequest("GET", passport.URL+"/api/v1/account/"+*newAccountId, nil)
	assert.Equal(t, nil, err)
	apiCall.Header.Add("Accept", "application/protobuf")
	apiCall.Header.Add("Authorization", "Bearer "+authToken)
	apiResponse, err = r.Do(apiCall)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 200, apiResponse.StatusCode)

	a := api.Methods[api.FetchAccount].ResponseBody().(api.Account)
	buff, _ = ioutil.ReadAll(apiResponse.Body)
	from_protobuf(&a, buff, t)
	t.Log("Got account", a.String())
	assert.Equal(t, *newAccountId, a.GetId())
	assert.Equal(t, newUser.Email, a.Primary.GetEmail())
	assert.Equal(t, "", a.Primary.GetUsername())
	assert.Equal(t, "", a.Primary.GetPassword())

	// Check for service related data
	assert.Equal(t, 1, len(a.Services))

	service := a.Services[0]
	assert.Equal(t, "test", *service.Id)
	assert.Equal(t, test_account_data.ScopesForTestService, service.Scopes)
	assert.Equal(t, 1, len(service.Attributes))
	attribute := service.Attributes[0]
	assert.Equal(t, test_account_data.NicknameKey, attribute.GetKey())
	assert.Equal(t, test_account_data.NicknameValue, attribute.GetStringValue())

	// Now try to authenticate as the service user
	t.Log("Authenticate newUser to access test service via passport")
	authToken = authenticate(t, nil, &newUser.Email, &newUser.Password, &passport.URL)
	token, _ := authService.Parse(authToken)
	// Verify that the token contains the data the 'test' service would be interested in.
	// This token can be used with all calls with the test service.
	assert.Equal(t, "test-service-account-"+*newAccountId, token.GetString("test/@id"))
	assert.Equal(t, test_account_data.ScopesForTestService, strings.Split(token.GetString("test/@scopes"), ","))
	assert.Equal(t, test_account_data.NicknameValue, token.GetString("test/nickname"))

	// The Identity returned can be reused by the client
	registerResult.Password = &newUser.Password
	t.Log("Reusing the Identity object returned on registration", registerResult.String())
	authToken = authenticate2(t, nil, registerResult, &passport.URL)
	token, _ = authService.Parse(authToken)
	assert.NotEqual(t, "", token)

	// Verify that the token contains the data the 'test' service would be interested in.
	// This token can be used with all calls with the test service.
	assert.Equal(t, test_account_data.ScopesForTestService, strings.Split(token.GetString("test/@scopes"), ","))
	assert.Equal(t, test_account_data.NicknameValue, token.GetString("test/nickname"))

}
