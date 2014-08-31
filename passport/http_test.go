package passport

import (
	"code.google.com/p/go-uuid/uuid"
	"code.google.com/p/goprotobuf/proto"
	"encoding/json"
	"github.com/bmizerany/assert"
	"github.com/drewolson/testflight"
	api "github.com/qorio/api/passport"
	omni_auth "github.com/qorio/omni/auth"
	omni_common "github.com/qorio/omni/common"
	"net/http"
	"strings"
	"testing"
)

type mock struct {
	findByEmail    func(email string) (account *api.Account, err error)
	findByPhone    func(phone string) (account *api.Account, err error)
	findByUsername func(username string) (account *api.Account, err error)
	findByOAuth2   func(provider, oauth2AccountId string) (account *api.Account, err error)

	saveAccount   func(account *api.Account) (err error)
	getAccount    func(id uuid.UUID) (account *api.Account, err error)
	deleteAccount func(id uuid.UUID) (err error)
}

func (this *mock) FindAccountByEmail(email string) (account *api.Account, err error) {
	return this.findByEmail(email)
}

func (this *mock) FindAccountByPhone(email string) (account *api.Account, err error) {
	return this.findByPhone(email)
}

func (this *mock) FindAccountByUsername(username string) (account *api.Account, err error) {
	return this.findByUsername(username)
}

func (this *mock) FindAccountByOAuth2(provider, accountId string) (account *api.Account, err error) {
	return this.findByOAuth2(provider, accountId)
}

func (this *mock) SaveAccount(account *api.Account) (err error) {
	return this.saveAccount(account)
}

func (this *mock) GetAccount(id uuid.UUID) (account *api.Account, err error) {
	return this.getAccount(id)
}

func (this *mock) DeleteAccount(id uuid.UUID) (err error) {
	return this.deleteAccount(id)
}

func (this *mock) Close() {
	return
}

func TestAuthNotFound(t *testing.T) {

	signKey := []byte("test")
	settings := Settings{}

	auth := omni_auth.Init(omni_auth.Settings{SignKey: signKey, TTLHours: 0})
	svc := &mock{
		findByEmail: func(email string) (account *api.Account, err error) {
			return nil, ERROR_NOT_FOUND
		},
		findByPhone: func(phone string) (account *api.Account, err error) {
			t.Error("testing look up by email; this shouldn't be called")
			return nil, nil
		},
	}

	endpoint, err := NewApiEndPoint(settings, auth, svc, nil, nil)

	if err != nil {
		t.Error(err)
	}

	authRequest := &api.Identity{
		Email:    proto.String("foo@bar.com"),
		Password: proto.String("test"),
	}

	data, err := json.Marshal(authRequest)
	if err != nil {
		t.Error(err)
	}

	// Account does not exist
	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Post("/api/v1/auth", "application/json", string(data))

		t.Log("Got response", response.Body)
		assert.Equal(t, 401, response.StatusCode)
		check_error_response_reason(t, response.Body, "error-account-not-found")
	})

	// Password does not match
	svc.findByEmail = func(email string) (account *api.Account, err error) {
		password := "not-a-match"
		return &api.Account{Primary: &api.Identity{Password: &password}}, nil
	}
	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Post("/api/v1/auth", "application/json", string(data))

		t.Log("Got response", response.Body)
		assert.Equal(t, 401, response.StatusCode)
		check_error_response_reason(t, response.Body, "error-bad-credentials")
	})

	// Find by phone
	authRequest.Email = proto.String("")
	authRequest.Phone = proto.String("123-111-2222")
	data, err = json.Marshal(authRequest)
	if err != nil {
		t.Error(err)
	}
	svc.findByEmail = func(email string) (account *api.Account, err error) {
		t.Error("should not call this function")
		return nil, nil
	}
	svc.findByPhone = func(email string) (account *api.Account, err error) {
		password := "no-match"
		return &api.Account{Primary: &api.Identity{Password: &password}}, nil
	}
	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Post("/api/v1/auth", "application/json", string(data))

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
		response := r.Post("/api/v1/auth", "application/json", string(data))

		t.Log("Got response", response.Body)
		assert.Equal(t, 400, response.StatusCode)
		check_error_response_reason(t, response.Body, "error-missing-input")
	})
}

