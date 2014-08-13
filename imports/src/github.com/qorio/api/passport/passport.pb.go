// Code generated by protoc-gen-go.
// source: passport.proto
// DO NOT EDIT!

/*
Package passport is a generated protocol buffer package.

It is generated from these files:
	passport.proto
	passport_client.proto

It has these top-level messages:
	Blob
	Attribute
	Application
	Account
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
	EmbedSigninToken *bool           `protobuf:"varint,3,opt,name=embed_signin_token,def=0" json:"embed_signin_token,omitempty"`
	StringValue      *string         `protobuf:"bytes,4,opt,name=string_value" json:"string_value,omitempty"`
	NumberValue      *float64        `protobuf:"fixed64,5,opt,name=number_value" json:"number_value,omitempty"`
	BoolValue        *bool           `protobuf:"varint,6,opt,name=bool_value" json:"bool_value,omitempty"`
	BlobValue        *Blob           `protobuf:"bytes,7,opt,name=blob_value" json:"blob_value,omitempty"`
	XXX_unrecognized []byte          `json:"-"`
}

func (m *Attribute) Reset()         { *m = Attribute{} }
func (m *Attribute) String() string { return proto.CompactTextString(m) }
func (*Attribute) ProtoMessage()    {}

const Default_Attribute_EmbedSigninToken bool = false

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

func (m *Attribute) GetEmbedSigninToken() bool {
	if m != nil && m.EmbedSigninToken != nil {
		return *m.EmbedSigninToken
	}
	return Default_Attribute_EmbedSigninToken
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

type Application struct {
	Id               *string      `protobuf:"bytes,1,req,name=id" json:"id,omitempty"`
	Status           *string      `protobuf:"bytes,2,req,name=status" json:"status,omitempty"`
	AccountId        *string      `protobuf:"bytes,3,req,name=accountId" json:"accountId,omitempty"`
	Permissions      []string     `protobuf:"bytes,4,rep,name=permissions" json:"permissions,omitempty"`
	StartTimestamp   *float64     `protobuf:"fixed64,5,opt,name=startTimestamp" json:"startTimestamp,omitempty"`
	Attributes       []*Attribute `protobuf:"bytes,6,rep,name=attributes" json:"attributes,omitempty"`
	XXX_unrecognized []byte       `json:"-"`
}

func (m *Application) Reset()         { *m = Application{} }
func (m *Application) String() string { return proto.CompactTextString(m) }
func (*Application) ProtoMessage()    {}

func (m *Application) GetId() string {
	if m != nil && m.Id != nil {
		return *m.Id
	}
	return ""
}

func (m *Application) GetStatus() string {
	if m != nil && m.Status != nil {
		return *m.Status
	}
	return ""
}

func (m *Application) GetAccountId() string {
	if m != nil && m.AccountId != nil {
		return *m.AccountId
	}
	return ""
}

func (m *Application) GetPermissions() []string {
	if m != nil {
		return m.Permissions
	}
	return nil
}

func (m *Application) GetStartTimestamp() float64 {
	if m != nil && m.StartTimestamp != nil {
		return *m.StartTimestamp
	}
	return 0
}

func (m *Application) GetAttributes() []*Attribute {
	if m != nil {
		return m.Attributes
	}
	return nil
}

type Account struct {
	Id               *string        `protobuf:"bytes,1,opt,name=id" json:"id,omitempty"`
	Status           *string        `protobuf:"bytes,2,opt,name=status" json:"status,omitempty"`
	Primary          *Login         `protobuf:"bytes,3,req,name=primary" json:"primary,omitempty"`
	CreatedTimestamp *float64       `protobuf:"fixed64,4,opt,name=createdTimestamp" json:"createdTimestamp,omitempty"`
	Services         []*Application `protobuf:"bytes,5,rep,name=services" json:"services,omitempty"`
	XXX_unrecognized []byte         `json:"-"`
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

func (m *Account) GetPrimary() *Login {
	if m != nil {
		return m.Primary
	}
	return nil
}

func (m *Account) GetCreatedTimestamp() float64 {
	if m != nil && m.CreatedTimestamp != nil {
		return *m.CreatedTimestamp
	}
	return 0
}

func (m *Account) GetServices() []*Application {
	if m != nil {
		return m.Services
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
