package api

type AuthScope int
type AuthScopes map[AuthScope]string

type ServiceMethod int
type EventKey string

type ObjectFactory func() interface{}

type MethodSpec struct {
	Doc                  string
	UrlRoute             string
	HttpMethod           string
	ContentTypes         []string
	RequestBody          ObjectFactory
	ResponseBody         ObjectFactory
	CallbackEvent        EventKey
	CallbackBodyTemplate string
	AuthScope            string
}

type ServiceMethods map[ServiceMethod]MethodSpec
