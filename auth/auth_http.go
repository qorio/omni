package auth

import (
	"fmt"
	"github.com/golang/glog"
	"net/http"
	"strings"
)

type Info struct {
	AppKey UUID
}

type HttpHandler func(auth *Info, resp http.ResponseWriter, req *http.Request)

func (service *Service) RequiresAuth(handler HttpHandler) func(http.ResponseWriter, *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		info := &Info{
			AppKey: UUID("dev"),
		}
		if *doAuth {
			// Get the auth header
			header := req.Header.Get("Authorization")
			if header == "" {
				renderError(resp, req, "Missing auth token", http.StatusUnauthorized)
				return
			}

			token := strings.Trim(strings.TrimLeft(header, "Bearer "), " ")
			appKey, err := service.GetAppKey(token)
			if err != nil {
				glog.Warningln("auth-error", err)
				renderError(resp, req, err.Error(), http.StatusUnauthorized)
				return
			}

			// Get the auth info
			info.AppKey = appKey
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
