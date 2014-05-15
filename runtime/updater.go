package runtime

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/inconshreveable/go-update"
	omni_http "github.com/qorio/omni/http"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

type UpdateExecutableRequest struct {
	DownloadUrl string `json:"downloadUrl"`
}

type UpdateResult struct {
	Error        error
	RecoverError error
}

func NewManagerEndPoint() http.Handler {
	router := mux.NewRouter()
	router.HandleFunc("/update", StartUpdateHandler).Methods("POST").Name("update")
	return router
}

func StartUpdateHandler(resp http.ResponseWriter, request *http.Request) {
	omni_http.SetCORSHeaders(resp)
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		renderJsonError(resp, request, err.Error(), http.StatusInternalServerError)
		return
	}

	var message UpdateExecutableRequest
	dec := json.NewDecoder(strings.NewReader(string(body)))
	for {
		if err := dec.Decode(&message); err == io.EOF {
			break
		} else if err != nil {
			renderJsonError(resp, request, err.Error(), http.StatusBadRequest)
			return
		}
	}

	if message.DownloadUrl == "" {
		renderJsonError(resp, request, "update-no-download-url", http.StatusBadRequest)
		return
	}

	// Blocks while update is happening
	result := <-RunUpdate(&message)

	if result.Error != nil {
		renderJsonError(resp, request, "update-error-download", http.StatusInternalServerError)
		return
	} else if result.RecoverError != nil {
		renderJsonError(resp, request, "update-recover-error", http.StatusInternalServerError)
		return
	}

	resp.Write([]byte(fmt.Sprintf("{\"status\":\"installed\"}")))
	return
}

func RunUpdate(request *UpdateExecutableRequest) <-chan UpdateResult {
	resultChan := make(chan UpdateResult)
	go func() {
		updater := update.New()

		glog.Infoln("Starting update executable from", request.DownloadUrl)

		err, recoverErr := updater.FromUrl(request.DownloadUrl)
		if err != nil {
			glog.Warningln("update-executable-error", err)
		}

		resultChan <- UpdateResult{
			Error:        err,
			RecoverError: recoverErr,
		}
	}()
	return resultChan
}

func renderJsonError(resp http.ResponseWriter, req *http.Request, message string, code int) (err error) {
	resp.WriteHeader(code)
	resp.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", message)))
	return
}
