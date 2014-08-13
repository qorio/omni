package lighthouse

import (
	"flag"
)

var (
	authKeyFile = flag.String("auth_public_key_file", "", "Auth public key file")
)

func ptr(s string) *string {
	return &s
}
