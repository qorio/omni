package passport

import (
	"code.google.com/p/goprotobuf/proto"
	"encoding/json"
	"flag"
	"github.com/bmizerany/assert"
	"github.com/drewolson/testflight"
	omni_auth "github.com/qorio/omni/auth"
	"net/http"
	"strings"
	"testing"
)

var (
	authKeyFile = flag.String("auth_public_key_file", "", "Auth public key file")
)

type mock struct {
	findByEmail   func(email string) (account *Account, err error)
	findByPhone   func(phone string) (account *Account, err error)
	saveAccount   func(account *Account) (err error)
	getAccount    func(id string) (account *Account, err error)
	deleteAccount func(id string) (err error)
}

func (this *mock) FindAccountByEmail(email string) (account *Account, err error) {
	return this.findByEmail(email)
}

func (this *mock) FindAccountByPhone(email string) (account *Account, err error) {
	return this.findByPhone(email)
}

func (this *mock) SaveAccount(account *Account) (err error) {
	return this.saveAccount(account)
}

func (this *mock) GetAccount(id string) (account *Account, err error) {
	return this.getAccount(id)
}

func (this *mock) DeleteAccount(id string) (err error) {
	return this.deleteAccount(id)
}
func ptr(s string) *string {
	return &s
}

func check_error_response_reason(t *testing.T, body string, expected string) {
	dec := json.NewDecoder(strings.NewReader(body))
	authResponse := make(map[string]string)

	if err := dec.Decode(&authResponse); err != nil {
		t.Error(err)
	}

	reason, has := authResponse["error"]
	assert.Equal(t, true, has)
	assert.Equal(t, expected, reason)
}

func TestAuthNotFound(t *testing.T) {

	signKey := []byte("test")
	settings := Settings{}

	auth := omni_auth.Init(omni_auth.Settings{SignKey: signKey, TTLHours: 0})
	service := &mock{
		findByEmail: func(email string) (account *Account, err error) {
			return nil, ERROR_ACCOUNT_NOT_FOUND
		},
		findByPhone: func(phone string) (account *Account, err error) {
			t.Error("testing look up by email; this shouldn't be called")
			return nil, nil
		},
	}

	endpoint, err := NewApiEndPoint(settings, auth, service)

	if err != nil {
		t.Error(err)
	}

	authRequest := &AuthRequest{
		Email:    ptr("foo@bar.com"),
		Password: ptr("test"),
	}

	data, err := json.Marshal(authRequest)
	if err != nil {
		t.Error(err)
	}

	// Account does not exist
	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Post("/api/v1/auth", "applicaton/json", string(data))

		t.Log("Got response", response.Body)
		assert.Equal(t, 401, response.StatusCode)
		check_error_response_reason(t, response.Body, "error-account-not-found")
	})

	// Password does not match
	service.findByEmail = func(email string) (account *Account, err error) {
		password := "not-a-match"
		return &Account{Primary: &Login{Password: &password}}, nil
	}
	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Post("/api/v1/auth", "applicaton/json", string(data))

		t.Log("Got response", response.Body)
		assert.Equal(t, 401, response.StatusCode)
		check_error_response_reason(t, response.Body, "error-bad-credentials")
	})

	// Find by phone
	authRequest.Email = ptr("")
	authRequest.Phone = ptr("123-111-2222")
	data, err = json.Marshal(authRequest)
	if err != nil {
		t.Error(err)
	}
	service.findByEmail = func(email string) (account *Account, err error) {
		t.Error("should not call this function")
		return nil, nil
	}
	service.findByPhone = func(email string) (account *Account, err error) {
		password := "no-match"
		return &Account{Primary: &Login{Password: &password}}, nil
	}
	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Post("/api/v1/auth", "applicaton/json", string(data))

		t.Log("Got response", response.Body)
		assert.Equal(t, 401, response.StatusCode)
		check_error_response_reason(t, response.Body, "error-bad-credentials")
	})

	// Bad request: no email or phone
	authRequest.Email = nil
	authRequest.Phone = nil
	data, err = json.Marshal(authRequest)
	if err != nil {
		t.Error(err)
	}
	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Post("/api/v1/auth", "applicaton/json", string(data))

		t.Log("Got response", response.Body)
		assert.Equal(t, 400, response.StatusCode)
		check_error_response_reason(t, response.Body, "error-no-phone-or-email")
	})
}

func (authRequest *AuthRequest) to_json(t *testing.T) []byte {
	data, err := json.Marshal(authRequest)
	if err != nil {
		t.Error(err)
	}
	return data
}

func (account *Account) to_json(t *testing.T) []byte {
	data, err := json.Marshal(account)
	if err != nil {
		t.Error(err)
	}
	return data
}

func (account *Account) to_protobuf(t *testing.T) []byte {
	data, err := proto.Marshal(account)
	if err != nil {
		t.Error(err)
	}
	return data
}

