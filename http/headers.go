package http

import (
	"net/http"
)

func SetNoCachingHeaders(w http.ResponseWriter) {
	w.Header().Add("Pragma", "no-cache")
	w.Header().Add("Cache-Control", "no-cache, no-store, max-age=0, must-revalidate")
	w.Header().Add("Expires", "Mon, 01 Jan 1990 00:00:00 GMT")
}

func SetCORSHeaders(w http.ResponseWriter) {
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Access-Control-Allow-Origin", "*")
}