func TestNotAMember(t *testing.T) {

	signKey := []byte("test")
	settings := Settings{}

	auth := omni_auth.Init(omni_auth.Settings{SignKey: signKey, TTLHours: 0})
	svc := &mock{}

	endpoint, err := NewApiEndPoint(settings, auth, svc, nil, nil)
	if err != nil {
		t.Error(err)
	}

	ar := api.Identity{
		Email:    proto.String("foo@bar.com"),
		Password: proto.String("test"),
	}

	svc.findByEmail = func(email string) (account *api.Account, err error) {
		password := "test"
		Password(&password).Hash().Update()
		return &api.Account{Primary: &api.Identity{Password: &password}}, nil
	}

	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Post("/api/v1/auth", "application/json", string(to_json(ar, t)))

		t.Log("Got response", response.Body)
		assert.Equal(t, 401, response.StatusCode)
		check_error_response_reason(t, response.Body, "error-not-a-member")
	})
}

func TestFoundAccountAndService(t *testing.T) {

	signKey := []byte("test")
	settings := Settings{
		ResolveServiceId: func(req *http.Request) string {
			return "test-app"
		},
	}

	auth := omni_auth.Init(omni_auth.Settings{SignKey: signKey, TTLHours: 0})
	svc := &mock{}

	endpoint, err := NewApiEndPoint(settings, auth, svc, nil, nil)
	if err != nil {
		t.Error(err)
	}

	ar := &api.Identity{
		Email:    proto.String("foo@bar.com"),
		Password: proto.String("test"),
	}

	serviceId := "test-app"
	serviceStatus := "verified"
	serviceAccountId := "12345"
	attribute1, value1 := "attribute1", "value1"

	t.Log("test finding by email")
	svc.findByEmail = func(email string) (account *api.Account, err error) {
		password := "test"
		Password(&password).Hash().Update()
		return &api.Account{
			Primary: &api.Identity{Password: proto.String(password)},
			Services: []*api.Service{
				&api.Service{
					Id:        proto.String(serviceId),
					Status:    proto.String(serviceStatus),
					AccountId: proto.String(serviceAccountId),
					Scopes: []string{
						"admin",
						"readwrite",
					},
					Attributes: []*api.Attribute{
						&api.Attribute{
							Type:         api.Attribute_STRING.Enum(),
							Key:          proto.String(attribute1),
							EmbedInToken: proto.Bool(true),
							StringValue:  proto.String(value1),
						},
					},
				},
			},
		}, nil
	}

	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		t.Log("Testing authRequest in json")
		response := r.Post("/api/v1/auth", "application/json", string(to_json(ar, t)))

		t.Log("Got response", response.Body)
		assert.Equal(t, 200, response.StatusCode)

		dec := json.NewDecoder(strings.NewReader(response.Body))
		authResponse := api.AuthResponse{}

		if err := dec.Decode(&authResponse); err != nil {
			t.Error(err)
		}

		assert.NotEqual(t, "", authResponse.Token)

		// decode the token
		token, err := auth.Parse(*authResponse.Token)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, serviceStatus, token.GetString("test-app/@status"))
		assert.Equal(t, serviceAccountId, token.GetString("test-app/@id"))
		assert.Equal(t, "admin,readwrite", token.GetString("test-app/@scopes"))
		assert.Equal(t, value1, token.GetString("test-app/"+attribute1))
	})

	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		t.Log("Testing authRequest in protobuf")
		response := r.Post("/api/v1/auth", "application/protobuf", string(to_protobuf(ar, t)))

		t.Log("Got response", response.Body)
		assert.Equal(t, 200, response.StatusCode)

		authResponse := api.AuthResponse{}
		if err := proto.Unmarshal(response.RawBody, &authResponse); err != nil {
			t.Error(err)
		}

		assert.NotEqual(t, "", authResponse.Token)

		// decode the token
		token, err := auth.Parse(*authResponse.Token)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, serviceStatus, token.GetString("test-app/@status"))
		assert.Equal(t, serviceAccountId, token.GetString("test-app/@id"))
		assert.Equal(t, "admin,readwrite", token.GetString("test-app/@scopes"))
		assert.Equal(t, value1, token.GetString("test-app/"+attribute1))
	})

	// test finding by phone
	t.Log("test finding by phone")
	svc.findByPhone = svc.findByEmail
	svc.findByEmail = nil

	ar = &api.Identity{
		Phone:    proto.String("123-222-3333"),
		Password: proto.String("test"),
	}
	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Post("/api/v1/auth", "application/json", string(to_json(ar, t)))

		t.Log("Got response", response.Body)
		assert.Equal(t, 200, response.StatusCode)

		dec := json.NewDecoder(strings.NewReader(response.Body))
		authResponse := api.AuthResponse{}

		if err := dec.Decode(&authResponse); err != nil {
			t.Error(err)
		}

		assert.NotEqual(t, "", authResponse.Token)

		// decode the token
		token, err := auth.Parse(*authResponse.Token)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, serviceStatus, token.GetString("test-app/@status"))
		assert.Equal(t, serviceAccountId, token.GetString("test-app/@id"))
		assert.Equal(t, value1, token.GetString("test-app/"+attribute1))
	})
}