func TestNotAMember(t *testing.T) {

	signKey := []byte("test")
	settings := Settings{}

	auth := omni_auth.Init(omni_auth.Settings{SignKey: signKey, TTLHours: 0})
	service := &mock{}

	endpoint, err := NewApiEndPoint(settings, auth, service)
	if err != nil {
		t.Error(err)
	}

	ar := AuthRequest{
		Email:    ptr("foo@bar.com"),
		Password: ptr("test"),
	}

	service.findByEmail = func(email string) (account *Account, err error) {
		password := "test"
		return &Account{Primary: &Login{Password: &password}}, nil
	}

	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Post("/api/v1/auth", "applicaton/json", string(ar.to_json(t)))

		t.Log("Got response", response.Body)
		assert.Equal(t, 401, response.StatusCode)
		check_error_response_reason(t, response.Body, "error-not-a-member")
	})
}

func TestFoundAccountAndApplication(t *testing.T) {

	signKey := []byte("test")
	settings := Settings{
		ResolveApplicationId: func(req *http.Request) string {
			return "test-app"
		},
	}

	auth := omni_auth.Init(omni_auth.Settings{SignKey: signKey, TTLHours: 0})
	service := &mock{}

	endpoint, err := NewApiEndPoint(settings, auth, service)
	if err != nil {
		t.Error(err)
	}

	ar := &AuthRequest{
		Email:    ptr("foo@bar.com"),
		Password: ptr("test"),
	}

	applicationId := "test-app"
	applicationStatus := "verified"
	applicationAccountId := "12345"

	attributeType1 := Attribute_STRING
	embed := true
	attribute1, value1 := "attribute1", "value1"

	t.Log("test finding by email")
	service.findByEmail = func(email string) (account *Account, err error) {
		password := "test"
		return &Account{
			Primary: &Login{Password: &password},
			Services: []*Application{
				&Application{
					Id:        &applicationId,
					Status:    &applicationStatus,
					AccountId: &applicationAccountId,
					Attributes: []*Attribute{
						&Attribute{
							Type:             &attributeType1,
							Key:              &attribute1,
							EmbedSigninToken: &embed,
							StringValue:      &value1,
						},
					},
				},
			},
		}, nil
	}

	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Post("/api/v1/auth", "applicaton/json", string(ar.to_json(t)))

		t.Log("Got response", response.Body)
		assert.Equal(t, 200, response.StatusCode)

		dec := json.NewDecoder(strings.NewReader(response.Body))
		authResponse := AuthResponse{}

		if err := dec.Decode(&authResponse); err != nil {
			t.Error(err)
		}

		assert.NotEqual(t, "", authResponse.Token)

		// decode the token
		token, err := auth.Parse(*authResponse.Token)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, applicationStatus, token.GetString("@status"))
		assert.Equal(t, applicationAccountId, token.GetString("@accountId"))
		assert.Equal(t, value1, token.GetString(attribute1))
	})

	// test finding by phone
	t.Log("test finding by phone")
	service.findByPhone = service.findByEmail
	service.findByEmail = nil

	ar = &AuthRequest{
		Phone:    ptr("123-222-3333"),
		Password: ptr("test"),
	}
	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Post("/api/v1/auth", "applicaton/json", string(ar.to_json(t)))

		t.Log("Got response", response.Body)
		assert.Equal(t, 200, response.StatusCode)

		dec := json.NewDecoder(strings.NewReader(response.Body))
		authResponse := AuthResponse{}

		if err := dec.Decode(&authResponse); err != nil {
			t.Error(err)
		}

		assert.NotEqual(t, "", authResponse.Token)

		// decode the token
		token, err := auth.Parse(*authResponse.Token)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, applicationStatus, token.GetString("@status"))
		assert.Equal(t, applicationAccountId, token.GetString("@accountId"))
		assert.Equal(t, value1, token.GetString(attribute1))
	})
}

func TestFoundAccountButNotMatchApplication(t *testing.T) {

	signKey := []byte("test")
	settings := Settings{
		ResolveApplicationId: func(req *http.Request) string {
			t.Log("Calling resolve application")
			return "test-app-not-match"
		},
	}

	auth := omni_auth.Init(omni_auth.Settings{SignKey: signKey, TTLHours: 0})
	service := &mock{}

	endpoint, err := NewApiEndPoint(settings, auth, service)
	if err != nil {
		t.Error(err)
	}

	ar := &AuthRequest{
		Email:    ptr("foo@bar.com"),
		Password: ptr("test"),
	}

	applicationId := "test-app"
	applicationStatus := "verified"
	applicationAccountId := "12345"

	attributeType1 := Attribute_STRING
	embed := true
	attribute1, value1 := "attribute1", "value1"

	service.findByEmail = func(email string) (account *Account, err error) {
		password := "test"
		return &Account{
			Primary: &Login{Password: &password},
			Services: []*Application{
				&Application{
					Id:        &applicationId,
					Status:    &applicationStatus,
					AccountId: &applicationAccountId,
					Attributes: []*Attribute{
						&Attribute{
							Type:             &attributeType1,
							Key:              &attribute1,
							EmbedSigninToken: &embed,
							StringValue:      &value1,
						},
					},
				},
			},
		}, nil
	}

	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Post("/api/v1/auth", "applicaton/json", string(ar.to_json(t)))

		t.Log("Got response", response.Body)
		assert.Equal(t, 401, response.StatusCode)
		check_error_response_reason(t, response.Body, "error-not-a-member")
	})
}

