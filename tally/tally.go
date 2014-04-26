package tally

import (
	"encoding/json"
	"math"
	"regexp"
	"strconv"
	"time"
)

var nanoseconds = math.Pow10(9)
var quoted = regexp.MustCompile("^\"|\"$")

func NewEvent() *Event {
	now := to_seconds(time.Now().UnixNano())
	return &Event{
		Timestamp: &now,
	}
}

func (this *Event) SetAttribute(key, value string) {
	this.Attributes = append(this.Attributes, parse_attribute(key, value))
}

func (this *Event) ToJSON(indent bool) (bytes []byte, err error) {
	bytes, err = format_json(this, indent)
	return
}

func unix_timestamp(secs float64) string {
	t := time.Unix(int64(secs/nanoseconds), int64(secs*nanoseconds))
	return t.Format(time.RFC3339Nano)
}

func to_seconds(nanos int64) float64 {
	return float64(nanos) / nanoseconds
}

func to_geojson(loc *Location) []float64 {
	return []float64{*loc.Lon, *loc.Lat}
}

func format_json(event *Event, indent bool) (bytes []byte, err error) {
	payload := map[string]interface{}{
		"@timestamp": unix_timestamp(*event.Timestamp),
		"@appKey":    event.AppKey,
		"@type":      event.Type,
		"@source":    event.Source,
		"@context":   event.Context,
		"@location":  to_geojson(event.Location),
	}
	for _, attr := range event.Attributes {
		if attr.BoolValue != nil {
			payload[*attr.Key] = attr.BoolValue
		} else if attr.IntValue != nil {
			payload[*attr.Key] = attr.IntValue
		} else if attr.DoubleValue != nil {
			payload[*attr.Key] = attr.DoubleValue
		} else if attr.StringValue != nil {
			payload[*attr.Key] = attr.StringValue
		}
	}
	if indent {
		return json.MarshalIndent(payload, "", "    ")
	} else {
		return json.Marshal(payload)
	}

}

func parse_attribute(key string, value string) *Attribute {
	attr := &Attribute{
		Key: &key,
	}
	if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
		attr.DoubleValue = &floatValue
		return attr
	} else if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
		attr.IntValue = &intValue
		return attr
	} else if boolValue, err := strconv.ParseBool(value); err == nil {
		attr.BoolValue = &boolValue
		return attr
	} else {
		s := quoted.ReplaceAllString(value, "")
		attr.StringValue = &s
		return attr
	}
}