func TestFoundAccountButNotMatchService(t *testing.T) {

	signKey := []byte("test")
	settings := Settings{
		ResolveServiceId: func(req *http.Request) string {
			t.Log("Calling resolve service")
			return "test-app-not-match"
		},
	}

	auth := omni_auth.Init(omni_auth.Settings{SignKey: signKey, TTLHours: 0})
	svc := &mock{}

	endpoint, err := NewApiEndPoint(settings, auth, svc, nil, nil)
	if err != nil {
		t.Error(err)
	}

	ar := &api.Identity{
		Email:    proto.String("foo@bar.com"),
		Password: proto.String("test"),
	}

	serviceId := "test-app"
	serviceStatus := "verified"
	serviceAccountId := "12345"

	attributeType1 := api.Attribute_STRING
	embed := true
	attribute1, value1 := "attribute1", "value1"

	svc.findByEmail = func(email string) (account *api.Account, err error) {
		password := "test"
		Password(&password).Hash().Update()
		return &api.Account{
			Primary: &api.Identity{Password: &password},
			Services: []*api.Service{
				&api.Service{
					Id:        &serviceId,
					Status:    &serviceStatus,
					AccountId: &serviceAccountId,
					Attributes: []*api.Attribute{
						&api.Attribute{
							Type:         &attributeType1,
							Key:          &attribute1,
							EmbedInToken: &embed,
							StringValue:  &value1,
						},
					},
				},
			},
		}, nil
	}

	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Post("/api/v1/auth/not-test-app", "application/json", string(to_json(ar, t)))

		t.Log("Got response", response.Body)
		assert.Equal(t, 401, response.StatusCode)
		check_error_response_reason(t, response.Body, "error-not-a-member")
	})
}

var no_auth = func() bool { return false }

