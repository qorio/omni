package passport

import (
	"encoding/json"
	"flag"
	"github.com/bmizerany/assert"
	"github.com/drewolson/testflight"
	omni_auth "github.com/qorio/omni/auth"
	"strings"
	"testing"
)

var (
	authKeyFile = flag.String("auth_public_key_file", "", "Auth public key file")
)

type mock struct {
	findByEmail func(email string) (account *Account, err error)
	findByPhone func(phone string) (account *Account, err error)
}

func (this *mock) FindAccountByEmail(email string) (account *Account, err error) {
	return this.findByEmail(email)
}

func (this *mock) FindAccountByPhone(email string) (account *Account, err error) {
	return this.findByPhone(email)
}

func ptr(s string) *string {
	return &s
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

		dec := json.NewDecoder(strings.NewReader(response.Body))
		authResponse := make(map[string]string)

		if err := dec.Decode(&authResponse); err != nil {
			t.Error(err)
		}

		reason, has := authResponse["error"]
		assert.Equal(t, true, has)
		assert.Equal(t, "error-account-not-found", reason)
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

		dec := json.NewDecoder(strings.NewReader(response.Body))
		authResponse := make(map[string]string)

		if err := dec.Decode(&authResponse); err != nil {
			t.Error(err)
		}

		reason, has := authResponse["error"]
		assert.Equal(t, true, has)
		assert.Equal(t, "error-bad-credentials", reason)
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

		dec := json.NewDecoder(strings.NewReader(response.Body))
		authResponse := make(map[string]string)

		if err := dec.Decode(&authResponse); err != nil {
			t.Error(err)
		}

		reason, has := authResponse["error"]
		assert.Equal(t, true, has)
		assert.Equal(t, "error-bad-credentials", reason)
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

		dec := json.NewDecoder(strings.NewReader(response.Body))
		authResponse := make(map[string]string)

		if err := dec.Decode(&authResponse); err != nil {
			t.Error(err)
		}

		reason, has := authResponse["error"]
		assert.Equal(t, true, has)
		assert.Equal(t, "error-no-phone-or-email", reason)
	})
}

func to_json(t *testing.T, authRequest *AuthRequest) []byte {
	data, err := json.Marshal(authRequest)
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

	data := to_json(t, &AuthRequest{
		Email:    ptr("foo@bar.com"),
		Password: ptr("test"),
	})

	service.findByEmail = func(email string) (account *Account, err error) {
		password := "test"
		return &Account{Primary: &Login{Password: &password}}, nil
	}

	testflight.WithServer(endpoint, func(r *testflight.Requester) {
		response := r.Post("/api/v1/auth", "applicaton/json", string(data))

		t.Log("Got response", response.Body)
		assert.Equal(t, 401, response.StatusCode)

		dec := json.NewDecoder(strings.NewReader(response.Body))
		authResponse := make(map[string]string)

		if err := dec.Decode(&authResponse); err != nil {
			t.Error(err)
		}

		reason, has := authResponse["error"]
		assert.Equal(t, true, has)
		assert.Equal(t, "error-not-a-member", reason)
	})
}

func TestFoundAccountAndApplication(t *testing.T) {

	signKey := []byte("test")
	settings := Settings{}

	auth := omni_auth.Init(omni_auth.Settings{SignKey: signKey, TTLHours: 0})
	service := &mock{}

	endpoint, err := NewApiEndPoint(settings, auth, service)
	if err != nil {
		t.Error(err)
	}

	data := to_json(t, &AuthRequest{
		Email:    ptr("foo@bar.com"),
		Password: ptr("test"),
	})

	applicationId := "" // will match the mock url
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
		response := r.Post("/api/v1/auth", "applicaton/json", string(data))

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
