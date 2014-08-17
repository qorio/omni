package shorty

import (
	"encoding/json"
	"flag"
	"io"
	"testing"
)

var (
	authKeyFile = flag.String("auth_public_key_file", "", "Auth public key file")
)

func from_json(o interface{}, src io.Reader, t *testing.T) interface{} {
	err := json.NewDecoder(src).Decode(&o)
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func to_json(o interface{}, t *testing.T) []byte {
	data, err := json.Marshal(o)
	if err != nil {
		t.Error(err)
	}
	return data
}
