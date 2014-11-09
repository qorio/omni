package http

import (
	"net/http"
)

func SetNoCachingHeaders(w http.ResponseWriter) {
	w.Header().Add("Pragma", "no-cache")
	w.Header().Add("Cache-Control", "no-cache, no-store, max-age=0, must-revalidate")
	w.Header().Add("Expires", "Mon, 01 Jan 1990 00:00:00 GMT")
}

// http://enable-cors.org/server_nginx.html
// TODO - add support for OPTIONS call for preflight check
func SetCORSHeaders(w http.ResponseWriter) {
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Credentials", "true")
}
