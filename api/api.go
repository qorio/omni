package api

type AuthScope int
type AuthScopes map[AuthScope]string

type ServiceMethod int
type EventKey string

type ObjectFactory func() interface{}

type HttpMethod string

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
	HttpMethod           HttpMethod
	ContentTypes         []string
	RequestBody          ObjectFactory
	ResponseBody         ObjectFactory
	CallbackEvent        EventKey
	CallbackBodyTemplate string
	AuthScope            string
}

type ServiceMethods map[ServiceMethod]MethodSpec
