package shorty

import (
	"encoding/json"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	omni_auth "github.com/qorio/omni/auth"
	omni_common "github.com/qorio/omni/common"
	omni_http "github.com/qorio/omni/http"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

var accountIdKey string = "@accountId"

func (this *ShortyEndPoint) ApiAddCampaignHandler(credential *omni_auth.Info, resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	campaign := this.service.Campaign()
	dec := json.NewDecoder(strings.NewReader(string(body)))
	if err := dec.Decode(campaign); err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	if campaign.Id == "" {
		uuidStr, err := omni_common.NewUUID()
		if err != nil {
			renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
			return
		}
		campaign.Id = UUID(uuidStr)
	}

	campaign.AccountId = UUID(credential.GetString(accountIdKey))

	err = campaign.Save()
	if err != nil {
		renderJsonError(resp, req, "Failed to save campaign", http.StatusInternalServerError)
		return
	}

	buff, err := json.Marshal(campaign)
	if err != nil {
		renderJsonError(resp, req, "malformed-campaign", http.StatusInternalServerError)
		return
	}
	resp.Write(buff)
}

func (this *ShortyEndPoint) ApiGetCampaignHandler(credential *omni_auth.Info, resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)
	vars := mux.Vars(req)
	campaignId := vars["campaignId"]

	campaign, err := this.service.FindCampaign(UUID(campaignId))
	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	if campaign == nil {
		renderJsonError(resp, req, "campaign-not-found", http.StatusBadRequest)
		return
	}

	buff, err := json.Marshal(campaign)
	if err != nil {
		renderJsonError(resp, req, "malformed-campaign", http.StatusInternalServerError)
		return
	}
	resp.Write(buff)
}

func (this *ShortyEndPoint) ApiUpdateCampaignHandler(credential *omni_auth.Info, resp http.ResponseWriter, req *http.Request) {

	omni_http.SetCORSHeaders(resp)
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(req)
	campaignId := vars["campaignId"]
	campaign, err := this.service.FindCampaign(UUID(campaignId))
	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	if campaign == nil {
		renderJsonError(resp, req, "campaign-not-found", http.StatusBadRequest)
		return
	}

	campaign = this.service.Campaign() // new value from the post body
	dec := json.NewDecoder(strings.NewReader(string(body)))
	if err := dec.Decode(campaign); err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	if string(campaign.Id) != "" && string(campaign.Id) != campaignId {
		renderJsonError(resp, req, "id-mismatch", http.StatusBadRequest)
		return
	}

	campaign.Id = UUID(campaignId)
	err = campaign.Save()
	glog.Infoln("Saved ", campaign)
	if err != nil {
		renderJsonError(resp, req, "failed-to-save-campaign", http.StatusInternalServerError)
		return
	}

	buff, err := json.Marshal(campaign)
	if err != nil {
		renderJsonError(resp, req, "malformed-campaign", http.StatusInternalServerError)
		return
	}
	resp.Write(buff)
}

