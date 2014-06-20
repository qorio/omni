package shorty

import (
	"flag"
	"github.com/golang/glog"
	omni_http "github.com/qorio/omni/http"
)

var (
	fingerPrintExpirationMinutes = flag.Int64("fingerprint_expiration_minutes", 2, "Minutes TTL matching by fingerprint")
	fingerPrintMinMatchingScore  = flag.Float64("fingerprint_min_score", 0.8, "Minimum score to match by fingerprint")
)

func init() {
	var err error
	secureCookie, err = omni_http.NewSecureCookie([]byte(""), nil)
	if err != nil {
		glog.Warningln("Cannot initialize secure cookie!")
		panic(err)
	}
}
