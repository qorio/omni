package main

import (
	"flag"
	"fmt"
	"github.com/golang/glog"
	"github.com/qorio/omni/shorty"
	"net/http"
)

var redisHost = flag.String("redis_host", "", "Redis host (leave empty for localhost)")
var redisPort = flag.Int("redis_port", 6379, "Redis port")
var redisPrefix = flag.String("redis_prefix", "goshorty:", "Redis prefix to use")
var restrictDomain = flag.String("restrict_domain", "", "Restrict destination URLs to a single domain")
var redirect404 = flag.String("redirect_404", "", "Global redirect when no record found")
var urlLength = flag.Int("url_length", 7, "How many characters should the short code have")
var port = flag.Int("port", 8080, "Port where server is listening on")
var geoDbFilePath = flag.String("geo_db", "./GeoIP.dat", "Location to the MaxMind GeoIP country database file")

func main() {

	flag.Parse()

	settings := shorty.Settings{
		RedisUrl:       fmt.Sprintf("%s:%d", *redisHost, *redisPort),
		RedisPrefix:    *redisPrefix,
		RestrictDomain: *restrictDomain,
		UrlLength:      *urlLength,
	}

	shortyService := shorty.Init(settings)
	if endpoint, err := shorty.NewApiEndPoint(shorty.ApiEndPointSettings{
		Redirect404:     *redirect404,
		GeoIpDbFilePath: *geoDbFilePath,
	}, shortyService); err == nil {

		glog.Infoln("Server listening on port", *port)

		err = http.ListenAndServe(fmt.Sprintf(":%d", *port), endpoint)
		if err != nil {
			panic(err)
		}
	} else {
		panic(err)
	}
}
