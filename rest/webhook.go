package rest

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/golang/glog"
	"net/http"
	"net/url"
	"text/template"
)

var (
	ERROR_NO_SERVICE_DEFINED = errors.New("no-service-defined")
	ERROR_NO_WEBHOOK_DEFINED = errors.New("no-webhook-defined")

	WebHookHmacHeader = "X-Passport-Hmac"
)

// Webhook callbacks
type EventKeyUrlMap map[string]WebHook
type WebHook struct {
	Url string `json:"destination_url"`
}
type WebHooks map[string]EventKeyUrlMap

type WebHooksService interface {
	Send(string, string, interface{}, string) error
	RegisterWebHooks(string, EventKeyUrlMap) error
	RemoveWebHooks(string) error
}

func (this *WebHooks) ToJSON() []byte {
	if buff, err := json.Marshal(this); err == nil {
		return buff
	}
	return nil
}

func (this *WebHooks) FromJSON(s []byte) error {
	dec := json.NewDecoder(bytes.NewBuffer(s))
	return dec.Decode(this)
}

func (this *EventKeyUrlMap) ToJSON() []byte {
	if buff, err := json.Marshal(this); err == nil {
		return buff
	}
	return nil
}

func (this *EventKeyUrlMap) FromJSON(s []byte) error {
	dec := json.NewDecoder(bytes.NewBuffer(s))
	return dec.Decode(this)
}

// Default in-memory implementation
func (this WebHooks) Send(serviceKey, eventKey string, message interface{}, templateString string) error {
	m := this[serviceKey]
	if m == nil {
		return ERROR_NO_SERVICE_DEFINED
	}
	hook, has := m[eventKey]
	if !has {
		return ERROR_NO_WEBHOOK_DEFINED
	}
	return hook.Send(message, templateString)
}

func (this WebHooks) RegisterWebHooks(serviceKey string, ekum EventKeyUrlMap) error {
	this[serviceKey] = ekum
	return nil
}

func (this WebHooks) RemoveWebHooks(serviceKey string) error {
	delete(this, serviceKey)
	return nil
}

func (hook *WebHook) Send(message interface{}, templateString string) error {
	url, err := url.Parse(hook.Url)
	if err != nil {
		return err
	}

	go func() {
		glog.Infoln("Sending callback to", url)

		var buffer bytes.Buffer
		if templateString != "" {
			t := template.Must(template.New(templateString).Parse(templateString))
			err := t.Execute(&buffer, message)
			if err != nil {
				glog.Warningln("Cannot build payload for event", message)
				return
			}
		} else {
			glog.Infoln("no-payload", url)
		}
		// Determine where to send the event.
		client := &http.Client{}
		post, err := http.NewRequest("POST", url.String(), &buffer)
		post.Header.Add(WebHookHmacHeader, "TO DO: compute a HMAC here")
		resp, err := client.Do(post)
		if err != nil {
			glog.Warningln("Cannot deliver callback to", url, "error:", err)
		} else {
			glog.Infoln("Sent callback to ", url, "response=", resp)
		}
	}()
	return nil

}
