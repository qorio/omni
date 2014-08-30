// Code generated by protoc-gen-go.
// source: passport.proto
// DO NOT EDIT!

/*
Package passport is a generated protocol buffer package.

It is generated from these files:
	passport.proto

It has these top-level messages:
	Identity
	AuthResponse
	Account
	Blob
	Attribute
	Service
	AccountLogs
*/
package passport

import proto "code.google.com/p/goprotobuf/proto"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = math.Inf

type Attribute_Type int32

const (
	Attribute_STRING Attribute_Type = 1
	Attribute_NUMBER Attribute_Type = 2
	Attribute_BOOL   Attribute_Type = 3
	Attribute_BLOB   Attribute_Type = 4
)

var Attribute_Type_name = map[int32]string{
	1: "STRING",
	2: "NUMBER",
	3: "BOOL",
	4: "BLOB",
}
var Attribute_Type_value = map[string]int32{
	"STRING": 1,
	"NUMBER": 2,
	"BOOL":   3,
	"BLOB":   4,
}

func (x Attribute_Type) Enum() *Attribute_Type {
	p := new(Attribute_Type)
	*p = x
	return p
}
func (x Attribute_Type) String() string {
	return proto.EnumName(Attribute_Type_name, int32(x))
}
func (x *Attribute_Type) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(Attribute_Type_value, data, "Attribute_Type")
	if err != nil {
		return err
	}
	*x = Attribute_Type(value)
	return nil
}

// The type of the Identity is determined by what is
// presented to the server.  If password is specified,
// native passport identity by username/email/phone + password
// is used.  If oauth2_access_token is presented, the
// oauth2 identity is assumed.
type Identity struct {
	Id       *string            `protobuf:"bytes,1,opt,name=id" json:"id,omitempty"`
	Service  *string            `protobuf:"bytes,2,opt,name=service" json:"service,omitempty"`
	Location *Identity_Location `protobuf:"bytes,3,opt,name=location" json:"location,omitempty"`
	// NATIVE identity
	Password *string `protobuf:"bytes,10,opt,name=password" json:"password,omitempty"`
	Email    *string `protobuf:"bytes,11,opt,name=email" json:"email,omitempty"`
	Phone    *string `protobuf:"bytes,12,opt,name=phone" json:"phone,omitempty"`
	Username *string `protobuf:"bytes,13,opt,name=username" json:"username,omitempty"`
	// OAUTH2 identity here assumes that the
	// client has performed auth with the provider
	// and the provider has granted an access token.
	// The access token is then verified on the server
	// side via the providers server api.  Once the
	// access token is verified, another token for
	// accessing passport-authenticated systems is
	// issued.
	Oauth2Provider    *string `protobuf:"bytes,20,opt,name=oauth2_provider" json:"oauth2_provider,omitempty"`
	Oauth2AccountId   *string `protobuf:"bytes,21,opt,name=oauth2_account_id" json:"oauth2_account_id,omitempty"`
	Oauth2AccessToken *string `protobuf:"bytes,22,opt,name=oauth2_access_token" json:"oauth2_access_token,omitempty"`
	Oauth2AppId       *string `protobuf:"bytes,23,opt,name=oauth2_app_id" json:"oauth2_app_id,omitempty"`
	XXX_unrecognized  []byte  `json:"-"`
}

func (m *Identity) Reset()         { *m = Identity{} }
func (m *Identity) String() string { return proto.CompactTextString(m) }
func (*Identity) ProtoMessage()    {}

func (m *Identity) GetId() string {
	if m != nil && m.Id != nil {
		return *m.Id
	}
	return ""
}

func (m *Identity) GetService() string {
	if m != nil && m.Service != nil {
		return *m.Service
	}
	return ""
}

func (m *Identity) GetLocation() *Identity_Location {
	if m != nil {
		return m.Location
	}
	return nil
}

func (m *Identity) GetPassword() string {
	if m != nil && m.Password != nil {
		return *m.Password
	}
	return ""
}

func (m *Identity) GetEmail() string {
	if m != nil && m.Email != nil {
		return *m.Email
	}
	return ""
}

func (m *Identity) GetPhone() string {
	if m != nil && m.Phone != nil {
		return *m.Phone
	}
	return ""
}

func (m *Identity) GetUsername() string {
	if m != nil && m.Username != nil {
		return *m.Username
	}
	return ""
}

func (m *Identity) GetOauth2Provider() string {
	if m != nil && m.Oauth2Provider != nil {
		return *m.Oauth2Provider
	}
	return ""
}

func (m *Identity) GetOauth2AccountId() string {
	if m != nil && m.Oauth2AccountId != nil {
		return *m.Oauth2AccountId
	}
	return ""
}

