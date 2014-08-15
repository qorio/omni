// Code generated by protoc-gen-go.
// source: passport_client.proto
// DO NOT EDIT!

package passport

import proto "code.google.com/p/goprotobuf/proto"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = math.Inf

type AuthRequest struct {
	Application      *string `protobuf:"bytes,1,opt,name=application" json:"application,omitempty"`
	Password         *string `protobuf:"bytes,2,req,name=password" json:"password,omitempty"`
	Email            *string `protobuf:"bytes,3,opt,name=email" json:"email,omitempty"`
	Phone            *string `protobuf:"bytes,4,opt,name=phone" json:"phone,omitempty"`
	Username         *string `protobuf:"bytes,5,opt,name=username" json:"username,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *AuthRequest) Reset()         { *m = AuthRequest{} }
func (m *AuthRequest) String() string { return proto.CompactTextString(m) }
func (*AuthRequest) ProtoMessage()    {}

func (m *AuthRequest) GetApplication() string {
	if m != nil && m.Application != nil {
		return *m.Application
	}
	return ""
}

func (m *AuthRequest) GetPassword() string {
	if m != nil && m.Password != nil {
		return *m.Password
	}
	return ""
}

func (m *AuthRequest) GetEmail() string {
	if m != nil && m.Email != nil {
		return *m.Email
	}
	return ""
}

func (m *AuthRequest) GetPhone() string {
	if m != nil && m.Phone != nil {
		return *m.Phone
	}
	return ""
}

func (m *AuthRequest) GetUsername() string {
	if m != nil && m.Username != nil {
		return *m.Username
	}
	return ""
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

type Login struct {
	Id               *string         `protobuf:"bytes,1,opt,name=id" json:"id,omitempty"`
	Email            *string         `protobuf:"bytes,2,opt,name=email" json:"email,omitempty"`
	Phone            *string         `protobuf:"bytes,3,opt,name=phone" json:"phone,omitempty"`
	Password         *string         `protobuf:"bytes,4,req,name=password" json:"password,omitempty"`
	Location         *Login_Location `protobuf:"bytes,5,opt,name=location" json:"location,omitempty"`
	Username         *string         `protobuf:"bytes,6,opt,name=username" json:"username,omitempty"`
	XXX_unrecognized []byte          `json:"-"`
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

func (m *Login) GetPhone() string {
	if m != nil && m.Phone != nil {
		return *m.Phone
	}
	return ""
}

func (m *Login) GetPassword() string {
	if m != nil && m.Password != nil {
		return *m.Password
	}
	return ""
}

func (m *Login) GetLocation() *Login_Location {
	if m != nil {
		return m.Location
	}
	return nil
}

func (m *Login) GetUsername() string {
	if m != nil && m.Username != nil {
		return *m.Username
	}
	return ""
}

type Login_Location struct {
	Lon              *float64 `protobuf:"fixed64,1,req,name=lon" json:"lon,omitempty"`
	Lat              *float64 `protobuf:"fixed64,2,req,name=lat" json:"lat,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (m *Login_Location) Reset()         { *m = Login_Location{} }
func (m *Login_Location) String() string { return proto.CompactTextString(m) }
func (*Login_Location) ProtoMessage()    {}

func (m *Login_Location) GetLon() float64 {
	if m != nil && m.Lon != nil {
		return *m.Lon
	}
	return 0
}

func (m *Login_Location) GetLat() float64 {
	if m != nil && m.Lat != nil {
		return *m.Lat
	}
	return 0
}

func init() {
}
