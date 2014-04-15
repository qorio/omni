package shorty

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	omni_http "github.com/qorio/omni/http"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type ApiAddRequest struct {
	LongUrl string
}

type ApiEndPointSettings struct {
	Redirect404     string
	GeoIpDbFilePath string
}

type ApiEndPoint struct {
	settings      ApiEndPointSettings
	router        *mux.Router
	requestParser *omni_http.RequestParser
	service       Shorty
}

func NewApiEndPoint(settings ApiEndPointSettings, service Shorty) (api *ApiEndPoint, err error) {
	if requestParser, err := omni_http.NewRequestParser(settings.GeoIpDbFilePath); err == nil {
		api = &ApiEndPoint{
			settings:      settings,
			router:        mux.NewRouter(),
			requestParser: requestParser,
			service:       service,
		}

		// configure router
		api.router.HandleFunc("/api/v1/url", api.ApiAddHandler).Methods("POST").Name("add")

		regex := fmt.Sprintf("[A-Za-z0-9]{%d}", service.UrlLength())
		api.router.HandleFunc("/{id:"+regex+"}", api.RedirectHandler).Name("redirect")
	}
	return
}

func (this *ApiEndPoint) ApiAddHandler(resp http.ResponseWriter, req *http.Request) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	var message ApiAddRequest
	dec := json.NewDecoder(strings.NewReader(string(body)))
	for {
		if err := dec.Decode(&message); err == io.EOF {
			break
		} else if err != nil {
			renderJsonError(resp, req, err.Error(), http.StatusBadRequest)
			return
		}
	}

	if message.LongUrl == "" {
		renderJsonError(resp, req, "No URL to shorten", http.StatusBadRequest)
		return
	}

	shortUrl, err := this.service.ShortUrl(message.LongUrl)
	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	url, err := this.router.Get("redirect").URL("id", shortUrl.Id)
	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	json := fmt.Sprintf("{\"id\":\"http://%s%s\",\"longUrl\":\"%s\"}", req.Host, url, shortUrl.Destination)
	resp.Write([]byte(json))
}

func (this *ApiEndPoint) RedirectHandler(resp http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	shortUrl, err := this.service.Find(vars["id"])
	if err != nil {
		renderError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	} else if shortUrl == nil {
		if this.settings.Redirect404 != "" {
			originalUrl, err := this.router.Get("redirect").URL("id", vars["id"])
			if err != nil {
				renderError(resp, req, err.Error(), http.StatusInternalServerError)
				return
			}
			url404 := strings.Replace(this.settings.Redirect404, "$gosURL", url.QueryEscape(fmt.Sprintf("http://%s%s", req.Host, originalUrl.String())), 1)
			http.Redirect(resp, req, url404, http.StatusTemporaryRedirect)
			return
		}
		renderError(resp, req, "No URL was found with that goshorty code", http.StatusNotFound)
		return
	}

	// Record stats asynchronously
	go func() {
		if origin, err := this.requestParser.Parse(req); err == nil {
			shortUrl.Record(origin)
		}
	}()
	http.Redirect(resp, req, shortUrl.Destination, http.StatusMovedPermanently)
}

func renderJsonError(resp http.ResponseWriter, req *http.Request, message string, code int) (err error) {
	resp.WriteHeader(code)
	resp.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", message)))
	return
}

func renderError(resp http.ResponseWriter, req *http.Request, message string, code int) (err error) {
	/*
		body, err := render(req, "layout", "error", map[string]string{"Error": message})
		if err != nil {
			http.Error(resp, err.Error(), http.StatusInternalServerError)
			return
		}

		resp.WriteHeader(code)
		resp.Write(body)
	*/
	return nil
}