func (m *Identity) GetOauth2AccessToken() string {
	if m != nil && m.Oauth2AccessToken != nil {
		return *m.Oauth2AccessToken
	}
	return ""
}

func (m *Identity) GetOauth2AppId() string {
	if m != nil && m.Oauth2AppId != nil {
		return *m.Oauth2AppId
	}
	return ""
}

type Identity_Location struct {
	Lon              *float64 `protobuf:"fixed64,1,req,name=lon" json:"lon,omitempty"`
	Lat              *float64 `protobuf:"fixed64,2,req,name=lat" json:"lat,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (m *Identity_Location) Reset()         { *m = Identity_Location{} }
func (m *Identity_Location) String() string { return proto.CompactTextString(m) }
func (*Identity_Location) ProtoMessage()    {}

func (m *Identity_Location) GetLon() float64 {
	if m != nil && m.Lon != nil {
		return *m.Lon
	}
	return 0
}

func (m *Identity_Location) GetLat() float64 {
	if m != nil && m.Lat != nil {
		return *m.Lat
	}
	return 0
}

type AuthResponse struct {
	Token            *string `protobuf:"bytes,1,req,name=token" json:"token,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *AuthResponse) Reset()         { *m = AuthResponse{} }
func (m *AuthResponse) String() string { return proto.CompactTextString(m) }
func (*AuthResponse) ProtoMessage()    {}

func (m *AuthResponse) GetToken() string {
	if m != nil && m.Token != nil {
		return *m.Token
	}
	return ""
}

type Account struct {
	Id               *string     `protobuf:"bytes,1,opt,name=id" json:"id,omitempty"`
	Status           *string     `protobuf:"bytes,2,opt,name=status" json:"status,omitempty"`
	CreatedTimestamp *float64    `protobuf:"fixed64,4,opt,name=created_timestamp" json:"created_timestamp,omitempty"`
	Services         []*Service  `protobuf:"bytes,5,rep,name=services" json:"services,omitempty"`
	Primary          *Identity   `protobuf:"bytes,3,req,name=primary" json:"primary,omitempty"`
	Identities       []*Identity `protobuf:"bytes,6,rep,name=identities" json:"identities,omitempty"`
	XXX_unrecognized []byte      `json:"-"`
}

func (m *Account) Reset()         { *m = Account{} }
func (m *Account) String() string { return proto.CompactTextString(m) }
func (*Account) ProtoMessage()    {}

func (m *Account) GetId() string {
	if m != nil && m.Id != nil {
		return *m.Id
	}
	return ""
}

func (m *Account) GetStatus() string {
	if m != nil && m.Status != nil {
		return *m.Status
	}
	return ""
}

func (m *Account) GetCreatedTimestamp() float64 {
	if m != nil && m.CreatedTimestamp != nil {
		return *m.CreatedTimestamp
	}
	return 0
}

func (m *Account) GetServices() []*Service {
	if m != nil {
		return m.Services
	}
	return nil
}

func (m *Account) GetPrimary() *Identity {
	if m != nil {
		return m.Primary
	}
	return nil
}

func (m *Account) GetIdentities() []*Identity {
	if m != nil {
		return m.Identities
	}
	return nil
}

type Blob struct {
	Type             *string `protobuf:"bytes,1,req,name=type" json:"type,omitempty"`
	Data             []byte  `protobuf:"bytes,2,req,name=data" json:"data,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *Blob) Reset()         { *m = Blob{} }
func (m *Blob) String() string { return proto.CompactTextString(m) }
func (*Blob) ProtoMessage()    {}

func (m *Blob) GetType() string {
	if m != nil && m.Type != nil {
		return *m.Type
	}
	return ""
}

func (m *Blob) GetData() []byte {
	if m != nil {
		return m.Data
	}
	return nil
}

type Attribute struct {
	Type             *Attribute_Type `protobuf:"varint,1,req,name=type,enum=passport.Attribute_Type" json:"type,omitempty"`
	Key              *string         `protobuf:"bytes,2,req,name=key" json:"key,omitempty"`
	EmbedInToken     *bool           `protobuf:"varint,3,opt,name=embed_in_token,def=0" json:"embed_in_token,omitempty"`
	StringValue      *string         `protobuf:"bytes,4,opt,name=string_value" json:"string_value,omitempty"`
	NumberValue      *float64        `protobuf:"fixed64,5,opt,name=number_value" json:"number_value,omitempty"`
	BoolValue        *bool           `protobuf:"varint,6,opt,name=bool_value" json:"bool_value,omitempty"`
	BlobValue        *Blob           `protobuf:"bytes,7,opt,name=blob_value" json:"blob_value,omitempty"`
	XXX_unrecognized []byte          `json:"-"`
}

func (m *Attribute) Reset()         { *m = Attribute{} }
func (m *Attribute) String() string { return proto.CompactTextString(m) }
func (*Attribute) ProtoMessage()    {}

const Default_Attribute_EmbedInToken bool = false

func (m *Attribute) GetType() Attribute_Type {
	if m != nil && m.Type != nil {
		return *m.Type
	}
	return Attribute_STRING
}

func (m *Attribute) GetKey() string {
	if m != nil && m.Key != nil {
		return *m.Key
	}
	return ""
}

func (m *Attribute) GetEmbedInToken() bool {
	if m != nil && m.EmbedInToken != nil {
		return *m.EmbedInToken
	}
	return Default_Attribute_EmbedInToken
}

func (m *Attribute) GetStringValue() string {
	if m != nil && m.StringValue != nil {
		return *m.StringValue
	}
	return ""
}

func (m *Attribute) GetNumberValue() float64 {
	if m != nil && m.NumberValue != nil {
		return *m.NumberValue
	}
	return 0
}

func (m *Attribute) GetBoolValue() bool {
	if m != nil && m.BoolValue != nil {
		return *m.BoolValue
	}
	return false
}

func (m *Attribute) GetBlobValue() *Blob {
	if m != nil {
		return m.BlobValue
	}
	return nil
}

type Service struct {
	Id               *string      `protobuf:"bytes,1,req,name=id" json:"id,omitempty"`
	Status           *string      `protobuf:"bytes,2,req,name=status" json:"status,omitempty"`
	AccountId        *string      `protobuf:"bytes,3,req,name=account_id" json:"account_id,omitempty"`
	Scopes           []string     `protobuf:"bytes,4,rep,name=scopes" json:"scopes,omitempty"`
	StartTimestamp   *float64     `protobuf:"fixed64,5,opt,name=start_timestamp" json:"start_timestamp,omitempty"`
	Attributes       []*Attribute `protobuf:"bytes,6,rep,name=attributes" json:"attributes,omitempty"`
	XXX_unrecognized []byte       `json:"-"`
}

func (m *Service) Reset()         { *m = Service{} }
func (m *Service) String() string { return proto.CompactTextString(m) }
func (*Service) ProtoMessage()    {}

func (m *Service) GetId() string {
	if m != nil && m.Id != nil {
		return *m.Id
	}
	return ""
}

func (m *Service) GetStatus() string {
	if m != nil && m.Status != nil {
		return *m.Status
	}
	return ""
}

func (m *Service) GetAccountId() string {
	if m != nil && m.AccountId != nil {
		return *m.AccountId
	}
	return ""
}

func (m *Service) GetScopes() []string {
	if m != nil {
		return m.Scopes
	}
	return nil
}

func (m *Service) GetStartTimestamp() float64 {
	if m != nil && m.StartTimestamp != nil {
		return *m.StartTimestamp
	}
	return 0
}

func (m *Service) GetAttributes() []*Attribute {
	if m != nil {
		return m.Attributes
	}
	return nil
}

type AccountLogs struct {
	Id               *string            `protobuf:"bytes,1,req,name=id" json:"id,omitempty"`
	Entries          []*AccountLogs_Log `protobuf:"bytes,2,rep,name=entries" json:"entries,omitempty"`
	XXX_unrecognized []byte             `json:"-"`
}

func (m *AccountLogs) Reset()         { *m = AccountLogs{} }
func (m *AccountLogs) String() string { return proto.CompactTextString(m) }
func (*AccountLogs) ProtoMessage()    {}

func (m *AccountLogs) GetId() string {
	if m != nil && m.Id != nil {
		return *m.Id
	}
	return ""
}

func (m *AccountLogs) GetEntries() []*AccountLogs_Log {
	if m != nil {
		return m.Entries
	}
	return nil
}

type AccountLogs_Log struct {
	Timestamp        *float64 `protobuf:"fixed64,1,req,name=timestamp" json:"timestamp,omitempty"`
	User             *string  `protobuf:"bytes,2,req,name=user" json:"user,omitempty"`
	Entry            *string  `protobuf:"bytes,3,req,name=entry" json:"entry,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (m *AccountLogs_Log) Reset()         { *m = AccountLogs_Log{} }
func (m *AccountLogs_Log) String() string { return proto.CompactTextString(m) }
func (*AccountLogs_Log) ProtoMessage()    {}

func (m *AccountLogs_Log) GetTimestamp() float64 {
	if m != nil && m.Timestamp != nil {
		return *m.Timestamp
	}
	return 0
}

func (m *AccountLogs_Log) GetUser() string {
	if m != nil && m.User != nil {
		return *m.User
	}
	return ""
}

func (m *AccountLogs_Log) GetEntry() string {
	if m != nil && m.Entry != nil {
		return *m.Entry
	}
	return ""
}

func init() {
	proto.RegisterEnum("passport.Attribute_Type", Attribute_Type_name, Attribute_Type_value)
}
