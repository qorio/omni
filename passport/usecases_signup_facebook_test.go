package passport

import (
	"bytes"
	"code.google.com/p/goprotobuf/proto"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bmizerany/assert"
	_ "github.com/golang/glog"
	api "github.com/qorio/api/passport"
	omni_common "github.com/qorio/omni/common"
	"io/ioutil"
	"net/http"
	"testing"
)

// This sets up the configuration for the register facebook app that gets the user token/ profile.
var initialize_oauth2_service_insert_app_configs = func(t *testing.T, impl *oauth2Impl) {
	// This is an actual registered app with facebook, QLTest.
	// See https://developers.facebook.com/apps/769962796379311/settings/
	appConfig1 := &OAuth2AppConfig{
		Status:     OAuth2AppStatusLive,
		Provider:   "facebook.com",
		AppId:      "769962796379311",
		AppSecret:  "7fb04adfc64c6c154eca2395f0191540",
		ServiceIds: []string{"test"},
	}
	err := impl.SaveAppConfig(appConfig1)
	if err != nil {
		t.Fatal(err)
	}
}

func start_full_passport_server(t *testing.T) string {
	// Start up the passport server.
	passport_service := test_service(t, default_settings(),
		initialize_service_insert_passport_user_accounts)
	oauth2_service := test_oauth2(t, default_settings(),
		initialize_oauth2_service_insert_app_configs)
	passport := start_passport(t, test_endpoint(t, default_auth_settings(t), passport_service, oauth2_service))
	t.Log("Started passport at", passport.URL)
	return passport.URL
}

func get_passport_account_id_from_webhook_call(t *testing.T, req *http.Request) (string, error) {
	v := from_json(make(map[string]interface{}), req.Body, t).(map[string]interface{})
	if id, has := v["id"]; !has {
		return "", errors.New("no id property")
	} else {
		return id.(string), nil
	}
}

// Typedef the facebook object
type facebook_me map[string]interface{}

func partner_fetch_user_facebook_profile(t *testing.T, passport_url, passport_account_id string) (facebook_me, error) {
	// Need to authenticate as a partner system
	authToken := authenticate(t, nil, &partner.Email, &partner.Password, &passport_url)
	if authToken == "" {
		t.Fatal(errors.New("no auth token"))
	}
	client := &http.Client{}
	url := passport_url + "/api/v1/account/" + passport_account_id + "/profile/test/facebook.com"
	get, err := http.NewRequest("GET", url, nil)
	get.Header.Add("Authorization", "Bearer "+authToken)
	resp, err := client.Do(get)
	if err != nil {
		return nil, err
	}

	buff, err := ioutil.ReadAll(resp.Body)
	assert.Equal(t, 200, resp.StatusCode)

	me := make(facebook_me)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(buff, &me)
	if err != nil {
		return nil, err
	}
	return me, nil
}

func partner_create_account(t *testing.T, passport_account_id string, me facebook_me) (account_id string, err error) {
	// Here we create a local account
	account_id = omni_common.NewUUID().String()
	return account_id, nil
}

func partner_create_passport_service_record(t *testing.T, local_account_id string, me facebook_me) *api.Service {
	service := api.Methods[api.AddOrUpdateAccountService].RequestBody().(api.Service)
	service.Id = proto.String("test")
	service.Status = proto.String("created")
	service.AccountId = proto.String(local_account_id)
	service.Scopes = []string{"test_permission1", "test_permission2"}

	// These attributes will be embedded in the auth token whenever the user signs in via passport.
	service.Attributes = []*api.Attribute{
		&api.Attribute{
			Type:         api.Attribute_STRING.Enum(),
			EmbedInToken: proto.Bool(true),
			Key:          proto.String("gender"),
			StringValue:  proto.String(me["gender"].(string)),
		},
	}
	return &service
}

func post_to_passport(t *testing.T, passport_url, url, api_user, api_secret *string, payload proto.Message) error {
	authToken := authenticate(t, nil, api_user, api_secret, passport_url)
	if authToken == "" {
		t.Fatal(errors.New("no auth token"))
	}
	client := &http.Client{}
	post, err := http.NewRequest("POST", *passport_url+*url, bytes.NewBuffer(to_protobuf(payload, t)))
	post.Header.Add("Content-Type", "application/protobuf")
	post.Header.Add("Authorization", "Bearer "+authToken)
	postResponse, err := client.Do(post)
	if err != nil {
		t.Fatal(err)
		return err
	}
	assert.Equal(t, 200, postResponse.StatusCode)
	return err
}

