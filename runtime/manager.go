package runtime

import (
	"encoding/json"
	"github.com/gorilla/mux"
	omni_http "github.com/qorio/omni/http"
	"net/http"
	"time"
)

type Info struct {
	Uptime         float64 `json:"uptime_seconds"`
	Commit         string  `json:"git_commit"`
	BuildTimestamp string  `json:"build_timestamp"`
	BuildNumber    string  `json:"build"`
}

var (
	startTime time.Time
)

func init() {
	startTime = time.Now()
}

func NewManagerEndPoint() http.Handler {
	router := mux.NewRouter()
	router.HandleFunc("/update", StartUpdateHandler).Methods("POST").Name("update")
	router.HandleFunc("/info", InfoHandler).Methods("GET").Name("info")
	return router
}

func InfoHandler(resp http.ResponseWriter, request *http.Request) {
	omni_http.SetCORSHeaders(resp)
	buildInfo := BuildInfo()
	info := Info{
		Commit:         buildInfo.Commit,
		BuildTimestamp: buildInfo.Timestamp,
		BuildNumber:    buildInfo.Number,
		Uptime:         time.Since(startTime).Seconds(),
	}
	enc := json.NewEncoder(resp)
	_ = enc.Encode(info)
	return
}
