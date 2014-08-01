package lighthouse

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	omni_auth "github.com/qorio/omni/auth"
	omni_http "github.com/qorio/omni/http"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

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

	// ALPR
	api.router.HandleFunc("/api/v1/alpr", api.ApiMultiPartUpload).
		Methods("POST").Name("alpr-multipart")
	api.router.HandleFunc("/api/v1/alpr/{country}/{region}/{id}", api.ApiSingleUpload).
		Methods("POST").Name("alpr")
	api.router.HandleFunc("/api/v1/alpr/{country}/{region}/{id}", api.ApiGet).
		Methods("GET").Name("alpr-get")

	glog.Infoln("Api started")

	return api, nil
}

func (this *EndPoint) ServeHTTP(resp http.ResponseWriter, request *http.Request) {
	this.router.ServeHTTP(resp, request)
}

func getPath(root, country, region, id string) string {
	return filepath.Join(root, fmt.Sprintf("%s-%s-%s", country, region, id))
}

func (this *EndPoint) ApiGet(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)
	vars := mux.Vars(req)
	country := vars["country"]
	region := vars["region"]
	id := vars["id"]

	// TODO - check Content-Type

	path := getPath(this.settings.FsSettings.RootDir, country, region, id)
	glog.Infoln("Saving to file", path)

	f, err := os.Open(path)
	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	resp.Header().Add("Content-Type", "image/jpeg")
	resp.Header().Add("Content-Length", fmt.Sprintf("%d", stat.Size()))

	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	if copied, err := io.Copy(resp, f); err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	} else {
		glog.Infoln("sent", copied, "bytes")
	}
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
	resp.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", message)))
	return
}
