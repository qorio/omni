package auth

import (
	"fmt"
	"github.com/golang/glog"
	"net/http"
	"strings"
)

type Context struct {
	token *Token
}

func (this *Context) HasKey(key string) bool {
	return this.token.HasKey(key)
}

func (this *Context) GetString(key string) string {
	return this.token.GetString(key)
}

func (this *Context) Get(key string) interface{} {
	return this.token.Get(key)
}

type HttpHandler func(auth *Context, resp http.ResponseWriter, req *http.Request)

func (service *Service) RequiresAuth(handler HttpHandler) func(http.ResponseWriter, *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		info := &Context{}
		if *doAuth {
			// Get the auth header
			header := req.Header.Get("Authorization")
			if header == "" {
				renderError(resp, req, "Missing auth token", http.StatusUnauthorized)
				return
			}

			tokenString := strings.Trim(strings.TrimLeft(header, "Bearer "), " ")
			token, err := service.Parse(tokenString)
			if err != nil {
				glog.Warningln("auth-error", err)
				renderError(resp, req, err.Error(), http.StatusUnauthorized)
				return
			}

			info.token = token
		}
		glog.Infoln("AuthHandler", info)
		handler(info, resp, req)
	}
}

func renderError(resp http.ResponseWriter, req *http.Request, message string, code int) (err error) {
	resp.WriteHeader(code)
	resp.Write([]byte(fmt.Sprintf("<html><body>Error: %s </body></html>", message)))
	return
}
