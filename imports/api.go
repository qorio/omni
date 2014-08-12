package api

import (
	"net/http"
)

type ObjectFactory func() interface{}

type ServiceMethod struct {
	Doc          string
	Name         string
	UrlRoute     string
	HttpMethod   string
	ContentTypes []string
	RequestBody  ObjectFactory
	ResponseBody ObjectFactory
}