func TestGetAccount(t *testing.T) {

	signKey := []byte("test")
	settings := Settings{
		ResolveServiceId: func(req *http.Request) string {
			t.Log("Calling resolve service")
			return "test-app-not-match"
		},
	}

	auth := omni_auth.Init(omni_auth.Settings{SignKey: signKey, TTLHours: 0, IsAuthOn: no_auth})
	svc := &mock{}

	endpoint, err := NewApiEndPoint(settings, auth, svc, nil, nil)
	if err != nil {
		t.Error(err)
	}

	password := "test"
	serviceId := "test-app"
	serviceStatus := "verified"
	serviceAccountId := "12345"

	attribute1, value1 := "attribute1", "value1"

	svc.getAccount = func(id uuid.UUID) (account *api.Account, err error) {
		return &api.Account{
			Primary: &api.Identity{Password: &password},
			Services: []*api.Service{
				&api.Service{
					Id:        &serviceId,
					Status:    &serviceStatus,
					AccountId: &serviceAccountId,
					Attributes: []*api.Attribute{
						&api.Attribute{
							Type:         api.Attribute_STRING.Enum(),
							Key:          &attribute1,
							EmbedInToken: proto.Bool(true),
							StringValue:  &value1,
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

		account := &api.Account{}
		dec := json.NewDecoder(strings.NewReader(response.Body))
		if err := dec.Decode(account); err != nil {
			t.Error(err)
		}

		assert.Equal(t, serviceId, account.GetServices()[0].GetId())
		assert.Equal(t, serviceStatus, account.GetServices()[0].GetStatus())
		assert.Equal(t, true, account.GetServices()[0].GetAttributes()[0].GetEmbedInToken())
	})
}

func TestDeleteAccount(t *testing.T) {

	signKey := []byte("test")
	settings := Settings{
		ResolveServiceId: func(req *http.Request) string {
			t.Log("Calling resolve service")
			return "test-app-not-match"
		},
	}

	auth := omni_auth.Init(omni_auth.Settings{SignKey: signKey, TTLHours: 0, IsAuthOn: no_auth})
	svc := &mock{}

	endpoint, err := NewApiEndPoint(settings, auth, svc, nil, nil)
	if err != nil {
		t.Error(err)
	}

	svc.deleteAccount = func(id uuid.UUID) (err error) {
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

	auth := omni_auth.Init(omni_auth.Settings{SignKey: signKey, TTLHours: 0, IsAuthOn: no_auth})
	svc := &mock{}

	endpoint, err := NewApiEndPoint(settings, auth, svc, nil, nil)
	if err != nil {
		t.Error(err)
	}

	password := "test"
	serviceId := "test-app"
	serviceStatus := "verified"
	serviceAccountId := "12345"

	attributeType1 := api.Attribute_STRING
	embed := true
	attribute1, value1 := "attribute1", "value1"

	input := &api.Account{
		Primary: &api.Identity{
			Email:    proto.String("test@foo.com"),
			Password: &password},
		Services: []*api.Service{
			&api.Service{
				Id:        &serviceId,
				Status:    &serviceStatus,
				AccountId: &serviceAccountId,
				Attributes: []*api.Attribute{
					&api.Attribute{
						Type:         &attributeType1,
						Key:          &attribute1,
						EmbedInToken: &embed,
						StringValue:  &value1,
					},
				},
			},
		},
	}

	svc.findByEmail = func(email string) (*api.Account, error) {
		return nil, ERROR_NOT_FOUND
	}
	svc.saveAccount = func(account *api.Account) (err error) {
		assert.NotEqual(t, "", account.GetId())
		t.Log("account id", account.GetId())
		return nil
	}

	t.Log("using application/json serialization")
	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Post("/api/v1/account", "application/json", string(to_json(input, t)))
		t.Log("Got response", response)
		assert.Equal(t, 200, response.StatusCode)
	})

	t.Log("using application/protobuf serialization")
	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Post("/api/v1/account", "application/protobuf", string(to_protobuf(input, t)))
		assert.Equal(t, 200, response.StatusCode)
	})

}

func TestNewAccount(t *testing.T) {

	signKey := []byte("test")
	settings := Settings{}

	auth := omni_auth.Init(omni_auth.Settings{SignKey: signKey, TTLHours: 0, IsAuthOn: no_auth})
	svc := &mock{}

	endpoint, err := NewApiEndPoint(settings, auth, svc, nil, nil)
	if err != nil {
		t.Error(err)
	}

	input := &api.Account{
		Primary: &api.Identity{
			Password: proto.String("password"),
			Phone:    proto.String("111-222-9999"),
		},
	}

	var state struct {
		CalledFindByPhone bool
		CalledSaveAccount bool
	}
	svc.findByPhone = func(phone string) (account *api.Account, err error) {
		(&state).CalledFindByPhone = true
		return nil, ERROR_NOT_FOUND
	}
	svc.saveAccount = func(account *api.Account) (err error) {
		(&state).CalledSaveAccount = true
		assert.NotEqual(t, "", account.GetId())
		t.Log("account id", account.GetId())
		return nil
	}

	t.Log("using application/json serialization")
	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Post("/api/v1/account", "application/json", string(to_json(input, t)))
		assert.Equal(t, 200, response.StatusCode)
	})

	t.Log("using application/protobuf serialization")
	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Post("/api/v1/account", "application/protobuf", string(to_protobuf(input, t)))
		assert.Equal(t, 200, response.StatusCode)
	})

	assert.Equal(t, true, state.CalledFindByPhone)
	assert.Equal(t, true, state.CalledSaveAccount)
}

