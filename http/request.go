package http

import (
	"errors"
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
	Latitude    float32
	Longitude   float32
}

type RequestOrigin struct {
	Ip       string
	Referrer string
	Bot      bool
	Mobile   bool
	OS       string
	Browser  string
	Version  string
	Location *Location
}

func NewRequestParser(geoDb string) (parser *RequestParser, err error) {
	parser = &RequestParser{}
	parser.geoIp, err = libgeo.Load(geoDb)
	if err != nil {
		return nil, err
	}
	return parser, nil
}

func (this *RequestParser) Parse(req *http.Request) (r *RequestOrigin, err error) {
	ip, location, _ := this.geo(req)
	r = &RequestOrigin{
		Ip: ip,
	}

	if location != nil {
		r.Location = &Location{
			CountryCode: location.CountryCode,
			CountryName: location.CountryName,
			Region:      location.Region,
			City:        location.City,
			PostalCode:  location.PostalCode,
			Latitude:    location.Latitude,
			Longitude:   location.Longitude,
		}
	}

	ua := new(user_agent.UserAgent)
	ua.Parse(req.UserAgent())
	r.Referrer = req.Referer()
	if r.Referrer == "" {
		r.Referrer = "DIRECT"
	}
	r.Bot = ua.Bot()
	r.Mobile = ua.Mobile()
	r.OS = ua.OS()
	r.Browser, r.Version = ua.Browser()
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
