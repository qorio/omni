package passport

import (
	"github.com/qorio/omni/rest"
)

var DefaultWebHooks = rest.WebHooks{
	"test": rest.EventKeyUrlMap{
		"new-user-registration": rest.WebHook{
			Url: "http://localhost:9999/event/new-user-registration",
		},
	},
	"test2": rest.EventKeyUrlMap{
		"new-user-registration": rest.WebHook{
			Url: "http://localhost:9998/event/new-user-registration",
		},
	},
}
