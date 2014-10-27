package rest

import (
	"github.com/bmizerany/assert"
	"testing"
)

var hooks = Webhooks{
	"service1": EventKeyUrlMap{
		"event1": Webhook{
			Url: "http://foo.com/bar/callback1",
		},
		"event2": Webhook{
			Url: "http://foo.com/bar/callback2",
		},
	},
	"service2": EventKeyUrlMap{
		"event1": Webhook{
			Url: "http://bar.com/bar/callback1",
		},
		"event2": Webhook{
			Url: "http://bar.com/bar/callback2",
		},
	},
}

type impl int

func (this *impl) Load() *Webhooks {
	return &hooks
}

func TestWebhookSerialization(t *testing.T) {

	bytes := hooks.ToJSON()
	assert.NotEqual(t, nil, bytes)
	assert.NotEqual(t, 0, len(bytes))

	hooks2 := Webhooks{}

	hooks2.FromJSON(bytes)

	assert.Equal(t, hooks, hooks2)

	t.Log("json", string(hooks2.ToJSON()))
}

func TestWebhooksManager(t *testing.T) {

}
