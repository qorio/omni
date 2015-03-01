package api

import (
	"net/http"
)

type AuthScope int
type AuthScopes map[AuthScope]string

type ServiceMethod int
type EventKey string

type ObjectFactory func(*http.Request) interface{}

type HttpMethod string
type QueryDefault interface{}
type UrlQueries map[string]QueryDefault
type HttpHeaders map[string]string

var (
	GET       HttpMethod = HttpMethod("GET")
	POST      HttpMethod = HttpMethod("POST")
	PUT       HttpMethod = HttpMethod("PUT")
	DELETE    HttpMethod = HttpMethod("DELETE")
	MULTIPART HttpMethod = HttpMethod("POST")
)

type MethodSpec struct {
	Doc                  string
	UrlRoute             string
	HttpHeaders          HttpHeaders
	HttpMethod           HttpMethod
	UrlQueries           UrlQueries
	ContentTypes         []string
	RequestBody          ObjectFactory
	ResponseBody         ObjectFactory
	CallbackEvent        EventKey
	CallbackBodyTemplate string
	AuthScope            string
}

type ServiceMethods map[ServiceMethod]MethodSpec