func TestNewAccountMissingInput(t *testing.T) {

	signKey := []byte("test")
	settings := Settings{}

	auth := omni_auth.Init(omni_auth.Settings{SignKey: signKey, TTLHours: 0, IsAuthOn: no_auth})
	svc := &mock{}

	endpoint, err := NewApiEndPoint(settings, auth, svc, nil, nil)
	if err != nil {
		t.Error(err)
	}

	input := &api.Account{
		Primary: &api.Identity{Password: proto.String("foo")},
	}

	var state struct {
		CalledFindByPhone bool
		CalledSaveAccount bool
	}
	svc.findByPhone = func(phone string) (account *api.Account, err error) {
		(&state).CalledFindByPhone = true
		return nil, ERROR_NOT_FOUND
	}
	svc.saveAccount = func(account *api.Account) (err error) {
		(&state).CalledSaveAccount = true
		assert.NotEqual(t, "", account.GetId())
		t.Log("account id", account.GetId())
		return nil
	}

	t.Log("using application/protobuf serialization")
	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Post("/api/v1/account", "application/protobuf", string(to_protobuf(input, t)))
		assert.Equal(t, 400, response.StatusCode)
	})

	assert.Equal(t, false, state.CalledFindByPhone)
	assert.Equal(t, false, state.CalledSaveAccount)
}

func TestNewAccountConflict(t *testing.T) {

	signKey := []byte("test")
	settings := Settings{}

	auth := omni_auth.Init(omni_auth.Settings{SignKey: signKey, TTLHours: 0, IsAuthOn: no_auth})
	svc := &mock{}

	endpoint, err := NewApiEndPoint(settings, auth, svc, nil, nil)
	if err != nil {
		t.Error(err)
	}

	input := &api.Account{
		Primary: &api.Identity{
			Password: proto.String("password"),
			Phone:    proto.String("111-222-9999"),
		},
	}

	var state struct {
		CalledFindByPhone bool
		CalledSaveAccount bool
	}

	svc.findByPhone = func(phone string) (account *api.Account, err error) {
		(&state).CalledFindByPhone = true
		return input, nil
	}
	svc.saveAccount = func(account *api.Account) (err error) {
		assert.NotEqual(t, "", account.GetId())
		assert.NotEqual(t, "", account.GetPrimary().GetId())
		t.Log("account id", account.GetId())
		(&state).CalledSaveAccount = true
		return nil
	}

	t.Log("using application/protobuf serialization")
	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Post("/api/v1/account", "application/protobuf", string(to_protobuf(input, t)))
		assert.Equal(t, 409, response.StatusCode)
		check_error_response_reason(t, response.Body, "error-conflict")
	})

	assert.Equal(t, true, state.CalledFindByPhone)
	assert.Equal(t, false, state.CalledSaveAccount)
}

