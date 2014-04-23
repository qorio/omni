package shorty

import (
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/qorio/omni/http"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Stat struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

type Stats []*Stat
type Descending Stats
type Format func(string) (string, error)

type OriginStats struct {
	Countries   Stats `json:"countries"`
	Regions     Stats `json:"regions"`
	Cities      Stats `json:"cities"`
	PostalCodes Stats `json:"postalcodes"`
	Browsers    Stats `json:"browsers"`
	Platforms   Stats `json:"platform"`
	OS          Stats `json:"os"`
	Referrers   Stats `json:"referrers"`
}

const (
	keyy = "year:%d"
	keym = "month:%0d-%0.2d"
	keyd = "day:%d-%0.2d-%0.2d"
	keyh = "hour:%d-%0.2d-%0.2d %0.2d"
	keyi = "minute:%d-%0.2d-%0.2d %0.2d:%0.2d"
)

func (this *ShortUrl) Record(r *http.RequestOrigin, isRevisit bool) (err error) {
	c := this.service.pool.Get()
	defer c.Close()

	now := time.Now()
	year, month, day := now.Date()
	hour := now.Hour()
	minute := 5 * int(math.Abs(float64(now.Minute()/5)))
	prefix := this.service.settings.RedisPrefix + "stats:" + this.Id + ":"

	hitsPrefix := prefix + "hits:"
	c.Send("INCR", hitsPrefix+"total")

	if isRevisit {
		c.Flush()
		return
	}

	// We only record the stats when we think the visit is not a revisit for this particular short url.

	c.Send("INCR", hitsPrefix+"uniques")

	countriesPrefix := prefix + "countries:"
	regionsPrefix := prefix + "regions:"
	citiesPrefix := prefix + "cities:"
	postalCodesPrefix := prefix + "postalcodes:"
	browsersPrefix := prefix + "browsers:"
	platformPrefix := prefix + "platform:"
	osPrefix := prefix + "os:"
	referrerPrefix := prefix + "referrers:"

	c.Send("INCR", fmt.Sprintf(hitsPrefix+keyy, year))
	c.Send("INCR", fmt.Sprintf(hitsPrefix+keym, year, month))
	c.Send("INCR", fmt.Sprintf(hitsPrefix+keyd, year, month, day))
	c.Send("INCR", fmt.Sprintf(hitsPrefix+keyh, year, month, day, hour))
	c.Send("INCR", fmt.Sprintf(hitsPrefix+keyi, year, month, day, hour, minute))

	if l := r.Location; l != nil {
		if l.CountryCode != "" {
			c.Send("INCR", countriesPrefix+"total:"+l.CountryCode)
			c.Send("INCR", fmt.Sprintf(countriesPrefix+keyy+":"+l.CountryCode, year))
			c.Send("INCR", fmt.Sprintf(countriesPrefix+keym+":"+l.CountryCode, year, month))
			c.Send("INCR", fmt.Sprintf(countriesPrefix+keyd+":"+l.CountryCode, year, month, day))
			c.Send("INCR", fmt.Sprintf(countriesPrefix+keyh+":"+l.CountryCode, year, month, day, hour))
			c.Send("INCR", fmt.Sprintf(countriesPrefix+keyi+":"+l.CountryCode, year, month, day, hour, minute))
		}
		if l.Region != "" {
			c.Send("INCR", regionsPrefix+"total:"+l.Region)
			c.Send("INCR", fmt.Sprintf(regionsPrefix+keyy+":"+l.Region, year))
			c.Send("INCR", fmt.Sprintf(regionsPrefix+keym+":"+l.Region, year, month))
			c.Send("INCR", fmt.Sprintf(regionsPrefix+keyd+":"+l.Region, year, month, day))
			c.Send("INCR", fmt.Sprintf(regionsPrefix+keyh+":"+l.Region, year, month, day, hour))
			c.Send("INCR", fmt.Sprintf(regionsPrefix+keyi+":"+l.Region, year, month, day, hour, minute))
		}
		if l.City != "" {
			c.Send("INCR", citiesPrefix+"total:"+l.City)
			c.Send("INCR", fmt.Sprintf(citiesPrefix+keyy+":"+l.City, year))
			c.Send("INCR", fmt.Sprintf(citiesPrefix+keym+":"+l.City, year, month))
			c.Send("INCR", fmt.Sprintf(citiesPrefix+keyd+":"+l.City, year, month, day))
			c.Send("INCR", fmt.Sprintf(citiesPrefix+keyh+":"+l.City, year, month, day, hour))
			c.Send("INCR", fmt.Sprintf(citiesPrefix+keyi+":"+l.City, year, month, day, hour, minute))
		}
		if l.PostalCode != "" {
			c.Send("INCR", postalCodesPrefix+"total:"+l.PostalCode)
			c.Send("INCR", fmt.Sprintf(postalCodesPrefix+keyy+":"+l.PostalCode, year))
			c.Send("INCR", fmt.Sprintf(postalCodesPrefix+keym+":"+l.PostalCode, year, month))
			c.Send("INCR", fmt.Sprintf(postalCodesPrefix+keyd+":"+l.PostalCode, year, month, day))
			c.Send("INCR", fmt.Sprintf(postalCodesPrefix+keyh+":"+l.PostalCode, year, month, day, hour))
			c.Send("INCR", fmt.Sprintf(postalCodesPrefix+keyi+":"+l.PostalCode, year, month, day, hour, minute))
		}
	}

	c.Send("INCR", referrerPrefix+"total:"+r.Referrer)
	c.Send("INCR", fmt.Sprintf(referrerPrefix+keyy+":"+r.Referrer, year))
	c.Send("INCR", fmt.Sprintf(referrerPrefix+keym+":"+r.Referrer, year, month))
	c.Send("INCR", fmt.Sprintf(referrerPrefix+keyd+":"+r.Referrer, year, month, day))
	c.Send("INCR", fmt.Sprintf(referrerPrefix+keyh+":"+r.Referrer, year, month, day, hour))
	c.Send("INCR", fmt.Sprintf(referrerPrefix+keyi+":"+r.Referrer, year, month, day, hour, minute))

	ua := r.UserAgent
	if !ua.Bot {
		c.Send("INCR", browsersPrefix+"total:"+ua.Browser)
		c.Send("INCR", fmt.Sprintf(browsersPrefix+keyy+":"+ua.Browser, year))
		c.Send("INCR", fmt.Sprintf(browsersPrefix+keym+":"+ua.Browser, year, month))
		c.Send("INCR", fmt.Sprintf(browsersPrefix+keyd+":"+ua.Browser, year, month, day))
		c.Send("INCR", fmt.Sprintf(browsersPrefix+keyh+":"+ua.Browser, year, month, day, hour))
		c.Send("INCR", fmt.Sprintf(browsersPrefix+keyi+":"+ua.Browser, year, month, day, hour, minute))

		c.Send("INCR", platformPrefix+"total:"+ua.Platform)
		c.Send("INCR", fmt.Sprintf(platformPrefix+keyy+":"+ua.Platform, year))
		c.Send("INCR", fmt.Sprintf(platformPrefix+keym+":"+ua.Platform, year, month))
		c.Send("INCR", fmt.Sprintf(platformPrefix+keyd+":"+ua.Platform, year, month, day))
		c.Send("INCR", fmt.Sprintf(platformPrefix+keyh+":"+ua.Platform, year, month, day, hour))
		c.Send("INCR", fmt.Sprintf(platformPrefix+keyi+":"+ua.Platform, year, month, day, hour, minute))

		c.Send("INCR", osPrefix+"total:"+ua.OS)
		c.Send("INCR", fmt.Sprintf(osPrefix+keyy+":"+ua.OS, year))
		c.Send("INCR", fmt.Sprintf(osPrefix+keym+":"+ua.OS, year, month))
		c.Send("INCR", fmt.Sprintf(osPrefix+keyd+":"+ua.OS, year, month, day))
		c.Send("INCR", fmt.Sprintf(osPrefix+keyh+":"+ua.OS, year, month, day, hour))
		c.Send("INCR", fmt.Sprintf(osPrefix+keyi+":"+ua.OS, year, month, day, hour, minute))
	}
	c.Flush()
	return
}

func (this *ShortUrl) Hits() (total int, err error) {
	c := this.service.pool.Get()
	defer c.Close()

	prefix := this.service.settings.RedisPrefix + "stats:" + this.Id + ":hits:"
	result, err := c.Do("GET", prefix+"total")
	if result == nil {
		return 0, nil
	}
	return redis.Int(result, err)
}

func (this *ShortUrl) Uniques() (total int, err error) {
	c := this.service.pool.Get()
	defer c.Close()

	prefix := this.service.settings.RedisPrefix + "stats:" + this.Id + ":hits:"
	result, err := c.Do("GET", prefix+"uniques")
	if result == nil {
		return 0, nil
	}
	return redis.Int(result, err)
}

func (this *ShortUrl) Countries(sorting bool) (Stats, error) {
	return this.keyStats(this.service.settings.RedisPrefix+"stats:"+this.Id+":countries:total:*", sorting)
}

func (this *ShortUrl) Regions(sorting bool) (Stats, error) {
	return this.keyStats(this.service.settings.RedisPrefix+"stats:"+this.Id+":regions:total:*", sorting)
}

func (this *ShortUrl) Cities(sorting bool) (Stats, error) {
	return this.keyStats(this.service.settings.RedisPrefix+"stats:"+this.Id+":cities:total:*", sorting)
}

func (this *ShortUrl) PostalCodes(sorting bool) (Stats, error) {
	return this.keyStats(this.service.settings.RedisPrefix+"stats:"+this.Id+":postalcodes:total:*", sorting)
}

func (this *ShortUrl) Browsers(sorting bool) (Stats, error) {
	return this.keyStats(this.service.settings.RedisPrefix+"stats:"+this.Id+":browsers:total:*", sorting)
}

func (this *ShortUrl) Platforms(sorting bool) (Stats, error) {
	return this.keyStats(this.service.settings.RedisPrefix+"stats:"+this.Id+":platform:total:*", sorting)
}

func (this *ShortUrl) OS(sorting bool) (Stats, error) {
	return this.keyStats(this.service.settings.RedisPrefix+"stats:"+this.Id+":os:total:*", sorting)
}

func (this *ShortUrl) Referrers(sorting bool) (Stats, error) {
	return this.keyStats(this.service.settings.RedisPrefix+"stats:"+this.Id+":referrers:total:*", sorting)
}

func (this *ShortUrl) Sources(sorting bool) (stats OriginStats, err error) {
	stats.Browsers, err = this.Browsers(sorting)
	if err != nil {
		return
	}

	stats.Countries, err = this.Countries(sorting)
	if err != nil {
		return
	}

	stats.Regions, err = this.Regions(sorting)
	if err != nil {
		return
	}

	stats.Cities, err = this.Cities(sorting)
	if err != nil {
		return
	}

	stats.PostalCodes, err = this.PostalCodes(sorting)
	if err != nil {
		return
	}

	stats.OS, err = this.OS(sorting)
	if err != nil {
		return
	}

	stats.Platforms, err = this.Platforms(sorting)
	if err != nil {
		return
	}

	stats.Referrers, err = this.Referrers(sorting)
	if err != nil {
		return
	}

	return
}

func (this *ShortUrl) Stats(past string) (stats Stats, err error) {
	c := this.service.pool.Get()
	defer c.Close()

	now := time.Now()
	year, month, day := now.Date()
	prefix := this.service.settings.RedisPrefix + "stats:" + this.Id + ":hits:"

	var (
		separator string
		search    string
		moment    int
		start     int
		limit     int
		increment int
	)

	start = 1
	increment = 1
	format := func(value string) (string, error) {
		return value, nil
	}
	switch {
	case past == "hour":
		search = prefix + keyi
		separator = " %d:"
		moment = now.Hour()
		search = fmt.Sprintf(search[0:strings.LastIndex(search, ":")]+":*", year, month, day, moment)
		start = 0
		limit = 60
		increment = 5
		format = func(value string) (string, error) {
			return fmt.Sprintf("%0.2d:%s", moment, value), nil
		}
	case past == "day":
		search = prefix + keyh
		separator = "-%d"
		moment = day
		search = fmt.Sprintf(search[0:strings.LastIndex(search, " ")]+" *", year, month, moment)
		limit = 24
		format = func(value string) (string, error) {
			return fmt.Sprintf("%s:00", value), nil
		}
	case past == "week":
		search = prefix + keyd
		separator = "%d-"
		moment = int(month)
		search = fmt.Sprintf(search[0:strings.LastIndex(search, "-%0.2d")]+"-*", year, moment)
		if int(now.Weekday()) == 0 {
			start = day - 6
		} else {
			start = day - (int(now.Weekday()) - 1)
		}
		limit = start + 7
		format = func(value string) (string, error) {
			date := fmt.Sprintf("%d-%0.2d-%s", year, month, value)
			time, err := time.Parse("2006-01-02", date)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%s %s", time.Weekday().String(), value), nil
		}
	case past == "month":
		search = prefix + keyd
		separator = "%d-"
		moment = int(month)
		search = fmt.Sprintf(search[0:strings.LastIndex(search, "-%0.2d")]+"-*", year, moment)
		limit = time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
		format = func(value string) (string, error) {
			date := fmt.Sprintf("%d-%0.2d-%s", year, month, value)
			time, err := time.Parse("2006-01-02", date)
			if err != nil {
				return "", err
			}
			month, _ := strconv.ParseInt(value, 10, 32)
			return fmt.Sprintf("%s %d", time.Month().String(), month), nil
		}
	case past == "year":
		search = prefix + keym
		separator = "%d-"
		moment = year
		search = fmt.Sprintf(search[0:strings.LastIndex(search, "-%0.2d")]+"-*", moment)
		limit = 12
		format = func(value string) (string, error) {
			date := fmt.Sprintf("%d-%s-01", year, value)
			time, err := time.Parse("2006-01-02", date)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%s %d", time.Month().String(), year), nil
		}
	case past == "all":
		search = prefix + keyy
		separator = ":"
		search = search[0:strings.LastIndex(search, "%d")] + "*"
		start = now.Year() - 10
		limit = int(now.Year()) + 1
	default:
		return nil, errors.New(fmt.Sprintf("Invalid stat requested: %s", past))
	}

	stats, err = getStats(c, search, separator, moment, start, limit, increment, format)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

func (this *ShortUrl) keyStats(search string, sorting bool) (stats Stats, err error) {
	c := this.service.pool.Get()
	defer c.Close()

	values, err := redis.Values(c.Do("KEYS", search))
	if err != nil {
		return nil, err
	} else if len(values) == 0 {
		return nil, nil
	}

	keys := make([]interface{}, len(values))
	i := 0
	for _, value := range values {
		key, err := redis.String(value, nil)
		if err == nil {
			keys[i] = key
			i++
		}
	}

	values, err = redis.Values(c.Do("MGET", keys...))
	if err != nil {
		return nil, err
	}

	stats = make(Stats, len(values))

	for i, value := range values {
		key := keys[i].(string)
		total, err := redis.Int(value, nil)
		if err == nil {
			stats[i] = &Stat{Name: key[strings.LastIndex(key, ":")+1:], Value: total}
		}
	}

	if sorting {
		sort.Sort(stats)
	}

	return stats, nil
}

func getStats(c redis.Conn, search string, separator string, moment int, start int, limit int, increment int, format Format) (Stats, error) {
	length := int(math.Ceil(float64((limit - start) / increment)))
	stats := make(Stats, length)

	redisKeys := make([]interface{}, length)
	j := 0

	prefix := search[:strings.LastIndex(search, "*")]
	for i := start; i < limit; i += increment {
		key := fmt.Sprintf("%0.2d", i)
		name, err := format(key)
		if err != nil {
			name = key
		}

		redisKeys[j] = prefix + key
		stats[j] = &Stat{
			Name:  name,
			Value: 0,
		}
		j++
	}

	values, err := redis.Values(c.Do("MGET", redisKeys...))
	if err != nil {
		return stats, err
	}

	if strings.Index(separator, "%d") >= 0 {
		separator = fmt.Sprintf(separator, moment)
	}

	for i, value := range values {
		total, err := redis.Int(value, nil)
		if err == nil {
			stats[i].Value = total
		}
	}

	return stats, nil
}

func (s Stats) Len() int {
	return len(s)
}

func (s Stats) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s Stats) Less(i, j int) bool {
	return s[i].Value >= s[j].Value
}
