// Code generated by protoc-gen-go.
// source: passport.proto
// DO NOT EDIT!

/*
Package passport is a generated protocol buffer package.

It is generated from these files:
	passport.proto

It has these top-level messages:
	Attribute
	Login
	Application
	Account
*/
package passport

import proto "code.google.com/p/goprotobuf/proto"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = math.Inf

type Attribute struct {
	Key              *string  `protobuf:"bytes,1,req,name=key" json:"key,omitempty"`
	StringValue      *string  `protobuf:"bytes,2,opt,name=string_value" json:"string_value,omitempty"`
	IntValue         *int64   `protobuf:"varint,3,opt,name=int_value" json:"int_value,omitempty"`
	DoubleValue      *float64 `protobuf:"fixed64,4,opt,name=double_value" json:"double_value,omitempty"`
	BoolValue        *bool    `protobuf:"varint,5,opt,name=bool_value" json:"bool_value,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (m *Attribute) Reset()         { *m = Attribute{} }
func (m *Attribute) String() string { return proto.CompactTextString(m) }
func (*Attribute) ProtoMessage()    {}

func (m *Attribute) GetKey() string {
	if m != nil && m.Key != nil {
		return *m.Key
	}
	return ""
}

func (m *Attribute) GetStringValue() string {
	if m != nil && m.StringValue != nil {
		return *m.StringValue
	}
	return ""
}

func (m *Attribute) GetIntValue() int64 {
	if m != nil && m.IntValue != nil {
		return *m.IntValue
	}
	return 0
}

func (m *Attribute) GetDoubleValue() float64 {
	if m != nil && m.DoubleValue != nil {
		return *m.DoubleValue
	}
	return 0
}

func (m *Attribute) GetBoolValue() bool {
	if m != nil && m.BoolValue != nil {
		return *m.BoolValue
	}
	return false
}

type Login struct {
	Id               *string `protobuf:"bytes,1,req,name=id" json:"id,omitempty"`
	Email            *string `protobuf:"bytes,2,req,name=email" json:"email,omitempty"`
	Password         *string `protobuf:"bytes,3,req,name=password" json:"password,omitempty"`
	AccountId        *string `protobuf:"bytes,4,req,name=accountId" json:"accountId,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *Login) Reset()         { *m = Login{} }
func (m *Login) String() string { return proto.CompactTextString(m) }
func (*Login) ProtoMessage()    {}

func (m *Login) GetId() string {
	if m != nil && m.Id != nil {
		return *m.Id
	}
	return ""
}

func (m *Login) GetEmail() string {
	if m != nil && m.Email != nil {
		return *m.Email
	}
	return ""
}

func (m *Login) GetPassword() string {
	if m != nil && m.Password != nil {
		return *m.Password
	}
	return ""
}

func (m *Login) GetAccountId() string {
	if m != nil && m.AccountId != nil {
		return *m.AccountId
	}
	return ""
}

type Application struct {
	Status           *string      `protobuf:"bytes,1,req,name=status" json:"status,omitempty"`
	Id               *string      `protobuf:"bytes,2,req,name=id" json:"id,omitempty"`
	AccountId        *string      `protobuf:"bytes,3,req,name=accountId" json:"accountId,omitempty"`
	StartTimestamp   *float64     `protobuf:"fixed64,4,req,name=startTimestamp" json:"startTimestamp,omitempty"`
	Attributes       []*Attribute `protobuf:"bytes,5,rep,name=attributes" json:"attributes,omitempty"`
	XXX_unrecognized []byte       `json:"-"`
}

func (m *Application) Reset()         { *m = Application{} }
func (m *Application) String() string { return proto.CompactTextString(m) }
func (*Application) ProtoMessage()    {}

func (m *Application) GetStatus() string {
	if m != nil && m.Status != nil {
		return *m.Status
	}
	return ""
}

func (m *Application) GetId() string {
	if m != nil && m.Id != nil {
		return *m.Id
	}
	return ""
}

func (m *Application) GetAccountId() string {
	if m != nil && m.AccountId != nil {
		return *m.AccountId
	}
	return ""
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
	Id               *string        `protobuf:"bytes,1,req,name=id" json:"id,omitempty"`
	Primary          *Login         `protobuf:"bytes,2,req,name=primary" json:"primary,omitempty"`
	CreatedTimestamp *float64       `protobuf:"fixed64,3,req,name=createdTimestamp" json:"createdTimestamp,omitempty"`
	Services         []*Application `protobuf:"bytes,4,rep,name=services" json:"services,omitempty"`
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

func init() {
}
