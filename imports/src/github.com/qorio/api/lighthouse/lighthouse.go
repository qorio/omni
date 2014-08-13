package lighthouse

import (
	"github.com/qorio/api"
)

const (
	AddOrUpdateBeacon api.ServiceMethod = iota
	ListAllBeacons
)

var Methods = map[api.ServiceMethod]*api.MethodSpec{

	AddOrUpdateBeacon: &api.MethodSpec{
		RequiresAuth: true,
		Doc: `
Create or update a beacon inventory entry
`,
		Name:         "AddOrUpdateBeacon",
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

	ListAllBeacons: &api.MethodSpec{
		RequiresAuth: true,
		Doc: `
List all beacons
`,
		Name:         "ListAllBeacons",
		UrlRoute:     "/api/v1/beacon/",
		HttpMethod:   "GET",
		ContentTypes: []string{"application/protobuf", "application/json"},
		RequestBody:  nil,
		ResponseBody: func() interface{} {
			return []Beacon{}
		},
	},
}
