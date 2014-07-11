package passport

import (
	"code.google.com/p/goprotobuf/proto"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	omni_auth "github.com/qorio/omni/auth"
	omni_common "github.com/qorio/omni/common"
	omni_http "github.com/qorio/omni/http"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

var (
	ERROR_UNKNOWN_CONTENT_TYPE = errors.New("error-no-content-type")
)

type Settings struct {
	// Function that takes the http request and determine the application id
	// The default is to take the request's URL host, e.g. qor.io or shorty.qor.io
	ResolveApplicationId func(req *http.Request) string
}

type EndPoint struct {
	settings Settings
	router   *mux.Router
	auth     *omni_auth.Service
	service  Service
}

func defaultResolveApplicationId(req *http.Request) string {
	return req.URL.Host
}

func NewApiEndPoint(settings Settings, auth *omni_auth.Service, service Service) (api *EndPoint, err error) {
	api = &EndPoint{
		settings: settings,
		router:   mux.NewRouter(),
		auth:     auth,
		service:  service,
	}

	// Authentication endpoint
	api.router.HandleFunc("/api/v1/auth", api.ApiAuthenticate).
		Methods("POST").Name("auth")

	// Account management endpoints
	api.router.HandleFunc("/api/v1/account", api.ApiSaveAccount).
		Methods("POST").Name("account-save")
	api.router.HandleFunc("/api/v1/account/{id}/primary", api.ApiSaveAccountPrimary).
		Methods("POST").Name("account-login-update")
	api.router.HandleFunc("/api/v1/account/{id}/services", api.ApiSaveAccountService).
		Methods("POST").Name("account-services-update")
	api.router.HandleFunc("/api/v1/account/{id}", api.ApiGetAccount).
		Methods("GET").Name("account-get")
	api.router.HandleFunc("/api/v1/account/{id}", api.ApiDeleteAccount).
		Methods("DELETE").Name("account-delete")

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
	var account *Account

	switch {
	case request.GetEmail() != "":
		account, err = this.service.FindAccountByEmail(request.GetEmail())
	case request.GetPhone() != "":
		account, err = this.service.FindAccountByPhone(request.GetPhone())
	case request.GetPhone() == "" && request.GetEmail() == "":
		renderJsonError(resp, req, "error-no-phone-or-email", http.StatusBadRequest)
		return
	}

	switch {
	case err == ERROR_ACCOUNT_NOT_FOUND:
		renderJsonError(resp, req, "error-account-not-found", http.StatusUnauthorized)
		return
	case err != nil:
		renderJsonError(resp, req, "error-lookup-account", http.StatusInternalServerError)
		return
	case err == nil && account.Primary.GetPassword() != request.GetPassword():
		renderJsonError(resp, req, "error-bad-credentials", http.StatusUnauthorized)
		return
	}

	// now look for the application
	requestedApplicationId := request.GetApplication()
	if requestedApplicationId == "" {
		if this.settings.ResolveApplicationId != nil {
			requestedApplicationId = this.settings.ResolveApplicationId(req)
		} else {
			requestedApplicationId = defaultResolveApplicationId(req)
		}
	}
	var application *Application
	for _, test := range account.GetServices() {
		if test.GetId() == requestedApplicationId {
			application = test
			break
		}
	}

	if application == nil {
		renderJsonError(resp, req, "error-not-a-member", http.StatusUnauthorized)
		return
	}

	// encode the token
	token := this.auth.NewToken()
	token.Add("@id", application.GetId()).
		Add("@status", application.GetStatus()).
		Add("@accountId", application.GetAccountId())

	for _, attribute := range application.GetAttributes() {
		if attribute.GetEmbedSigninToken() {
			switch attribute.GetType() {
			case Attribute_STRING:
				token.Add(attribute.GetKey(), attribute.GetStringValue())
			case Attribute_NUMBER:
				token.Add(attribute.GetKey(), attribute.GetNumberValue())
			case Attribute_BOOL:
				token.Add(attribute.GetKey(), attribute.GetBoolValue())
			case Attribute_BLOB:
				token.Add(attribute.GetKey(), attribute.GetBlobValue())
			}
		}
	}
	tokenString, err := this.auth.SignedString(token)
	if err != nil {
		glog.Warningln("error-generating-auth-token", err)
		renderJsonError(resp, req, "cannot-generate-auth-token", http.StatusInternalServerError)
		return
	}

	// Response
	authResponse := &AuthResponse{
		Token: &tokenString,
	}

	buff, err := json.Marshal(authResponse)
	if err != nil {
		renderJsonError(resp, req, "malformed-response", http.StatusInternalServerError)
		return
	}
	resp.Write(buff)
}

func unmarshal(contentType string, body io.ReadCloser, typed proto.Message) (err error) {
	switch {
	case contentType == "application/json":
		dec := json.NewDecoder(body)
		return dec.Decode(typed)
	case contentType == "application/protobuf":
		buff, err := ioutil.ReadAll(body)
		if err != nil {
			return err
		}
		return proto.Unmarshal(buff, typed)
	default:
		return ERROR_UNKNOWN_CONTENT_TYPE
	}
}

func (this *EndPoint) ApiSaveAccount(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)

	account := &Account{}
	err := unmarshal(req.Header.Get("Content-Type"), req.Body, account)
	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	if account.GetId() == "" {
		uuid, _ := omni_common.NewUUID()
		account.Id = &uuid
	}
	if account.GetPrimary().GetId() == "" {
		uuid, _ := omni_common.NewUUID()
		account.GetPrimary().Id = &uuid
	}

	err = this.service.SaveAccount(account)
	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (this *EndPoint) ApiSaveAccountPrimary(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)
	vars := mux.Vars(req)
	id := vars["id"]

	login := &Login{}
	err := unmarshal(req.Header.Get("Content-Type"), req.Body, login)
	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	if id == "" {
		renderJsonError(resp, req, "error-missing-id", http.StatusBadRequest)
		return
	}

	account, err := this.service.GetAccount(id)

	switch {
	case err == ERROR_ACCOUNT_NOT_FOUND:
		renderJsonError(resp, req, "error-account-not-found", http.StatusNotFound)
		return
	case err != nil:
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	switch {
	case login.GetPhone() == "" && login.GetEmail() == "":
		renderJsonError(resp, req, "error-missing-email-or-phone", http.StatusBadRequest)
		return
	case login.GetPassword() == "":
		renderJsonError(resp, req, "error-missing-password", http.StatusBadRequest)
		return
	}

	if login.GetId() == "" {
		uuid, _ := omni_common.NewUUID()
		login.Id = &uuid
	}

	// update the primary
	account.Primary = login

	err = this.service.SaveAccount(account)
	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (this *EndPoint) ApiSaveAccountService(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)
	vars := mux.Vars(req)
	id := vars["id"]

	application := &Application{}
	err := unmarshal(req.Header.Get("Content-Type"), req.Body, application)
	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	if id == "" {
		renderJsonError(resp, req, "error-missing-id", http.StatusBadRequest)
		return
	}

	account, err := this.service.GetAccount(id)

	switch {
	case err == ERROR_ACCOUNT_NOT_FOUND:
		renderJsonError(resp, req, "error-account-not-found", http.StatusNotFound)
		return
	case err != nil:
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	// find the application by id and replace it
	if len(account.GetServices()) == 0 {
		account.Services = []*Application{
			application,
		}
	} else {
		for i, app := range account.GetServices() {
			if app.GetId() == application.GetId() {
				account.Services[i] = application
				break
			}
		}
	}

	err = this.service.SaveAccount(account)
	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (this *EndPoint) ApiGetAccount(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)
	vars := mux.Vars(req)
	id := vars["id"]

	if id == "" {
		renderJsonError(resp, req, "error-missing-id", http.StatusBadRequest)
		return
	}

	account, err := this.service.GetAccount(id)

	switch {
	case err == ERROR_ACCOUNT_NOT_FOUND:
		renderJsonError(resp, req, "error-account-not-found", http.StatusNotFound)
		return
	case err != nil:
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	buff, err := json.Marshal(account)
	if err != nil {
		renderJsonError(resp, req, "malformed-account", http.StatusInternalServerError)
		return
	}
	resp.Write(buff)
}

func (this *EndPoint) ApiDeleteAccount(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)
	vars := mux.Vars(req)
	id := vars["id"]

	if id == "" {
		renderJsonError(resp, req, "error-missing-id", http.StatusBadRequest)
		return
	}

	err := this.service.DeleteAccount(id)

	switch {
	case err == ERROR_ACCOUNT_NOT_FOUND:
		renderJsonError(resp, req, "error-account-not-found", http.StatusNotFound)
		return
	case err != nil:
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}
}

func renderJsonError(resp http.ResponseWriter, req *http.Request, message string, code int) (err error) {
	resp.WriteHeader(code)
	resp.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", message)))
	return
}
