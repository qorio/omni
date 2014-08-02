package blinker

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	omni_auth "github.com/qorio/omni/auth"
	omni_http "github.com/qorio/omni/http"
	"io"
	"net/http"
	"os"
	"strings"
)

type EndPoint struct {
	settings Settings
	router   *mux.Router
	auth     *omni_auth.Service
	service  Service
}

func NewApiEndPoint(settings Settings, auth *omni_auth.Service, service Service) (api *EndPoint, err error) {
	api = &EndPoint{
		settings: settings,
		router:   mux.NewRouter(),
		auth:     auth,
		service:  service,
	}

	// ALPR
	api.router.HandleFunc("/api/v1/alpr", api.ApiMultiPartUpload).
		Methods("POST").Name("alpr-multipart")
	api.router.HandleFunc("/api/v1/alpr/{country}/{region}/{id}", api.ApiExecAlpr).
		Methods("POST").Name("alpr")
	api.router.HandleFunc("/api/v1/alpr/{country}/{region}/{id}", api.ApiGet).
		Methods("GET").Name("alpr-get")

	api.router.HandleFunc("/api/v1/images/{country}/{region}/{id}", api.ApiSingleUpload).
		Methods("POST").Name("alpr")
	api.router.HandleFunc("/api/v1/images/{country}/{region}/{id}", api.ApiGet).
		Methods("GET").Name("alpr-get")

	glog.Infoln("Api started")

	return api, nil
}

func (this *EndPoint) ServeHTTP(resp http.ResponseWriter, request *http.Request) {
	this.router.ServeHTTP(resp, request)
}

func (this *EndPoint) ApiGet(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)
	vars := mux.Vars(req)
	country := vars["country"]
	region := vars["region"]
	id := vars["id"]

	bytes, size, err := this.service.GetImage(country, region, id)
	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	resp.Header().Add("Content-Type", "image/jpeg")
	resp.Header().Add("Content-Length", fmt.Sprintf("%d", size))

	if copied, err := io.Copy(resp, bytes); err != nil {
		glog.Infoln("ERROR", err)
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	} else {
		glog.Infoln("sent", copied, "bytes")
	}
}

func (this *EndPoint) ApiExecAlpr(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)
	vars := mux.Vars(req)
	country := vars["country"]
	region := vars["region"]
	id := vars["id"]

	stdout, err := this.service.ExecAlpr(country, region, id, req.Body)

	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	resp.Write(stdout)
}

func (this *EndPoint) ApiSingleUpload(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)
	vars := mux.Vars(req)
	country := vars["country"]
	region := vars["region"]
	id := vars["id"]

	// TODO - check Content-Type

	path := getPath(this.settings.FsSettings.RootDir, country, region, id)
	glog.Infoln("Saving to file", path)

	dst, err := os.Create(path)
	defer dst.Close()

	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	if copied, err := io.Copy(dst, req.Body); err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	} else {
		glog.Infoln("copied", copied, "bytes")
	}
}

func (this *EndPoint) ApiMultiPartUpload(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)

	sink := omni_http.FileSystemSink(this.settings.FsSettings.RootDir)
	err := omni_http.ProcessMultiPartUpload(resp, req, sink)

	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}
}

func renderJsonError(resp http.ResponseWriter, req *http.Request, message string, code int) (err error) {
	resp.WriteHeader(code)
	resp.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", strings.Replace(message, "\"", "'", -1))))
	return
}
