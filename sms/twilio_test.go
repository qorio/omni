package sms

import (
	. "gopkg.in/check.v1"
	"testing"
)

func TestSmsTests(t *testing.T) { TestingT(t) }

type SmsTests struct {
}

var _ = Suite(&SmsTests{})

// Database set up for circle_ci:
// psql> create role ubuntu login password 'password';
// psql> create database circle_ci with owner ubuntu encoding 'UTF8';
func (suite *SmsTests) SetUpSuite(c *C) {
}

func (suite *SmsTests) TearDownSuite(c *C) {
}

func (suite *SmsTests) TestSendMessage(c *C) {
	t := &Twilio{
		TwilioAccount: TwilioAccount{
			AccountSid: "AC30385d2f8be9e1f9442b1d4ba646a143",
			AuthToken:  "a2675a4266b1f27ebd0cd528fc136227",
		},
		From: func(*Message) Phone {
			return Phone("14154134134")
		},
		BodyTemplate: func(*Message) string {
			return "Hello {{.Username}}, here is the link where you can reset your password: {{.Link}}."
		},
	}

	resp, err := t.Send(&Message{
		To: Phone("4155097294"),
		Context: map[string]string{
			"Username": "David",
			"Link":     "http://yahoo.com",
		},
	})

	c.Log("Sent message", *resp, err)
	c.Assert(err, Equals, nil)

}
