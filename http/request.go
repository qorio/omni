package http

import (
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/mssola/user_agent"
	"github.com/nranchev/go-libGeoIP"
	"net/http"
	"strings"
)

type RequestParser struct {
	geoIp *libgeo.GeoIP
}

type Location struct {
	CountryCode string
	CountryName string
	Region      string
	City        string
	PostalCode  string
	Latitude    float64
	Longitude   float64
}

type UserAgent struct {
	Bot            bool
	Mobile         bool
	Platform       string // Linux, iPhone, iPad
	OS             string // Android, OS 7_1
	Make           string // samsung, kindle, etc.
	Browser        string
	BrowserVersion string
	Header         string // original header
}

type RequestOrigin struct {
	Ip          string
	Referrer    string
	UserAgent   *UserAgent
	Location    *Location
	HttpRequest *http.Request
	Cookied     bool
	Visits      int
	Destination string
	ShortCode   string
	LastVisit   string
}

func FingerPrint(origin *RequestOrigin) string {
	if origin.Location == nil {
		return origin.Ip // Not very good...
	}
	return strings.Join([]string{
		origin.Ip,
		fmt.Sprintf("%.5f", origin.Location.Longitude),
		fmt.Sprintf("%.5f", origin.Location.Latitude),
	}, ":")
}

func GetFingerPrintMatchQuery(fingerprint string) string {
	return fingerprint + "*"
}

func MatchFingerPrint(fp string, fingerprints []string) (match string, score float64) {
	// TODO - given list of candidates, return the highest score match
	for _, test := range fingerprints {
		if test == fp {
			score = 1.
		}

		glog.V(50).Infoln("Checking fingerprint", fp, ", candidate=", test, score)
		if score >= 1. {
			match = test
			return
		}

		// TODO - Otherwise, look for partial matches
		parts := strings.Split(test, ":")
		if test == parts[0] { // by ip
			return test, 0.8
		}
	}
	return "", 0.
}

func NewRequestParser(geoDb string) (parser *RequestParser, err error) {
	parser = &RequestParser{}
	parser.geoIp, err = libgeo.Load(geoDb)
	if err != nil {
		return nil, err
	}
	return parser, nil
}

func ParseUserAgent(req *http.Request) *UserAgent {
	ua := new(user_agent.UserAgent)
	ua.Parse(req.UserAgent())
	val := &UserAgent{
		Header:   req.UserAgent(),
		Bot:      ua.Bot(),
		Mobile:   ua.Mobile(),
		Platform: ua.Platform(),
		OS:       ua.OS(),
		Make:     req.UserAgent(), // TODO - fix the library
	}
	val.Browser, val.BrowserVersion = ua.Browser()
	return val
}

func (this *RequestParser) Parse(req *http.Request) (r *RequestOrigin, err error) {
	ip, location, _ := this.geo(req)
	r = &RequestOrigin{
		HttpRequest: req,
		Ip:          ip,
		UserAgent:   ParseUserAgent(req),
	}

	if location != nil {
		r.Location = &Location{
			CountryCode: location.CountryCode,
			CountryName: location.CountryName,
			Region:      location.Region,
			City:        location.City,
			PostalCode:  location.PostalCode,
			Latitude:    float64(location.Latitude),
			Longitude:   float64(location.Longitude),
		}
	} else {
		r.Location = &Location{}
	}

	r.Referrer = req.Referer()
	if r.Referrer == "" {
		r.Referrer = "DIRECT"
	}

	return r, err
}

func (this *RequestParser) geo(req *http.Request) (string, *libgeo.Location, error) {
	ip := req.Header.Get("X-Real-Ip")
	forwarded := req.Header.Get("X-Forwarded-For")
	if ip == "" && forwarded == "" {
		i := strings.LastIndex(req.RemoteAddr, ":")
		if i != -1 {
			ip = req.RemoteAddr[:i]
		} else {
			ip = req.RemoteAddr
		}
	} else if forwarded != "" {
		// X-Forwarded-For is potentially a list of addresses separated with ","
		parts := strings.Split(forwarded, ",")
		// TODO: should return first non-local address
		ip = parts[0]
	}

	ip = strings.TrimSpace(ip)
	if ip == "" {
		return "", nil, errors.New("Could not obtain IP address from request")
	} else if ip == "[::1]" {
		ip = "50.184.95.238" // fake ip -- for local development etc.
	}

	if location := this.geoIp.GetLocationByIP(ip); location == nil {
		return ip, nil, nil
	} else {
		return ip, location, nil
	}
}

func (this *RequestParser) Browser(req *http.Request) (bot bool, mobile bool, os string, browser string, version string) {
	ua := new(user_agent.UserAgent)
	ua.Parse(req.UserAgent())
	browserName, browserVersion := ua.Browser()
	return ua.Bot(), ua.Mobile(), ua.OS(), browserName, browserVersion
}
