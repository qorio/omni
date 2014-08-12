package api

type ServiceMethod int

type ObjectFactory func() interface{}

type MethodSpec struct {
	Doc          string
	Name         string
	UrlRoute     string
	HttpMethod   string
	ContentTypes []string
	RequestBody  ObjectFactory
	ResponseBody ObjectFactory
	RequiresAuth bool
}
