package lighthouse

import (
	"github.com/qorio/api"
	"github.com/qorio/api/passport"
)

const (
	RegisterUser api.ServiceMethod = iota
	GetUserProfile
	AuthenticateUser

	AddOrUpdateBeacon
	ListAllBeacons
)

var Methods = api.ServiceMethods{

	RegisterUser: api.MethodSpec{
		AuthScope: "*",
		Doc: `
Registers a user
`,
		UrlRoute:     "/api/v1/register",
		HttpMethod:   "POST",
		ContentTypes: []string{"application/protobuf", "application/json"},
		RequestBody: func() interface{} {
			return passport.Login{}
		},
		ResponseBody: func() interface{} {
			return passport.Login{} // will include assigned id.
		},
	},

	AddOrUpdateBeacon: api.MethodSpec{
		AuthScope: "*",
		Doc: `
Create or update a beacon inventory entry
`,
		UrlRoute:     "/api/v1/beacon",
		HttpMethod:   "POST",
		ContentTypes: []string{"application/protobuf", "application/json"},
		RequestBody: func() interface{} {
			return Beacon{}
		},
		ResponseBody: func() interface{} {
			// Success response echos the input beacon summary data, with id populated.
			return Beacon{}
		},
	},

	ListAllBeacons: api.MethodSpec{
		AuthScope: "*",
		Doc: `
List all beacons
`,
		UrlRoute:     "/api/v1/beacon/",
		HttpMethod:   "GET",
		ContentTypes: []string{"application/protobuf", "application/json"},
		RequestBody:  nil,
		ResponseBody: func() interface{} {
			return []Beacon{}
		},
	},
}
