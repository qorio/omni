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

func TestHandler(t *testing.T) {

	signKey := []byte("test")
	settings := Settings{}

	auth := omni_auth.Init(omni_auth.Settings{SignKey: signKey, TTLHours: 0})
	service := NewService(settings)

	endpoint, err := NewApiEndPoint(settings, auth, service)

	if err != nil {
		t.Error(err)
	}

	// Does not exist
	authRequest := &AuthRequest{
		Email:    "foo@bar.com",
		Password: "test",
	}

	data, err := json.Marshal(authRequest)
	if err != nil {
		t.Error(err)
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
		assert.Equal(t, "error-account-not-found", reason)
	})
}
