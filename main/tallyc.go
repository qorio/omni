package main

import (
	"flag"
	"github.com/golang/glog"
	"github.com/qorio/omni/tally"
	"strings"
)

var (
	timestamp  = flag.String("timestamp", "", "Event timestamp")
	appKey     = flag.String("appkey", "", "AppKey")
	eventType  = flag.String("type", "event", "Event type")
	context    = flag.String("context", "", "Event context")
	source     = flag.String("source", "", "Event source")
	lat        = flag.Float64("lat", 0., "Event location:latitude")
	lon        = flag.Float64("lon", 0., "Event location:longitude")
	attributes = flag.String("attributes", "", "Event attributes, {key=value;}+")
)

func main() {
	flag.Parse()

	event := tally.NewEvent()
	event.AppKey = appKey
	event.Type = eventType
	event.Source = event.Source
	event.Context = event.Context
	event.Location = &tally.Location{
		Lon: lon,
		Lat: lat,
	}

	for i, p := range strings.Split(*attributes, ";") {
		kv := strings.Split(p, ":")
		if len(kv) == 2 {
			glog.Infof("i=%d Key=%s, Value=%s", i, kv[0], kv[1])
			event.SetAttribute(kv[0], kv[1])
		}
	}

	if json, err := event.ToJSON(); err == nil {
		glog.Infof("JSON2 = %s", json)
	}
}
