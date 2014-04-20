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
	Countries Stats `json:"countries"`
	Browsers  Stats `json:"browsers"`
	OS        Stats `json:"os"`
	Referrers Stats `json:"referrers"`
}

const (
	keyy = "year:%d"
	keym = "month:%0d-%0.2d"
	keyd = "day:%d-%0.2d-%0.2d"
	keyh = "hour:%d-%0.2d-%0.2d %0.2d"
	keyi = "minute:%d-%0.2d-%0.2d %0.2d:%0.2d"
)

func (this *ShortUrl) Record(r *http.RequestOrigin) (err error) {
	c := this.service.pool.Get()
	defer c.Close()

	now := time.Now()
	year, month, day := now.Date()
	hour := now.Hour()
	minute := 5 * int(math.Abs(float64(now.Minute()/5)))
	prefix := this.service.settings.RedisPrefix + "stats:" + this.Id + ":"
	hitsPrefix := prefix + "hits:"
	countriesPrefix := prefix + "countries:"
	browsersPrefix := prefix + "browsers:"
	osPrefix := prefix + "os:"
	referrerPrefix := prefix + "referrers:"

	c.Send("INCR", hitsPrefix+"total")
	c.Send("INCR", fmt.Sprintf(hitsPrefix+keyy, year))
	c.Send("INCR", fmt.Sprintf(hitsPrefix+keym, year, month))
	c.Send("INCR", fmt.Sprintf(hitsPrefix+keyd, year, month, day))
	c.Send("INCR", fmt.Sprintf(hitsPrefix+keyh, year, month, day, hour))
	c.Send("INCR", fmt.Sprintf(hitsPrefix+keyi, year, month, day, hour, minute))

	if r.Country != "" {
		c.Send("INCR", countriesPrefix+"total:"+r.Country)
		c.Send("INCR", fmt.Sprintf(countriesPrefix+keyy+":"+r.Country, year))
		c.Send("INCR", fmt.Sprintf(countriesPrefix+keym+":"+r.Country, year, month))
		c.Send("INCR", fmt.Sprintf(countriesPrefix+keyd+":"+r.Country, year, month, day))
		c.Send("INCR", fmt.Sprintf(countriesPrefix+keyh+":"+r.Country, year, month, day, hour))
		c.Send("INCR", fmt.Sprintf(countriesPrefix+keyi+":"+r.Country, year, month, day, hour, minute))
	}

	if !r.Bot {
		c.Send("INCR", browsersPrefix+"total:"+r.Browser)
		c.Send("INCR", fmt.Sprintf(browsersPrefix+keyy+":"+r.Browser, year))
		c.Send("INCR", fmt.Sprintf(browsersPrefix+keym+":"+r.Browser, year, month))
		c.Send("INCR", fmt.Sprintf(browsersPrefix+keyd+":"+r.Browser, year, month, day))
		c.Send("INCR", fmt.Sprintf(browsersPrefix+keyh+":"+r.Browser, year, month, day, hour))
		c.Send("INCR", fmt.Sprintf(browsersPrefix+keyi+":"+r.Browser, year, month, day, hour, minute))

		c.Send("INCR", osPrefix+"total:"+r.OS)
		c.Send("INCR", fmt.Sprintf(osPrefix+keyy+":"+r.OS, year))
		c.Send("INCR", fmt.Sprintf(osPrefix+keym+":"+r.OS, year, month))
		c.Send("INCR", fmt.Sprintf(osPrefix+keyd+":"+r.OS, year, month, day))
		c.Send("INCR", fmt.Sprintf(osPrefix+keyh+":"+r.OS, year, month, day, hour))
		c.Send("INCR", fmt.Sprintf(osPrefix+keyi+":"+r.OS, year, month, day, hour, minute))
	}

	c.Send("INCR", referrerPrefix+"total:"+r.Referrer)
	c.Send("INCR", fmt.Sprintf(referrerPrefix+keyy+":"+r.Referrer, year))
	c.Send("INCR", fmt.Sprintf(referrerPrefix+keym+":"+r.Referrer, year, month))
	c.Send("INCR", fmt.Sprintf(referrerPrefix+keyd+":"+r.Referrer, year, month, day))
	c.Send("INCR", fmt.Sprintf(referrerPrefix+keyh+":"+r.Referrer, year, month, day, hour))
	c.Send("INCR", fmt.Sprintf(referrerPrefix+keyi+":"+r.Referrer, year, month, day, hour, minute))

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

func (this *ShortUrl) Countries(sorting bool) (Stats, error) {
	return this.keyStats(this.service.settings.RedisPrefix+"stats:"+this.Id+":countries:total:*", sorting)
}

func (this *ShortUrl) Browsers(sorting bool) (Stats, error) {
	return this.keyStats(this.service.settings.RedisPrefix+"stats:"+this.Id+":browsers:total:*", sorting)
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

	stats.OS, err = this.OS(sorting)
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
