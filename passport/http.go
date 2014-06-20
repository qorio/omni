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
}

func NewApiEndPoint(settings Settings, service *omni_auth.Service) (api *EndPoint, err error) {
	api = &EndPoint{
		settings: settings,
		router:   mux.NewRouter(),
		auth:     service,
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
	appKey := omni_auth.UUID("test-key")
	// encode the token

	token, err := this.auth.GetAppToken(appKey)
	if err != nil {
		glog.Warningln("error-generating-auth-token", err)
		renderJsonError(resp, req, "cannot-generate-auth-token", http.StatusInternalServerError)
		return
	}

	// Response
	authResponse := AuthResponse{
		Token: token,
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
