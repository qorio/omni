package rest

import (
	"github.com/bmizerany/assert"
	"testing"
)

var hooks = WebHooks{
	"service1": EventKeyUrlMap{
		"event1": WebHook{
			Url: "http://foo.com/bar/callback1",
		},
		"event2": WebHook{
			Url: "http://foo.com/bar/callback2",
		},
	},
	"service2": EventKeyUrlMap{
		"event1": WebHook{
			Url: "http://bar.com/bar/callback1",
		},
		"event2": WebHook{
			Url: "http://bar.com/bar/callback2",
		},
	},
}

type impl int

func (this *impl) Load() *WebHooks {
	return &hooks
}

func TestWebHookSerialization(t *testing.T) {

	bytes := hooks.ToJSON()
	assert.NotEqual(t, nil, bytes)
	assert.NotEqual(t, 0, len(bytes))

	hooks2 := WebHooks{}

	hooks2.FromJSON(bytes)

	assert.Equal(t, hooks, hooks2)

	t.Log("json", string(hooks2.ToJSON()))
}

func TestWebHooksService(t *testing.T) {

}
