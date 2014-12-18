package auth

import (
	"fmt"
	"github.com/golang/glog"
	"net/http"
	"strings"
)

type Context interface {
	HasKey(key string) bool
	GetString(key string) string
	Get(key string) interface{}
	GetStringFromService(service, key string) string
}

type context struct {
	token *Token
}

func (this *context) HasKey(key string) bool {
	return this.token.HasKey(key)
}

func (this *context) GetString(key string) string {
	return this.token.GetString(key)
}

func (this *context) GetStringFromService(service, key string) string {
	return this.token.GetString(fmt.Sprintf("%s/%s", service, key))
}

func (this *context) Get(key string) interface{} {
	return this.token.Get(key)
}

func (service *serviceImpl) RequiresAuth(scope string, get_scopes GetScopesFromToken, handler HttpHandler) func(http.ResponseWriter, *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		info := &context{}
		checkAuth := true
		if service.IsAuthOn != nil {
			checkAuth = service.IsAuthOn()
		} else {
			checkAuth = *doAuth
		}

		authed := false
		if checkAuth {

			// Format:  Authorization: Bearer|token|Oauth + ' ' + <token>

			// Get the auth header
			header := req.Header.Get("Authorization")
			if header == "" {
				renderError(resp, req, "Missing auth token", http.StatusUnauthorized)
				return
			}

			tokenString := strings.SplitAfterN(header, " ", 2)[1]
			token, err := service.Parse(tokenString)
			if err != nil {
				glog.Warningln("auth-error", err)
				renderError(resp, req, err.Error(), http.StatusUnauthorized)
				return
			}
			info.token = token

			// Check the scope
			scopes := []string{}
			if get_scopes != nil {
				scopes = get_scopes(info.token)
			} else {
				scopes = strings.Split(info.token.GetString("@scopes"), ",")
			}
			if service.CheckScope != nil {
				authed = service.CheckScope(scope, scopes)
			} else {
				for _, s := range scopes {
					if s == scope {
						authed = true
						break
					}
				}
			}
		} else {
			authed = true
		}

		if authed {
			handler(info, resp, req)
			return
		} else {
			// error
			renderError(resp, req, "not-permitted", http.StatusUnauthorized)
			return
		}
	}
}

func renderError(resp http.ResponseWriter, req *http.Request, message string, code int) (err error) {
	resp.WriteHeader(code)
	resp.Write([]byte(fmt.Sprintf("<html><body>Error: %s </body></html>", message)))
	return
}