func TestSaveAccountPrimay(t *testing.T) {

	signKey := []byte("test")
	settings := Settings{}

	auth := omni_auth.Init(omni_auth.Settings{SignKey: signKey, TTLHours: 0, IsAuthOn: no_auth})
	svc := &mock{}

	endpoint, err := NewApiEndPoint(settings, auth, svc, nil, nil)
	if err != nil {
		t.Error(err)
	}

	input := &api.Account{
		Primary: &api.Identity{},
		Services: []*api.Service{
			&api.Service{
				Attributes: []*api.Attribute{
					&api.Attribute{},
				},
			},
		},
	}

	embed := true
	attribute_type := api.Attribute_STRING

	uid := omni_common.NewUUID()
	input.Id = proto.String(uid.String())
	input.Primary.Password = proto.String("password-1")
	input.Services[0].Id = proto.String("app-1")
	input.Services[0].Status = proto.String("verified")
	input.Services[0].AccountId = proto.String("app-account-1")
	input.Services[0].Attributes[0].Key = proto.String("key-1")
	input.Services[0].Attributes[0].Type = &attribute_type
	input.Services[0].Attributes[0].EmbedInToken = &embed
	input.Services[0].Attributes[0].StringValue = proto.String("value-1")

	svc.getAccount = func(id uuid.UUID) (account *api.Account, err error) {
		assert.Equal(t, input.GetId(), id.String())
		t.Log("account id", id)
		return input, nil
	}

	login := &api.Identity{}
	login.Password = proto.String("new-password")
	login.Phone = proto.String("111-222-3333")

	svc.saveAccount = func(account *api.Account) (err error) {
		assert.Equal(t, input.GetId(), account.GetId())
		t.Log("account id", account.GetId(), "primary", account.GetPrimary())
		assert.Equal(t, login.GetPhone(), account.GetPrimary().GetPhone())
		assert.Equal(t, login.GetPassword(), account.GetPrimary().GetPassword())
		return nil
	}

	t.Log("using application/json serialization")
	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Post("/api/v1/account/"+input.GetId()+"/primary", "application/json", string(to_json(login, t)))
		t.Log("Got response", response.Body)
		assert.Equal(t, 200, response.StatusCode)
	})

	t.Log("using application/protobuf serialization")
	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Post("/api/v1/account/"+input.GetId()+"/primary", "application/protobuf", string(to_protobuf(login, t)))
		t.Log("Got response", response.Body)
		assert.Equal(t, 200, response.StatusCode)
	})
}

func TestSaveAccountService(t *testing.T) {

	signKey := []byte("test")
	settings := Settings{}

	auth := omni_auth.Init(omni_auth.Settings{SignKey: signKey, TTLHours: 0, IsAuthOn: no_auth})
	svc := &mock{}

	endpoint, err := NewApiEndPoint(settings, auth, svc, nil, nil)
	if err != nil {
		t.Error(err)
	}

	// initially no services
	input := &api.Account{
		Primary: &api.Identity{},
	}

	embed := true
	attribute_type := api.Attribute_STRING

	uid := omni_common.NewUUID()
	input.Id = proto.String(uid.String())
	input.Primary.Password = proto.String("password-1")

	service := &api.Service{Attributes: []*api.Attribute{&api.Attribute{}}}
	service.Id = proto.String(omni_common.NewUUID().String())
	service.Status = proto.String("verified")
	service.AccountId = proto.String("app-account-1")
	service.Attributes[0].Key = proto.String("key-1")
	service.Attributes[0].Type = &attribute_type
	service.Attributes[0].EmbedInToken = &embed
	service.Attributes[0].StringValue = proto.String("value-1")

	svc.getAccount = func(id uuid.UUID) (account *api.Account, err error) {
		assert.Equal(t, input.GetId(), id.String())
		t.Log("account id", id)
		return input, nil
	}

	svc.saveAccount = func(account *api.Account) (err error) {
		assert.Equal(t, input.GetId(), account.GetId())
		assert.Equal(t, 1, len(account.GetServices()))
		assert.Equal(t, service, account.GetServices()[0])
		return nil
	}

	t.Log("using application/json serialization")
	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Post("/api/v1/account/"+input.GetId()+"/services", "application/json", string(to_json(service, t)))
		t.Log("Got response", response.Body)
		assert.Equal(t, 200, response.StatusCode)
	})

	t.Log("using application/protobuf serialization")
	// now we change an app's attribute
	service.Attributes[0].StringValue = proto.String("value-1-changed")
	svc.saveAccount = func(account *api.Account) (err error) {
		assert.Equal(t, input.GetId(), account.GetId())
		assert.Equal(t, 1, len(account.GetServices()))
		assert.Equal(t, service, account.GetServices()[0])
		return nil
	}

	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Post("/api/v1/account/"+input.GetId()+"/services", "application/protobuf", string(to_protobuf(service, t)))
		t.Log("Got response", response.Body)
		assert.Equal(t, 200, response.StatusCode)
	})

	// Now do a get
	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Get("/api/v1/account/" + input.GetId())

		t.Log("Got response", response.Body)
		assert.Equal(t, 200, response.StatusCode)

		account := &api.Account{}
		dec := json.NewDecoder(strings.NewReader(response.Body))
		if err := dec.Decode(account); err != nil {
			t.Error(err)
		}

		assert.Equal(t, "value-1-changed", account.GetServices()[0].GetAttributes()[0].GetStringValue())
	})

}

