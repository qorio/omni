package http

import (
	"github.com/qorio/omni/auth"
	"net/http"
	"reflect"
)

type Handler func(http.ResponseWriter, *http.Request)
type AuthenticatedHandler func(*auth.Context, http.ResponseWriter, *http.Request)

type RestEndPoint struct {
	UrlRoute             string
	HttpMethod           string
	ContentTypes         []string
	RequestBody          reflect.Type
	ResponseBody         reflect.Type
	Doc                  string
	Handler              Handler
	AuthenticatedHandler AuthenticatedHandler
}
