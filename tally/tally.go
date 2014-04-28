package tally

import (
	"encoding/json"
	"github.com/garyburd/redigo/redis"
	"github.com/golang/glog"
	"math"
	"regexp"
	"strconv"
	"time"
)

type Settings struct {
	RedisUrl     string
	RedisChannel string
}

type Tally interface {
	Channel() chan<- *Event
	Start()
	Stop()
	Close()
}

type tallyImpl struct {
	settings Settings
	pool     *redis.Pool
	channel  chan *Event
	stop     chan bool
}

func Init(settings Settings) *tallyImpl {
	return &tallyImpl{
		settings: settings,
		channel:  make(chan *Event),
		stop:     make(chan bool),
		pool: &redis.Pool{
			MaxIdle:     5,
			IdleTimeout: 240 * time.Second,
			Dial: func() (redis.Conn, error) {
				c, err := redis.Dial("tcp", settings.RedisUrl)
				if err != nil {
					return nil, err
				}
				return c, nil
			},
			TestOnBorrow: func(c redis.Conn, t time.Time) error {
				_, err := c.Do("PING")
				return err
			},
		},
	}
}

func (this *tallyImpl) Channel() chan<- *Event {
	return this.channel
}

func (this *tallyImpl) Start() {
	go func() {
		for {
			select {
			case message := <-this.channel:
				count, err := this.publish(message)
				if err != nil {
					glog.Warningln("error-publish", err, this)
				} else if count == 0 {
					glog.Warningln("no-subscribers", this)
				}
			case stop := <-this.stop:
				if stop {
					return
				}
			}
		}
	}()
}

func (this *tallyImpl) Stop() {
	this.stop <- true
}

func (this *tallyImpl) Close() {
	if err := this.pool.Close(); err != nil {
		glog.Warningln("error-closing-pool", err)
	}
}

func (this *tallyImpl) publish(event *Event) (count int, err error) {
	c := this.pool.Get()
	defer c.Close()

	data, err := event.ToJSON(false)
	if err != nil {
		return -1, err
	}
	count, err = redis.Int(c.Do("PUBLISH", this.settings.RedisChannel, data))
	return
}

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

func (this *Event) SetAttributeBool(key string, value bool) {
	this.Attributes = append(this.Attributes, &Attribute{
		Key:       &key,
		BoolValue: &value,
	})
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
