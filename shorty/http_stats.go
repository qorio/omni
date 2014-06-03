package shorty

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	omni_http "github.com/qorio/omni/http"
	"math"
	"net/http"
	"time"
)

type StatsSummary struct {
	Id      string      `json:"id"`
	Created string      `json:"when"`
	Hits    int         `json:"hits"`
	Uniques int         `json:"uniques"`
	Summary OriginStats `json:"summary"`
	Config  ShortUrl    `json:"config"`
}

func (this *ShortyEndPoint) StatsHandler(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)

	vars := mux.Vars(req)
	shortyUrl, err := this.service.FindUrl(vars["id"])
	if err != nil {
		renderError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	} else if shortyUrl == nil {
		renderError(resp, req, "No URL was found with short code", http.StatusNotFound)
		return
	}

	hits, err := shortyUrl.Hits()
	if err != nil {
		renderError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	uniques, err := shortyUrl.Uniques()
	if err != nil {
		renderError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	originStats, err := shortyUrl.Sources(true)
	if err != nil {
		renderError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	summary := StatsSummary{
		Id:      shortyUrl.Id,
		Created: relativeTime(time.Now().Sub(shortyUrl.Created)),
		Hits:    hits,
		Uniques: uniques,
		Summary: originStats,
		Config:  *shortyUrl,
	}

	buff, err := json.Marshal(summary)
	if err != nil {
		renderJsonError(resp, req, "Malformed summary", http.StatusInternalServerError)
		return
	}
	resp.Write(buff)
}

func relativeTime(duration time.Duration) string {
	hours := int64(math.Abs(duration.Hours()))
	minutes := int64(math.Abs(duration.Minutes()))
	when := ""
	switch {
	case hours >= (365 * 24):
		when = "Over an year ago"
	case hours > (30 * 24):
		when = fmt.Sprintf("%d months ago", int64(hours/(30*24)))
	case hours == (30 * 24):
		when = "a month ago"
	case hours > 24:
		when = fmt.Sprintf("%d days ago", int64(hours/24))
	case hours == 24:
		when = "yesterday"
	case hours >= 2:
		when = fmt.Sprintf("%d hours ago", hours)
	case hours > 1:
		when = "over an hour ago"
	case hours == 1:
		when = "an hour ago"
	case minutes >= 2:
		when = fmt.Sprintf("%d minutes ago", minutes)
	case minutes > 1:
		when = "a minute ago"
	default:
		when = "just now"
	}
	return when
}