func TestGetAccount(t *testing.T) {

	signKey := []byte("test")
	settings := Settings{
		ResolveApplicationId: func(req *http.Request) string {
			t.Log("Calling resolve application")
			return "test-app-not-match"
		},
	}

	auth := omni_auth.Init(omni_auth.Settings{SignKey: signKey, TTLHours: 0})
	service := &mock{}

	endpoint, err := NewApiEndPoint(settings, auth, service)
	if err != nil {
		t.Error(err)
	}

	password := "test"
	applicationId := "test-app"
	applicationStatus := "verified"
	applicationAccountId := "12345"

	attributeType1 := Attribute_STRING
	embed := true
	attribute1, value1 := "attribute1", "value1"

	service.getAccount = func(id string) (account *Account, err error) {
		assert.Equal(t, "1234", id)
		return &Account{
			Primary: &Login{Password: &password},
			Services: []*Application{
				&Application{
					Id:        &applicationId,
					Status:    &applicationStatus,
					AccountId: &applicationAccountId,
					Attributes: []*Attribute{
						&Attribute{
							Type:             &attributeType1,
							Key:              &attribute1,
							EmbedSigninToken: &embed,
							StringValue:      &value1,
						},
					},
				},
			},
		}, nil
	}
	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Get("/api/v1/account/1234")

		t.Log("Got response", response.Body)
		assert.Equal(t, 200, response.StatusCode)

		account := &Account{}
		dec := json.NewDecoder(strings.NewReader(response.Body))
		if err := dec.Decode(account); err != nil {
			t.Error(err)
		}

		assert.Equal(t, applicationId, account.GetServices()[0].GetId())
		assert.Equal(t, applicationStatus, account.GetServices()[0].GetStatus())
		assert.Equal(t, embed, account.GetServices()[0].GetAttributes()[0].GetEmbedSigninToken())
	})
}

func TestDeleteAccount(t *testing.T) {

	signKey := []byte("test")
	settings := Settings{
		ResolveApplicationId: func(req *http.Request) string {
			t.Log("Calling resolve application")
			return "test-app-not-match"
		},
	}

	auth := omni_auth.Init(omni_auth.Settings{SignKey: signKey, TTLHours: 0})
	service := &mock{}

	endpoint, err := NewApiEndPoint(settings, auth, service)
	if err != nil {
		t.Error(err)
	}

	service.deleteAccount = func(id string) (err error) {
		assert.Equal(t, "1234", id)
		return nil
	}
	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Delete("/api/v1/account/1234", "application/json", "")
		t.Log("Got response", response.Body)
		assert.Equal(t, 200, response.StatusCode)
	})
}

func TestSaveAccount(t *testing.T) {

	signKey := []byte("test")
	settings := Settings{}

	auth := omni_auth.Init(omni_auth.Settings{SignKey: signKey, TTLHours: 0})
	service := &mock{}

	endpoint, err := NewApiEndPoint(settings, auth, service)
	if err != nil {
		t.Error(err)
	}

	password := "test"
	applicationId := "test-app"
	applicationStatus := "verified"
	applicationAccountId := "12345"

	attributeType1 := Attribute_STRING
	embed := true
	attribute1, value1 := "attribute1", "value1"

	input := &Account{
		Primary: &Login{Password: &password},
		Services: []*Application{
			&Application{
				Id:        &applicationId,
				Status:    &applicationStatus,
				AccountId: &applicationAccountId,
				Attributes: []*Attribute{
					&Attribute{
						Type:             &attributeType1,
						Key:              &attribute1,
						EmbedSigninToken: &embed,
						StringValue:      &value1,
					},
				},
			},
		},
	}

	service.saveAccount = func(account *Account) (err error) {
		assert.NotEqual(t, "", account.GetId())
		t.Log("account id", account.GetId())
		return nil
	}

	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Post("/api/v1/account", "application/json", string(input.to_json(t)))
		assert.Equal(t, 200, response.StatusCode)
	})

	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Post("/api/v1/account", "application/protobuf", string(input.to_protobuf(t)))
		assert.Equal(t, 200, response.StatusCode)
	})

}