func (this *ShortyEndPoint) ApiAddCampaignUrlHandler(credential *omni_auth.Info, resp http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	omni_http.SetCORSHeaders(resp)
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	var message ShortyAddRequest
	dec := json.NewDecoder(strings.NewReader(string(body)))
	for {
		if err := dec.Decode(&message); err == io.EOF {
			break
		} else if err != nil {
			renderJsonError(resp, req, err.Error(), http.StatusBadRequest)
			return
		}
	}

	if message.LongUrl == "" {
		renderJsonError(resp, req, "No URL to shorten", http.StatusBadRequest)
		return
	}

	// Load the campaign
	campaignId := vars["campaignId"]
	campaign, err := this.service.FindCampaign(UUID(campaignId))

	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	} else if campaign == nil {
		renderJsonError(resp, req, "campaign-not-found", http.StatusBadRequest)
		return
	}

	mergedRules := make([]RoutingRule, len(campaign.Rules))
	if len(campaign.Rules) > 0 && len(message.Rules) > 0 {
		// apply the message's rules ON TOP of the campaign defaults
		// first index the override rules
		overrides := make(map[string][]byte)
		for _, r := range message.Rules {
			if r.Id != "" {
				if buf, err := json.Marshal(r); err == nil {
					overrides[r.Id] = buf
				}
			}
		}

		// Now iterate through the base and then apply the override on top of it
		for i, b := range campaign.Rules {
			mergedRules[i] = b
			if b.Id != "" {
				if v, exists := overrides[b.Id]; exists {
					merged := &RoutingRule{}
					*merged = b
					json.Unmarshal(v, merged)
					mergedRules[i] = *merged
				}
			}
		}
	} else {
		mergedRules = campaign.Rules
	}

	// Set the starting values, and the api will validate the rules and return a saved reference.
	shortUrl := &ShortUrl{
		Origin: message.Origin,

		// TODO - add lookup of api token to valid apiKey.
		// A api token is used by client as a way to authenticate and identify the actual app.
		// This way, we can revoke the token and shut down a client.
		AccountId: UUID(campaign.AccountId),

		// TODO - this is a key that references a future struct that encapsulates all the
		// rules around default routing (appstore, etc.).  This will simplify the api by not
		// requiring ios client to send in rules on android, for example.  The service should
		// check to see if there's valid campaign for the same app key. If yes, then merge the
		// routing rules.  If not, just let this value be a tag of some kind.
		CampaignId: campaign.Id,
	}
	if message.Vanity != "" {
		shortUrl, err = this.service.VanityUrl(message.Vanity, message.LongUrl, mergedRules, *shortUrl)
	} else {
		shortUrl, err = this.service.ShortUrl(message.LongUrl, mergedRules, *shortUrl)
	}

	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	if _, err := this.router.Get("redirect").URL("id", shortUrl.Id); err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	buff, err := json.Marshal(shortUrl)
	if err != nil {
		renderJsonError(resp, req, "Malformed short url rule", http.StatusInternalServerError)
		return
	}
	resp.Write(buff)
}

func (this *ShortyEndPoint) ApiAddUrlHandler(credential *omni_auth.Info, resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	var message ShortyAddRequest
	dec := json.NewDecoder(strings.NewReader(string(body)))
	for {
		if err := dec.Decode(&message); err == io.EOF {
			break
		} else if err != nil {
			renderJsonError(resp, req, err.Error(), http.StatusBadRequest)
			return
		}
	}

	if message.LongUrl == "" {
		renderJsonError(resp, req, "No URL to shorten", http.StatusBadRequest)
		return
	}

	// Set the starting values, and the api will validate the rules and return a saved reference.
	shortUrl := &ShortUrl{
		Origin: message.Origin,

		// TODO - add lookup of api token to valid apiKey.
		// A api token is used by client as a way to authenticate and identify the actual app.
		// This way, we can revoke the token and shut down a client.
		AccountId: UUID(credential.GetString(accountIdKey)),

		// TODO - this is a key that references a future struct that encapsulates all the
		// rules around default routing (appstore, etc.).  This will simplify the api by not
		// requiring ios client to send in rules on android, for example.  The service should
		// check to see if there's valid campaign for the same app key. If yes, then merge the
		// routing rules.  If not, just let this value be a tag of some kind.
		CampaignId: UUID(message.Campaign),
	}
	if message.Vanity != "" {
		shortUrl, err = this.service.VanityUrl(message.Vanity, message.LongUrl, message.Rules, *shortUrl)
	} else {
		shortUrl, err = this.service.ShortUrl(message.LongUrl, message.Rules, *shortUrl)
	}

	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	if _, err := this.router.Get("redirect").URL("id", shortUrl.Id); err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	buff, err := json.Marshal(shortUrl)
	if err != nil {
		renderJsonError(resp, req, "Malformed short url rule", http.StatusInternalServerError)
		return
	}
	resp.Write(buff)
}
