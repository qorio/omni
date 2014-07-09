package passport

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	omni_auth "github.com/qorio/omni/auth"
	omni_http "github.com/qorio/omni/http"
	"io/ioutil"
	"net/http"
	"strings"
)

type Settings struct {
}

type EndPoint struct {
	settings Settings
	router   *mux.Router
	auth     *omni_auth.Service
	service  *Service
}

func NewApiEndPoint(settings Settings, auth *omni_auth.Service, service *Service) (api *EndPoint, err error) {
	api = &EndPoint{
		settings: settings,
		router:   mux.NewRouter(),
		auth:     auth,
		service:  service,
	}

	api.router.HandleFunc("/api/v1/auth", api.ApiAuthenticate).
		Methods("POST").Name("auth")

	return api, nil
}

func (this *EndPoint) ServeHTTP(resp http.ResponseWriter, request *http.Request) {
	this.router.ServeHTTP(resp, request)
}

// Authenticates and returns a token as the response
func (this *EndPoint) ApiAuthenticate(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	var request AuthRequest
	dec := json.NewDecoder(strings.NewReader(string(body)))
	if err := dec.Decode(&request); err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	// do the lookup here...
	account, err := this.service.FindAccountByEmail(request.Email)
	switch {
	case err == ERROR_ACCOUNT_NOT_FOUND:
		renderJsonError(resp, req, "error-lookup-account", http.StatusUnauthorized)
		return
	case err != nil:
		renderJsonError(resp, req, "error-lookup-account", http.StatusInternalServerError)
	case err == nil && account.Primary.GetPassword() != request.Password:
		renderJsonError(resp, req, "error-lookup-account", http.StatusUnauthorized)
		return
	}

	// encode the token
	token := this.auth.NewToken()
	//token.Add(accountIdKey, account.GetId())

	tokenString, err := this.auth.SignedString(token)
	if err != nil {
		glog.Warningln("error-generating-auth-token", err)
		renderJsonError(resp, req, "cannot-generate-auth-token", http.StatusInternalServerError)
		return
	}

	// Response
	authResponse := AuthResponse{
		Token: tokenString,
	}

	buff, err := json.Marshal(authResponse)
	if err != nil {
		renderJsonError(resp, req, "malformed-response", http.StatusInternalServerError)
		return
	}
	resp.Write(buff)
}

func renderJsonError(resp http.ResponseWriter, req *http.Request, message string, code int) (err error) {
	resp.WriteHeader(code)
	resp.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", message)))
	return
}
