package shorty

import (
	"github.com/golang/glog"
	"github.com/qorio/omni/http"
	"regexp"
	"strconv"
)

func (this *RoutingRule) Match(service Shorty, ua *http.UserAgent, origin *http.RequestOrigin, cookies http.Cookies) bool {
	// use bit mask to match
	var actual, expect int = 0, 0

	if len(this.MatchPlatform) > 0 {
		expect |= 1 << 0
		if matches, _ := regexp.MatchString(this.MatchPlatform, ua.Platform); matches {
			actual |= 1 << 0
		}
	}
	if len(this.MatchOS) > 0 {
		expect |= 1 << 1
		if matches, _ := regexp.MatchString(this.MatchOS, ua.OS); matches {
			actual |= 1 << 1
		}
	}
	if len(this.MatchMake) > 0 {
		expect |= 1 << 2
		if matches, _ := regexp.MatchString(this.MatchMake, ua.Make); matches {
			actual |= 1 << 2
		}
	}
	if len(this.MatchBrowser) > 0 {
		expect |= 1 << 3
		if matches, _ := regexp.MatchString(this.MatchBrowser, ua.Browser); matches {
			actual |= 1 << 3
		}
	}
	if len(this.MatchMobile) > 0 {
		expect |= 1 << 4
		if matches, _ := regexp.MatchString(this.MatchMobile, strconv.FormatBool(ua.Mobile)); matches {
			actual |= 1 << 4
		}
	}
	if len(this.MatchReferrer) > 0 {
		expect |= 1 << 5
		if matches, _ := regexp.MatchString(this.MatchReferrer, origin.Referrer); matches {
			actual |= 1 << 5
		}
	}
	if len(this.MatchInstalled) > 0 {
		expect |= 1 << 6
		if this.AppUrlScheme != "" {
			uuid, _ := cookies.GetPlainString(uuidCookieKey)
			_, found, _ := service.FindInstall(UrlScheme(this.AppUrlScheme), UUID(uuid))
			glog.Infoln("checking install", uuid, this.AppUrlScheme, found)
			if matches, _ := regexp.MatchString(this.MatchInstalled, strconv.FormatBool(found)); matches {
				actual |= 1 << 6
			}
		}
	}
	// By the time we get here, we have done a match all
	return actual == expect && expect > 0
}