/// Tests the complete flow of a user signing up with facebook SSO
/// This here shows how a user's facebook profile gets to a partner system via passport and
/// the user registering via oauth's implicit flow where the client (ios app) gets the access token.
func TestRegisterUserWithFacebookSignin(t *testing.T) {

	passport_url := start_full_passport_server(t)

	wait := start_server(t, "test", "new-user-registration", "/event/new-user-registration", "POST",

		func(resp http.ResponseWriter, req *http.Request) error {

			t.Log("Received webhook")

			// 1. Get the passport account id.  This can then be used for lookup of passport account, profile, etc.
			passport_account_id, err := get_passport_account_id_from_webhook_call(t, req)
			if err != nil {
				t.Fatal(err)
				return err
			}

			// 2. Call passport to get the user's facebook profile, if available.  This is OPTIONAL
			me, err := partner_fetch_user_facebook_profile(t, passport_url, passport_account_id)
			if err != nil {
				t.Fatal(err)
				return err
			}
			t.Log("Got profile from facebook", me)

			// 3. Create the local account.
			local_account_id, err := partner_create_account(t, passport_account_id, me)
			if err != nil {
				t.Fatal(err)
				return err
			}

			// 4. Call back to Passport so that Passport knows what permissions to give the user when she signs
			// in to use the service in the future.
			service_record := partner_create_passport_service_record(t, local_account_id, me)

			// 5. Post to passport
			url := fmt.Sprintf("/api/v1/account/%s/services", passport_account_id)
			err = post_to_passport(t, &passport_url, &url, &partner.Email, &partner.Password, service_record)
			if err != nil {
				t.Fatal(err)
				return err
			}
			return nil
		})

	// In the client app, an Identiy object is created when the user signs up.
	// The user in this case has gone through the Facebook signin flow and have granted the app access.
	// As a result, we have a user access token.
	new_signup := api.Methods[api.RegisterUser].RequestBody().(api.Identity)
	new_signup.Oauth2Provider = proto.String("facebook.com")
	new_signup.Oauth2AccessToken = proto.String("CAAK8Ru737K8BAOCuulle6JdrZBB7T3qgZBLl5iNpZB0miSZBygHGwFnQsvBLe3QUG7bLMxBRrsZAn1ZBay5WLNuHDrusL4R3Tpdt3agTmIgKlCxeOCxZBGQlUa8Td4b3m11NMmZBVwDW47Ry357LNS4ZCNVHeLtkOTcGcM0bsO5yICn2ZCQP254ne6ZBnUQBeNKsa6Urrb1Q0xA52IB1wlPK592jM1RAgHEODUZD")

	t.Log("Authenticate api call to passport")
	authToken := authenticate(t, nil, &apiUser.Email, &apiUser.Password, &passport_url)
	r := &http.Client{}
	apiCall, err := http.NewRequest("POST", passport_url+"/api/v1/register/test",
		bytes.NewBuffer(to_protobuf(&new_signup, t)))
	apiCall.Header.Add("Content-Type", "application/protobuf")
	apiCall.Header.Add("Authorization", "Bearer "+authToken)
	apiResponse, err := r.Do(apiCall)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 200, apiResponse.StatusCode)

	registeredIdentity := api.Methods[api.RegisterUser].ResponseBody().(api.Identity)
	buff, _ := ioutil.ReadAll(apiResponse.Body)
	from_protobuf(&registeredIdentity, buff, t)
	t.Log("Got login", registeredIdentity.String())
	assert.NotEqual(t, "", registeredIdentity.GetId())

	t.Log("Response from registration will have the access token unset.")
	assert.Equal(t, "", registeredIdentity.GetOauth2AccessToken())
	t.Log("Response will also have the passport account id and oauth information")
	assert.NotEqual(t, "", registeredIdentity.GetId())
	assert.Equal(t, "facebook.com", registeredIdentity.GetOauth2Provider())
	assert.NotEqual(t, "", registeredIdentity.GetOauth2AccountId())

	t.Log("Check for errors in the partner system")
	err = wait(2)
	assert.Equal(t, nil, err)

	// The Identity returned can be reused by the client
	t.Log("Reusing the Identity object returned on registration for future authentication")
	// Must set the access token
	registeredIdentity.Oauth2AccessToken = proto.String(new_signup.GetOauth2AccessToken())
	user_token := authenticate2(t, nil, &registeredIdentity, &passport_url)

	t.Log("Check returned token for permissions and data")
	token, _ := default_auth(t).Parse(user_token)
	assert.NotEqual(t, "", token)
	assert.Equal(t, "male", token.GetString("test/gender"))

}