func TestSaveAccountServiceAttribute(t *testing.T) {

	signKey := []byte("test")
	settings := Settings{}

	auth := omni_auth.Init(omni_auth.Settings{SignKey: signKey, TTLHours: 0, IsAuthOn: no_auth})
	svc := &mock{}

	endpoint, err := NewApiEndPoint(settings, auth, svc, nil, nil)
	if err != nil {
		t.Error(err)
	}

	// initially no services
	input := &api.Account{
		Primary: &api.Identity{},
	}

	embed := true
	attribute_type := api.Attribute_STRING

	input.Id = proto.String(omni_common.NewUUID().String())
	input.Primary.Password = proto.String("password-1")

	service := &api.Service{Attributes: []*api.Attribute{&api.Attribute{}}}
	service.Id = proto.String(omni_common.NewUUID().String())
	service.Status = proto.String("verified")
	service.AccountId = proto.String("app-account-1")
	service.Attributes[0].Key = proto.String("key-1")
	service.Attributes[0].Type = &attribute_type
	service.Attributes[0].EmbedInToken = &embed
	service.Attributes[0].StringValue = proto.String("value-1")

	svc.getAccount = func(id uuid.UUID) (account *api.Account, err error) {
		assert.Equal(t, input.GetId(), id.String())
		t.Log("account id", id)
		return input, nil
	}

	svc.saveAccount = func(account *api.Account) (err error) {
		assert.Equal(t, input.GetId(), account.GetId())
		assert.Equal(t, 1, len(account.GetServices()))
		assert.Equal(t, service, account.GetServices()[0])
		return nil
	}

	t.Log("using application/json serialization")
	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Post("/api/v1/account/"+input.GetId()+"/services", "application/json", string(to_json(service, t)))
		t.Log("Got response", response.Body)
		assert.Equal(t, 200, response.StatusCode)
	})

	t.Log("using application/protobuf serialization")
	attribute := service.Attributes[0]
	attribute.StringValue = proto.String("value-1-changed")
	svc.saveAccount = func(account *api.Account) (err error) {
		assert.Equal(t, input.GetId(), account.GetId())
		assert.Equal(t, 1, len(account.GetServices()))
		assert.Equal(t, attribute.GetStringValue(), account.GetServices()[0].GetAttributes()[0].GetStringValue())
		return nil
	}

	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Post("/api/v1/account/"+input.GetId()+"/service/"+service.GetId()+"/attributes", "application/protobuf",
			string(to_protobuf(attribute, t)))
		t.Log("Got response", response.Body)
		assert.Equal(t, 200, response.StatusCode)
	})

	// add a new attribute
	attribute2_t := api.Attribute_STRING
	attribute2 := &api.Attribute{
		Type:        &attribute2_t,
		Key:         proto.String("new-key"),
		StringValue: proto.String("new-value"),
	}

	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Post("/api/v1/account/"+input.GetId()+"/service/"+service.GetId()+"/attributes", "application/protobuf",
			string(to_protobuf(attribute2, t)))
		t.Log("Got response", response.Body)
		assert.Equal(t, 200, response.StatusCode)
	})

	// Now do a get
	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Get("/api/v1/account/" + input.GetId())

		t.Log("Got response", response.Body)
		assert.Equal(t, 200, response.StatusCode)

		account := &api.Account{}
		dec := json.NewDecoder(strings.NewReader(response.Body))
		if err := dec.Decode(account); err != nil {
			t.Error(err)
		}

		assert.Equal(t, "value-1-changed", account.GetServices()[0].GetAttributes()[0].GetStringValue())
		assert.Equal(t, 2, len(account.GetServices()[0].GetAttributes()))
	})

}
