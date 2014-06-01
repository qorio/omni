package shorty

import (
	"bytes"
	"crypto/rand"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/golang/glog"
	"github.com/qorio/omni/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type shortyImpl struct {
	settings            Settings
	pool                *redis.Pool
	decodeEventChannel  chan *DecodeEvent
	installEventChannel chan *InstallEvent
	linkEventChannel    chan *LinkEvent
	appOpenEventChannel chan *AppOpenEvent
}

const (
	alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

func Init(settings Settings) *shortyImpl {
	return &shortyImpl{
		settings: settings,
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

func (this *shortyImpl) Close() {
	if err := this.pool.Close(); err != nil {
		glog.Warningln("error-closing-connection-pool", err)
	}
}

func (this *shortyImpl) PublishDecode(event *DecodeEvent) {
	if this.decodeEventChannel != nil {
		this.decodeEventChannel <- event
	}
}

func (this *shortyImpl) PublishInstall(event *InstallEvent) {
	if this.installEventChannel != nil {
		this.installEventChannel <- event
	}
}

func (this *shortyImpl) PublishAppOpen(event *AppOpenEvent) {
	if this.appOpenEventChannel != nil {
		this.appOpenEventChannel <- event
	}
}

func (this *shortyImpl) PublishLink(event *LinkEvent) {
	if this.linkEventChannel != nil {
		this.linkEventChannel <- event
	}
}

func (this *shortyImpl) DecodeEventChannel() <-chan *DecodeEvent {
	if this.decodeEventChannel == nil {
		this.decodeEventChannel = make(chan *DecodeEvent)
	}
	return this.decodeEventChannel
}

func (this *shortyImpl) LinkEventChannel() <-chan *LinkEvent {
	if this.linkEventChannel == nil {
		this.linkEventChannel = make(chan *LinkEvent)
	}
	return this.linkEventChannel
}

func (this *shortyImpl) InstallEventChannel() <-chan *InstallEvent {
	if this.installEventChannel == nil {
		this.installEventChannel = make(chan *InstallEvent)
	}
	return this.installEventChannel
}

func (this *shortyImpl) AppOpenEventChannel() <-chan *AppOpenEvent {
	if this.appOpenEventChannel == nil {
		this.appOpenEventChannel = make(chan *AppOpenEvent)
	}
	return this.appOpenEventChannel
}

func (this *shortyImpl) UrlLength() int {
	return this.settings.UrlLength
}

func (this *shortyImpl) ShortUrl(data string, rules []RoutingRule, defaults ShortUrl) (entity *ShortUrl, err error) {
	return this.VanityUrl("", data, rules, defaults)
}

func (this *shortyImpl) VanityUrl(vanity, data string, rules []RoutingRule, defaults ShortUrl) (entity *ShortUrl, err error) {
	data = strings.TrimSpace(data)
	if len(data) == 0 {
		err = errors.New("Please specify an URL")
		return
	}

	u, err := url.Parse(data)
	if err != nil {
		return
	}

	if len(this.settings.RestrictDomain) > 0 {
		if matches, _ := regexp.MatchString("^[A-Za-z0-9.]*"+this.settings.RestrictDomain, u.Host); !matches {
			err = errors.New("Only URLs on " + this.settings.RestrictDomain + " domain allowed")
			return
		}
	}

	entity = &ShortUrl{Destination: u.String(), Created: time.Now(), service: this}
	for _, rule := range rules {
		err = rule.Validate()
		if err != nil {
			return
		}
		for _, sub := range rule.Special {
			err = sub.Validate()
			if err != nil {
				return
			}
		}
	}

	// Each RoutingRule supports overrides via the 'Special' field.  If these are specified,
	// compute the merged view for each of the special rule.  This is done by first copying
	// the fields in the top level rule and then overlay the fields in the special/override version.
	// We use JSON encode and decode to merge the fields in the structs instead of struct copy by value.
	processed := make([]RoutingRule, len(rules))
	for i, rule := range rules {
		// if there are subrules, merge parent attributes into them
		// use json.  We do this to avoid doing this at serving time.
		r := &RoutingRule{}
		*r = rule
		r.Special = []RoutingRule{} // empty it out
		for j, sub := range rule.Special {
			if buf, err := json.Marshal(sub); err == nil {
				// Copy the child's fields into a copy of the parent to achieve 'overrides'
				json.Unmarshal(buf, r)
				rule.Special[j] = *r
			}
		}
		processed[i] = rule
	}

	entity.Rules = processed

	c := this.pool.Get()
	defer c.Close()

	if vanity != "" {
		if exists, _ := redis.Bool(c.Do("EXISTS", this.settings.RedisPrefix+"url:"+vanity)); !exists {
			entity.Id = vanity
		}
		if exists, _ := this.Find(vanity); exists == nil {
			entity.Id = vanity
		}
	} else {
		bytes := make([]byte, this.settings.UrlLength)
		// TODO - probably should add a limit so we don't spend forever exploring the number space
		// Not adding it because right now the number space is large, don't expect collisions.
		for {
			rand.Read(bytes)
			for i, b := range bytes {
				bytes[i] = alphanum[b%byte(len(alphanum))]
			}
			id := string(bytes)
			if exists, _ := redis.Bool(c.Do("EXISTS", this.settings.RedisPrefix+"url:"+id)); !exists {
				entity.Id = id
				break
			}

			if exists, _ := this.Find(id); exists == nil {
				entity.Id = id
				break
			}
		}
	}

	if entity.Id != "" {
		// copy the defaults
		entity.Origin = defaults.Origin
		entity.AppKey = defaults.AppKey
		entity.CampaignKey = defaults.CampaignKey

		entity.Save()
		return entity, nil
	} else if vanity != "" {
		return nil, errors.New("Vanity code taken:" + vanity)
	} else {
		return nil, errors.New("Failed to assign code")
	}
}

func (this *shortyImpl) Link(appUrlScheme UrlScheme, prevContext, currentContext UUID, shortUrlId string) error {
	c := this.pool.Get()
	defer c.Close()

	// The key allows searching A:B or B:A by making it A:B:A
	key := fmt.Sprintf("%s:%s:%s:%s:%s", appUrlScheme, shortUrlId, prevContext, currentContext, prevContext)
	reply, err := c.Do("SET", this.settings.RedisPrefix+"uuid-pair:"+key, shortUrlId)
	if err == nil && reply != "OK" {
		err = errors.New("Invalid Redis response")
	}
	return err
}

func (this *shortyImpl) FindLink(appUuid, uuid UUID) (found bool, err error) {
	c := this.pool.Get()
	defer c.Close()

	// The key allows searching A:B or B:A by making it A:B:A
	wildcard := fmt.Sprintf("*%s:%s*", appUuid, uuid)
	reply, err := redis.String(c.Do("GET", this.settings.RedisPrefix+"uuid-pair:"+wildcard))
	found = err == nil && reply != ""
	return
}

func (this *shortyImpl) SaveFingerprintedVisit(visit *FingerprintedVisit) error {
	c := this.pool.Get()
	defer c.Close()

	value, err := json.Marshal(visit)
	if err != nil {
		return err
	}
	reply, err := c.Do("SET", this.settings.RedisPrefix+"fingerprint:"+visit.Fingerprint, value)
	if err == nil && reply != "OK" {
		err = errors.New("Invalid Redis response")
	}
	return err
}

func (this *shortyImpl) MatchFingerPrint(fingerprint string) (score float64, visit *FingerprintedVisit, err error) {
	c := this.pool.Get()
	defer c.Close()

	q := http.GetFingerPrintMatchQuery(fingerprint)
	reply, err := redis.Strings(c.Do("KEYS", this.settings.RedisPrefix+"fingerprint:"+q))
	if err != nil {
		glog.Warningln("Error", err)
		return
	}
	glog.Infoln("matching fingerprint: fingerprint=", fingerprint, "q=", q, "result=", reply)

	// need to remove the prefix before matching
	candidates := make([]string, len(reply))
	for i, v := range reply {
		candidates[i] = strings.Split(v, this.settings.RedisPrefix+"fingerprint:")[1]
	}
	match, matchScore := http.MatchFingerPrint(fingerprint, candidates)
	glog.Infoln("matching fingerprint: match=", match, "score=", score)

	value, err2 := redis.Bytes(c.Do("GET", this.settings.RedisPrefix+"fingerprint:"+match))
	if err2 != nil {
		glog.Warningln("Error", err2)
	}
	if err3 := json.Unmarshal(value, &visit); err3 == nil {
		score = matchScore
		return
	}

	return
}

func (this *shortyImpl) TrackInstall(app UrlScheme, context UUID) error {
	c := this.pool.Get()
	defer c.Close()

	// TODO - look at every pair that contains this uuid, get the other uuid and create
	// a record as well. This will speed up the read when decoding the shortlink to see if installed.
	key := fmt.Sprintf("%s:%s", context, app)
	reply, err := c.Do("SET", this.settings.RedisPrefix+"install:"+key, timestamp())
	if err == nil && reply != "OK" {
		err = errors.New("Invalid Redis response")
	}
	return err
}

func (this *shortyImpl) TrackAppOpen(app UrlScheme, appContext UUID, appOpen *AppOpen) error {
	c := this.pool.Get()
	defer c.Close()

	key := fmt.Sprintf("%s:%s:%s:%s:%s", app, appContext, appOpen.SourceContext, appOpen.ShortCode, appOpen.SourceApplication)

	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(appOpen)
	if err != nil {
		return err
	}
	reply, err := c.Do("SET", this.settings.RedisPrefix+"app-open:"+key, buff.Bytes())
	if err == nil && reply != "OK" {
		err = errors.New("Invalid Redis response")
	}
	return err
}

func (this *shortyImpl) FindAppOpen(app UrlScheme, sourceContext UUID) (appOpen *AppOpen, found bool, err error) {
	c := this.pool.Get()
	defer c.Close()

	key := fmt.Sprintf("%s:*:%s:*", app, sourceContext)
	reply, err := redis.Strings(c.Do("KEYS", this.settings.RedisPrefix+"app-open:"+key))
	if err == nil && len(reply) > 0 {
		// Do a get on the first hit
		value, err := redis.Bytes(c.Do("GET", reply[0]))
		if err == nil {
			buff := bytes.NewBuffer(value)
			dec := gob.NewDecoder(buff)
			err = dec.Decode(&appOpen)
			found = err == nil
		}
	}
	return
}

func (this *shortyImpl) FindInstall(app UrlScheme, context UUID) (expiration int64, found bool, err error) {
	c := this.pool.Get()
	defer c.Close()

	key := fmt.Sprintf("%s:%s", context, app)
	reply, err := c.Do("GET", this.settings.RedisPrefix+"install:"+key)
	found = reply != nil

	if found {
		expiration, err = redis.Int64(reply, err)
	}
	return
}

func (this *shortyImpl) Find(id string) (*ShortUrl, error) {
	c := this.pool.Get()
	defer c.Close()

	reply, err := c.Do("GET", this.settings.RedisPrefix+"url:"+id)
	if reply == nil {
		return nil, nil
	}

	data, err := redis.Bytes(reply, err)
	if err != nil {
		return nil, err
	}

	url := ShortUrl{service: this}
	if err := json.Unmarshal(data, &url); err != nil {
		return nil, err
	}

	return &url, nil
}

func (this *ShortUrl) Save() error {
	c := this.service.pool.Get()
	defer c.Close()

	data, err := json.Marshal(this)
	if err != nil {
		return err
	}

	reply, err := c.Do("SET", this.service.settings.RedisPrefix+"url:"+this.Id, data)
	if err == nil && reply != "OK" {
		err = errors.New("Invalid Redis response")
	}

	if err != nil {
		return err
	}

	return nil
}

func (this *ShortUrl) Delete() error {
	c := this.service.pool.Get()
	defer c.Close()

	reply, err := c.Do("DEL", this.service.settings.RedisPrefix+"url:"+this.Id)
	if err == nil && reply != "OK" {
		return errors.New("Invalid Redis response")
	}

	return nil
}

func timestamp() int64 {
	return time.Now().Unix()
}
