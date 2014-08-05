// Code generated by protoc-gen-go.
// source: lighthouse.proto
// DO NOT EDIT!

/*
Package lighthouse is a generated protocol buffer package.

It is generated from these files:
	lighthouse.proto

It has these top-level messages:
	Location
	Content
	UserProfile
	UserRef
	BeaconAdvertisement
	BeaconDeviceProfile
	BeaconSummary
	Beacon
	Acl
	Post
*/
package lighthouse

import proto "code.google.com/p/goprotobuf/proto"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = math.Inf

// To be transformed to GeoJson - ex)  {"location" : [-71.34, 41.12]}
// See http://www.elasticsearch.org/guide/en/elasticsearch/reference/current/mapping-geo-point-type.html
type Location struct {
	Lon              *float64 `protobuf:"fixed64,1,req,name=lon" json:"lon,omitempty"`
	Lat              *float64 `protobuf:"fixed64,2,req,name=lat" json:"lat,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (m *Location) Reset()         { *m = Location{} }
func (m *Location) String() string { return proto.CompactTextString(m) }
func (*Location) ProtoMessage()    {}

func (m *Location) GetLon() float64 {
	if m != nil && m.Lon != nil {
		return *m.Lon
	}
	return 0
}

func (m *Location) GetLat() float64 {
	if m != nil && m.Lat != nil {
		return *m.Lat
	}
	return 0
}

type Content struct {
	// Id - not required at creation time; assigned by server
	Uuid []byte `protobuf:"bytes,1,opt,name=uuid" json:"uuid,omitempty"`
	// MIME type. e.g. image/jpeg, image/png, video/mp4
	Type *string `protobuf:"bytes,2,req,name=type" json:"type,omitempty"`
	// Content data bytes
	Data []byte `protobuf:"bytes,3,opt,name=data" json:"data,omitempty"`
	// Or url as content - link sharing or content in cdn.
	Url              *string `protobuf:"bytes,4,opt,name=url" json:"url,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *Content) Reset()         { *m = Content{} }
func (m *Content) String() string { return proto.CompactTextString(m) }
func (*Content) ProtoMessage()    {}

func (m *Content) GetUuid() []byte {
	if m != nil {
		return m.Uuid
	}
	return nil
}

func (m *Content) GetType() string {
	if m != nil && m.Type != nil {
		return *m.Type
	}
	return ""
}

func (m *Content) GetData() []byte {
	if m != nil {
		return m.Data
	}
	return nil
}

func (m *Content) GetUrl() string {
	if m != nil && m.Url != nil {
		return *m.Url
	}
	return ""
}

type UserProfile struct {
	// Not required at creation time.  Assigned by server.
	Uuid   []byte  `protobuf:"bytes,1,opt,name=uuid" json:"uuid,omitempty"`
	Name   *string `protobuf:"bytes,2,req,name=name" json:"name,omitempty"`
	Status *string `protobuf:"bytes,3,opt,name=status" json:"status,omitempty"`
	// UI display of user
	Avatar           *Content `protobuf:"bytes,4,opt,name=avatar" json:"avatar,omitempty"`
	AvatarSmall      *Content `protobuf:"bytes,5,opt,name=avatar_small" json:"avatar_small,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (m *UserProfile) Reset()         { *m = UserProfile{} }
func (m *UserProfile) String() string { return proto.CompactTextString(m) }
func (*UserProfile) ProtoMessage()    {}

func (m *UserProfile) GetUuid() []byte {
	if m != nil {
		return m.Uuid
	}
	return nil
}

func (m *UserProfile) GetName() string {
	if m != nil && m.Name != nil {
		return *m.Name
	}
	return ""
}

func (m *UserProfile) GetStatus() string {
	if m != nil && m.Status != nil {
		return *m.Status
	}
	return ""
}

func (m *UserProfile) GetAvatar() *Content {
	if m != nil {
		return m.Avatar
	}
	return nil
}

func (m *UserProfile) GetAvatarSmall() *Content {
	if m != nil {
		return m.AvatarSmall
	}
	return nil
}

// Reference handle for user - either by id or by name (@david)
type UserRef struct {
	Uuid             []byte  `protobuf:"bytes,1,opt,name=uuid" json:"uuid,omitempty"`
	Name             *string `protobuf:"bytes,2,opt,name=name" json:"name,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *UserRef) Reset()         { *m = UserRef{} }
func (m *UserRef) String() string { return proto.CompactTextString(m) }
func (*UserRef) ProtoMessage()    {}

func (m *UserRef) GetUuid() []byte {
	if m != nil {
		return m.Uuid
	}
	return nil
}

func (m *UserRef) GetName() string {
	if m != nil && m.Name != nil {
		return *m.Name
	}
	return ""
}

type BeaconAdvertisement struct {
	Ibeacon *BeaconAdvertisement_IBeacon `protobuf:"bytes,1,opt,name=ibeacon" json:"ibeacon,omitempty"`
	// For supporting generic BLE mac address as id
	BleDevice        []byte `protobuf:"bytes,2,opt,name=ble_device" json:"ble_device,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (m *BeaconAdvertisement) Reset()         { *m = BeaconAdvertisement{} }
func (m *BeaconAdvertisement) String() string { return proto.CompactTextString(m) }
func (*BeaconAdvertisement) ProtoMessage()    {}

func (m *BeaconAdvertisement) GetIbeacon() *BeaconAdvertisement_IBeacon {
	if m != nil {
		return m.Ibeacon
	}
	return nil
}

func (m *BeaconAdvertisement) GetBleDevice() []byte {
	if m != nil {
		return m.BleDevice
	}
	return nil
}

type BeaconAdvertisement_IBeacon struct {
	Uuid             []byte `protobuf:"bytes,1,req,name=uuid" json:"uuid,omitempty"`
	Major            *int32 `protobuf:"varint,2,opt,name=major" json:"major,omitempty"`
	Minor            *int32 `protobuf:"varint,3,opt,name=minor" json:"minor,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (m *BeaconAdvertisement_IBeacon) Reset()         { *m = BeaconAdvertisement_IBeacon{} }
func (m *BeaconAdvertisement_IBeacon) String() string { return proto.CompactTextString(m) }
func (*BeaconAdvertisement_IBeacon) ProtoMessage()    {}

func (m *BeaconAdvertisement_IBeacon) GetUuid() []byte {
	if m != nil {
		return m.Uuid
	}
	return nil
}

func (m *BeaconAdvertisement_IBeacon) GetMajor() int32 {
	if m != nil && m.Major != nil {
		return *m.Major
	}
	return 0
}

func (m *BeaconAdvertisement_IBeacon) GetMinor() int32 {
	if m != nil && m.Minor != nil {
		return *m.Minor
	}
	return 0
}

// Hardware profile
type BeaconDeviceProfile struct {
	HardwareId       []byte               `protobuf:"bytes,1,req,name=hardware_id" json:"hardware_id,omitempty"`
	LocalName        *string              `protobuf:"bytes,2,req,name=local_name" json:"local_name,omitempty"`
	Password         []byte               `protobuf:"bytes,3,req,name=password" json:"password,omitempty"`
	AdvertiseInfo    *BeaconAdvertisement `protobuf:"bytes,4,req,name=advertise_info" json:"advertise_info,omitempty"`
	TxPower          *int32               `protobuf:"varint,5,opt,name=tx_power" json:"tx_power,omitempty"`
	TxFrequency      *int32               `protobuf:"varint,6,opt,name=tx_frequency" json:"tx_frequency,omitempty"`
	XXX_unrecognized []byte               `json:"-"`
}

func (m *BeaconDeviceProfile) Reset()         { *m = BeaconDeviceProfile{} }
func (m *BeaconDeviceProfile) String() string { return proto.CompactTextString(m) }
func (*BeaconDeviceProfile) ProtoMessage()    {}

func (m *BeaconDeviceProfile) GetHardwareId() []byte {
	if m != nil {
		return m.HardwareId
	}
	return nil
}

func (m *BeaconDeviceProfile) GetLocalName() string {
	if m != nil && m.LocalName != nil {
		return *m.LocalName
	}
	return ""
}

func (m *BeaconDeviceProfile) GetPassword() []byte {
	if m != nil {
		return m.Password
	}
	return nil
}

func (m *BeaconDeviceProfile) GetAdvertiseInfo() *BeaconAdvertisement {
	if m != nil {
		return m.AdvertiseInfo
	}
	return nil
}

func (m *BeaconDeviceProfile) GetTxPower() int32 {
	if m != nil && m.TxPower != nil {
		return *m.TxPower
	}
	return 0
}

func (m *BeaconDeviceProfile) GetTxFrequency() int32 {
	if m != nil && m.TxFrequency != nil {
		return *m.TxFrequency
	}
	return 0
}

// Summary of a beacon -- this is shared with mobile client.
type BeaconSummary struct {
	Id []byte `protobuf:"bytes,1,req,name=id" json:"id,omitempty"`
	// How this beacon advertises itself
	AdvertiseInfo *BeaconAdvertisement `protobuf:"bytes,2,req,name=advertise_info" json:"advertise_info,omitempty"`
	// Install date, unix time
	InstalledTimestamp *float64 `protobuf:"fixed64,3,req,name=installed_timestamp" json:"installed_timestamp,omitempty"`
	// Where the beacon is installed
	Location *Location `protobuf:"bytes,4,req,name=location" json:"location,omitempty"`
	Battery  *int32    `protobuf:"varint,5,opt,name=battery" json:"battery,omitempty"`
	// Owner of the beacon -- first user who provisioned a hardware beacon
	// Ownership can be transferred by releasing the beacon which will cause
	// a deletion of this record.
	Owner *UserRef `protobuf:"bytes,6,req,name=owner" json:"owner,omitempty"`
	// At least one label to establish the context of the beacon.  For v1, only 1 label.
	Labels []string `protobuf:"bytes,7,rep,name=labels" json:"labels,omitempty"`
	// For displaying the beacon icon/ logo etc.
	Avatar           *Content `protobuf:"bytes,8,opt,name=avatar" json:"avatar,omitempty"`
	AvatarSmall      *Content `protobuf:"bytes,9,opt,name=avatar_small" json:"avatar_small,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (m *BeaconSummary) Reset()         { *m = BeaconSummary{} }
func (m *BeaconSummary) String() string { return proto.CompactTextString(m) }
func (*BeaconSummary) ProtoMessage()    {}

func (m *BeaconSummary) GetId() []byte {
	if m != nil {
		return m.Id
	}
	return nil
}

func (m *BeaconSummary) GetAdvertiseInfo() *BeaconAdvertisement {
	if m != nil {
		return m.AdvertiseInfo
	}
	return nil
}

func (m *BeaconSummary) GetInstalledTimestamp() float64 {
	if m != nil && m.InstalledTimestamp != nil {
		return *m.InstalledTimestamp
	}
	return 0
}

func (m *BeaconSummary) GetLocation() *Location {
	if m != nil {
		return m.Location
	}
	return nil
}

func (m *BeaconSummary) GetBattery() int32 {
	if m != nil && m.Battery != nil {
		return *m.Battery
	}
	return 0
}

func (m *BeaconSummary) GetOwner() *UserRef {
	if m != nil {
		return m.Owner
	}
	return nil
}

func (m *BeaconSummary) GetLabels() []string {
	if m != nil {
		return m.Labels
	}
	return nil
}

func (m *BeaconSummary) GetAvatar() *Content {
	if m != nil {
		return m.Avatar
	}
	return nil
}

func (m *BeaconSummary) GetAvatarSmall() *Content {
	if m != nil {
		return m.AvatarSmall
	}
	return nil
}

// Detail beacon info -- server side; contains history, device profile, etc.
type Beacon struct {
	Id      []byte         `protobuf:"bytes,1,req,name=id" json:"id,omitempty"`
	Summary *BeaconSummary `protobuf:"bytes,2,req,name=summary" json:"summary,omitempty"`
	// Device programming history to support reseting of beacons on release or rollbacks.
	// Assumption: this doesn't happen often so a linear history can be stored in-place.
	// This may not be sent to a viewing mobile client.
	History []*BeaconDeviceProfile `protobuf:"bytes,3,rep,name=history" json:"history,omitempty"`
	// Access control list
	Acl              *string `protobuf:"bytes,4,req,name=acl" json:"acl,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *Beacon) Reset()         { *m = Beacon{} }
func (m *Beacon) String() string { return proto.CompactTextString(m) }
func (*Beacon) ProtoMessage()    {}

func (m *Beacon) GetId() []byte {
	if m != nil {
		return m.Id
	}
	return nil
}

func (m *Beacon) GetSummary() *BeaconSummary {
	if m != nil {
		return m.Summary
	}
	return nil
}

func (m *Beacon) GetHistory() []*BeaconDeviceProfile {
	if m != nil {
		return m.History
	}
	return nil
}

func (m *Beacon) GetAcl() string {
	if m != nil && m.Acl != nil {
		return *m.Acl
	}
	return ""
}

type Acl struct {
	Id               []byte     `protobuf:"bytes,1,opt,name=id" json:"id,omitempty"`
	Name             *string    `protobuf:"bytes,2,req,name=name" json:"name,omitempty"`
	Users            []*UserRef `protobuf:"bytes,3,rep,name=users" json:"users,omitempty"`
	XXX_unrecognized []byte     `json:"-"`
}

func (m *Acl) Reset()         { *m = Acl{} }
func (m *Acl) String() string { return proto.CompactTextString(m) }
func (*Acl) ProtoMessage()    {}

func (m *Acl) GetId() []byte {
	if m != nil {
		return m.Id
	}
	return nil
}

func (m *Acl) GetName() string {
	if m != nil && m.Name != nil {
		return *m.Name
	}
	return ""
}

func (m *Acl) GetUsers() []*UserRef {
	if m != nil {
		return m.Users
	}
	return nil
}

type Post struct {
	// The message id. Optional at creation time.  To be filled in by the server on commit.
	Uuid []byte `protobuf:"bytes,1,opt,name=uuid" json:"uuid,omitempty"`
	// Fractional seconds since epoch (unix time)
	Timestamp *float64 `protobuf:"fixed64,2,req,name=timestamp" json:"timestamp,omitempty"`
	Targets   [][]byte `protobuf:"bytes,3,rep,name=targets" json:"targets,omitempty"`
	// In reference of another post. Used by comments potentially.
	ReferencingPostId []byte `protobuf:"bytes,4,opt,name=referencing_post_id" json:"referencing_post_id,omitempty"`
	// Users are referenced with the '@' syntax in UI, like twitter
	// Values here should exclude the '@' character.
	From *UserRef   `protobuf:"bytes,5,req,name=from" json:"from,omitempty"`
	To   []*UserRef `protobuf:"bytes,6,rep,name=to" json:"to,omitempty"`
	// The content of this post.
	// One of the following should exist for a properly formed post.
	// It can be either actual post or comment to another post. Or an original post
	// with content and comment entered by user.
	Body    *Content `protobuf:"bytes,7,opt,name=body" json:"body,omitempty"`
	Comment *string  `protobuf:"bytes,8,opt,name=comment" json:"comment,omitempty"`
	// Hashtags, optional. For message targeting / matching - eg. #sfmuni
	// This may be populated after extracting from user comment.
	// Values here should exclude the '#' character.
	Hashtags         []string `protobuf:"bytes,9,rep,name=hashtags" json:"hashtags,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (m *Post) Reset()         { *m = Post{} }
func (m *Post) String() string { return proto.CompactTextString(m) }
func (*Post) ProtoMessage()    {}

func (m *Post) GetUuid() []byte {
	if m != nil {
		return m.Uuid
	}
	return nil
}

func (m *Post) GetTimestamp() float64 {
	if m != nil && m.Timestamp != nil {
		return *m.Timestamp
	}
	return 0
}

func (m *Post) GetTargets() [][]byte {
	if m != nil {
		return m.Targets
	}
	return nil
}

func (m *Post) GetReferencingPostId() []byte {
	if m != nil {
		return m.ReferencingPostId
	}
	return nil
}

func (m *Post) GetFrom() *UserRef {
	if m != nil {
		return m.From
	}
	return nil
}

func (m *Post) GetTo() []*UserRef {
	if m != nil {
		return m.To
	}
	return nil
}

func (m *Post) GetBody() *Content {
	if m != nil {
		return m.Body
	}
	return nil
}

func (m *Post) GetComment() string {
	if m != nil && m.Comment != nil {
		return *m.Comment
	}
	return ""
}

func (m *Post) GetHashtags() []string {
	if m != nil {
		return m.Hashtags
	}
	return nil
}

func init() {
}
