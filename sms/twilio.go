package sms

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/golang/glog"
	"io/ioutil"
	"net/http"
	"net/url"
	"text/template"
	"time"
)

/*
curl -X POST 'https://api.twilio.com/2010-04-01/Accounts/AC30385d2f8be9e1f9442b1d4ba646a143/Messages.json' \
--data-urlencode 'To=4155097294'  \
--data-urlencode 'From=+14154134134'  \
--data-urlencode 'Body=This is a test' \
-u AC30385d2f8be9e1f9442b1d4ba646a143:[AuthToken]

AuthToken=a2675a4266b1f27ebd0cd528fc136227

{
  "sid": "SMdea543d18046413c95b13cd6710ce910",
  "date_created": "Thu, 05 Mar 2015 06:38:45 +0000",
  "date_updated": "Thu, 05 Mar 2015 06:38:45 +0000",
  "date_sent": null,
  "account_sid": "AC30385d2f8be9e1f9442b1d4ba646a143",
  "to": "+14155097294",
  "from": "+14154134134",
  "body": "This is a test",
  "status": "queued",
  "num_segments": "1",
  "num_media": "0",
  "direction": "outbound-api",
  "api_version": "2010-04-01",
  "price": null,
  "price_unit": "USD",
  "error_code": null,
  "error_message": null,
  "uri": "/2010-04-01/Accounts/AC30385d2f8be9e1f9442b1d4ba646a143/Messages/SMdea543d18046413c95b13cd6710ce910.json",
  "subresource_uris": {
    "media": "/2010-04-01/Accounts/AC30385d2f8be9e1f9442b1d4ba646a143/Messages/SMdea543d18046413c95b13cd6710ce910/Media.json"
  }
}

*/
const (
	TwilioMessageEndpoint = "https://api.twilio.com/2010-04-01/Accounts/{{.AccountSid}}/Messages.json"
)

var (
	TwilioEndpointTemplate *template.Template
)

func init() {
	TwilioEndpointTemplate = template.Must(template.New("twilio-endpoint").Parse(TwilioMessageEndpoint))
}

type Twilio struct {
	AccountSid string `json:"account_sid"`
	AuthToken  string `json:"auth_token"`

	FromPhone    func(*Message) Phone
	BodyTemplate func(*Message) string
}

type Phone string

type Message struct {
	To      Phone       `json:"to"`
	Context interface{} `json:"context"` // For apply template
}

type Response struct {
	Id           string    `json:"id"`
	Created      time.Time `json:"date_created"`
	Updated      time.Time `json:"date_updated"`
	Sent         time.Time `json:"date_sent"`
	To           Phone     `json:"to"`
	From         Phone     `json:"from"`
	Body         string    `json:"body"`
	Status       string    `json:"status"`
	ErrorCode    string    `json:"error_code"`
	ErrorMessage string    `json:"error_message"`
	Uri          string    `json:"uri"`
}

var (
	ErrNoFromPhone    = errors.New("error-from-missing")
	ErrNoToPhone      = errors.New("error-to-missing")
	ErrNoBodyTemplate = errors.New("error-no-body-template")
)

func (p Phone) IsZero() bool {
	return "" == string(p)
}

func (t *Twilio) SendMessage(message *Message) (*Response, error) {
	if t.BodyTemplate == nil {
		return nil, ErrNoBodyTemplate
	}
	if t.FromPhone == nil {
		return nil, ErrNoFromPhone
	}
	if message.To.IsZero() {
		return nil, ErrNoToPhone
	}
	bt, err := template.New("sms-body").Parse(t.BodyTemplate(message))
	if err != nil {
		return nil, err
	}
	var buff bytes.Buffer
	err = bt.Execute(&buff, message.Context)
	if err != nil {
		return nil, err
	}

	var endpoint bytes.Buffer
	TwilioEndpointTemplate.Execute(&endpoint, t)

	client := &http.Client{}

	data := url.Values{}
	data.Set("From", string(t.FromPhone(message)))
	data.Set("To", string(message.To))
	data.Set("Body", buff.String())
	post, err := http.NewRequest("POST", endpoint.String(), bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, err
	}
	post.SetBasicAuth(t.AccountSid, t.AuthToken)
	post.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(post)
	if err != nil {
		return nil, err
	}
	content, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	glog.Infoln("Got response:", string(content))

	r := new(Response)
	err = json.Unmarshal(content, r)
	return r, err
}
