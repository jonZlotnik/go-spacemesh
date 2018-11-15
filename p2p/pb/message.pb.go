// Code generated by protoc-gen-go. DO NOT EDIT.
// source: message.proto

package pb

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// data common to all messages - Top level msg format
type CommonMessageData struct {
	SessionId            []byte   `protobuf:"bytes,1,opt,name=sessionId,proto3" json:"sessionId,omitempty"`
	Payload              []byte   `protobuf:"bytes,2,opt,name=payload,proto3" json:"payload,omitempty"`
	Timestamp            int64    `protobuf:"varint,3,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *CommonMessageData) Reset()         { *m = CommonMessageData{} }
func (m *CommonMessageData) String() string { return proto.CompactTextString(m) }
func (*CommonMessageData) ProtoMessage()    {}
func (*CommonMessageData) Descriptor() ([]byte, []int) {
	return fileDescriptor_33c57e4bae7b9afd, []int{0}
}

func (m *CommonMessageData) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CommonMessageData.Unmarshal(m, b)
}
func (m *CommonMessageData) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CommonMessageData.Marshal(b, m, deterministic)
}
func (m *CommonMessageData) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CommonMessageData.Merge(m, src)
}
func (m *CommonMessageData) XXX_Size() int {
	return xxx_messageInfo_CommonMessageData.Size(m)
}
func (m *CommonMessageData) XXX_DiscardUnknown() {
	xxx_messageInfo_CommonMessageData.DiscardUnknown(m)
}

var xxx_messageInfo_CommonMessageData proto.InternalMessageInfo

func (m *CommonMessageData) GetSessionId() []byte {
	if m != nil {
		return m.SessionId
	}
	return nil
}

func (m *CommonMessageData) GetPayload() []byte {
	if m != nil {
		return m.Payload
	}
	return nil
}

func (m *CommonMessageData) GetTimestamp() int64 {
	if m != nil {
		return m.Timestamp
	}
	return 0
}

// Handshake protocol data used for both request and response - sent unencrypted over the wire
type HandshakeData struct {
	SessionId            []byte   `protobuf:"bytes,1,opt,name=sessionId,proto3" json:"sessionId,omitempty"`
	Payload              []byte   `protobuf:"bytes,2,opt,name=payload,proto3" json:"payload,omitempty"`
	Timestamp            int64    `protobuf:"varint,3,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	ClientVersion        string   `protobuf:"bytes,4,opt,name=clientVersion,proto3" json:"clientVersion,omitempty"`
	NetworkID            int32    `protobuf:"varint,5,opt,name=networkID,proto3" json:"networkID,omitempty"`
	Protocol             string   `protobuf:"bytes,6,opt,name=protocol,proto3" json:"protocol,omitempty"`
	NodePubKey           []byte   `protobuf:"bytes,7,opt,name=nodePubKey,proto3" json:"nodePubKey,omitempty"`
	Iv                   []byte   `protobuf:"bytes,8,opt,name=iv,proto3" json:"iv,omitempty"`
	PubKey               []byte   `protobuf:"bytes,9,opt,name=pubKey,proto3" json:"pubKey,omitempty"`
	Hmac                 []byte   `protobuf:"bytes,10,opt,name=hmac,proto3" json:"hmac,omitempty"`
	Sign                 string   `protobuf:"bytes,11,opt,name=sign,proto3" json:"sign,omitempty"`
	Port                 uint32   `protobuf:"varint,12,opt,name=port,proto3" json:"port,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *HandshakeData) Reset()         { *m = HandshakeData{} }
func (m *HandshakeData) String() string { return proto.CompactTextString(m) }
func (*HandshakeData) ProtoMessage()    {}
func (*HandshakeData) Descriptor() ([]byte, []int) {
	return fileDescriptor_33c57e4bae7b9afd, []int{1}
}

func (m *HandshakeData) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_HandshakeData.Unmarshal(m, b)
}
func (m *HandshakeData) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_HandshakeData.Marshal(b, m, deterministic)
}
func (m *HandshakeData) XXX_Merge(src proto.Message) {
	xxx_messageInfo_HandshakeData.Merge(m, src)
}
func (m *HandshakeData) XXX_Size() int {
	return xxx_messageInfo_HandshakeData.Size(m)
}
func (m *HandshakeData) XXX_DiscardUnknown() {
	xxx_messageInfo_HandshakeData.DiscardUnknown(m)
}

var xxx_messageInfo_HandshakeData proto.InternalMessageInfo

func (m *HandshakeData) GetSessionId() []byte {
	if m != nil {
		return m.SessionId
	}
	return nil
}

func (m *HandshakeData) GetPayload() []byte {
	if m != nil {
		return m.Payload
	}
	return nil
}

func (m *HandshakeData) GetTimestamp() int64 {
	if m != nil {
		return m.Timestamp
	}
	return 0
}

func (m *HandshakeData) GetClientVersion() string {
	if m != nil {
		return m.ClientVersion
	}
	return ""
}

func (m *HandshakeData) GetNetworkID() int32 {
	if m != nil {
		return m.NetworkID
	}
	return 0
}

func (m *HandshakeData) GetProtocol() string {
	if m != nil {
		return m.Protocol
	}
	return ""
}

func (m *HandshakeData) GetNodePubKey() []byte {
	if m != nil {
		return m.NodePubKey
	}
	return nil
}

func (m *HandshakeData) GetIv() []byte {
	if m != nil {
		return m.Iv
	}
	return nil
}

func (m *HandshakeData) GetPubKey() []byte {
	if m != nil {
		return m.PubKey
	}
	return nil
}

func (m *HandshakeData) GetHmac() []byte {
	if m != nil {
		return m.Hmac
	}
	return nil
}

func (m *HandshakeData) GetSign() string {
	if m != nil {
		return m.Sign
	}
	return ""
}

func (m *HandshakeData) GetPort() uint32 {
	if m != nil {
		return m.Port
	}
	return 0
}

// used for protocol messages (non-handshake) - this is the decrypted CommonMessageData.payload
// it allows multi310.445plexing back to higher level protocols
// data is here and not in CommonMessageData to avoid leaked data on unencrypted connections
type ProtocolMessage struct {
	Metadata             *Metadata `protobuf:"bytes,1,opt,name=metadata,proto3" json:"metadata,omitempty"`
	Payload              []byte    `protobuf:"bytes,2,opt,name=payload,proto3" json:"payload,omitempty"`
	XXX_NoUnkeyedLiteral struct{}  `json:"-"`
	XXX_unrecognized     []byte    `json:"-"`
	XXX_sizecache        int32     `json:"-"`
}

func (m *ProtocolMessage) Reset()         { *m = ProtocolMessage{} }
func (m *ProtocolMessage) String() string { return proto.CompactTextString(m) }
func (*ProtocolMessage) ProtoMessage()    {}
func (*ProtocolMessage) Descriptor() ([]byte, []int) {
	return fileDescriptor_33c57e4bae7b9afd, []int{2}
}

func (m *ProtocolMessage) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ProtocolMessage.Unmarshal(m, b)
}
func (m *ProtocolMessage) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ProtocolMessage.Marshal(b, m, deterministic)
}
func (m *ProtocolMessage) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ProtocolMessage.Merge(m, src)
}
func (m *ProtocolMessage) XXX_Size() int {
	return xxx_messageInfo_ProtocolMessage.Size(m)
}
func (m *ProtocolMessage) XXX_DiscardUnknown() {
	xxx_messageInfo_ProtocolMessage.DiscardUnknown(m)
}

var xxx_messageInfo_ProtocolMessage proto.InternalMessageInfo

func (m *ProtocolMessage) GetMetadata() *Metadata {
	if m != nil {
		return m.Metadata
	}
	return nil
}

func (m *ProtocolMessage) GetPayload() []byte {
	if m != nil {
		return m.Payload
	}
	return nil
}

type Metadata struct {
	Protocol             string   `protobuf:"bytes,1,opt,name=protocol,proto3" json:"protocol,omitempty"`
	ClientVersion        string   `protobuf:"bytes,2,opt,name=clientVersion,proto3" json:"clientVersion,omitempty"`
	Timestamp            int64    `protobuf:"varint,3,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	Gossip               bool     `protobuf:"varint,4,opt,name=gossip,proto3" json:"gossip,omitempty"`
	AuthPubKey           []byte   `protobuf:"bytes,5,opt,name=authPubKey,proto3" json:"authPubKey,omitempty"`
	AuthorSign           string   `protobuf:"bytes,6,opt,name=authorSign,proto3" json:"authorSign,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Metadata) Reset()         { *m = Metadata{} }
func (m *Metadata) String() string { return proto.CompactTextString(m) }
func (*Metadata) ProtoMessage()    {}
func (*Metadata) Descriptor() ([]byte, []int) {
	return fileDescriptor_33c57e4bae7b9afd, []int{3}
}

func (m *Metadata) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Metadata.Unmarshal(m, b)
}
func (m *Metadata) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Metadata.Marshal(b, m, deterministic)
}
func (m *Metadata) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Metadata.Merge(m, src)
}
func (m *Metadata) XXX_Size() int {
	return xxx_messageInfo_Metadata.Size(m)
}
func (m *Metadata) XXX_DiscardUnknown() {
	xxx_messageInfo_Metadata.DiscardUnknown(m)
}

var xxx_messageInfo_Metadata proto.InternalMessageInfo

func (m *Metadata) GetProtocol() string {
	if m != nil {
		return m.Protocol
	}
	return ""
}

func (m *Metadata) GetClientVersion() string {
	if m != nil {
		return m.ClientVersion
	}
	return ""
}

func (m *Metadata) GetTimestamp() int64 {
	if m != nil {
		return m.Timestamp
	}
	return 0
}

func (m *Metadata) GetGossip() bool {
	if m != nil {
		return m.Gossip
	}
	return false
}

func (m *Metadata) GetAuthPubKey() []byte {
	if m != nil {
		return m.AuthPubKey
	}
	return nil
}

func (m *Metadata) GetAuthorSign() string {
	if m != nil {
		return m.AuthorSign
	}
	return ""
}

type MessageWrapper struct {
	Type                 uint32   `protobuf:"varint,1,opt,name=type,proto3" json:"type,omitempty"`
	Req                  bool     `protobuf:"varint,2,opt,name=req,proto3" json:"req,omitempty"`
	ReqID                uint64   `protobuf:"varint,3,opt,name=reqID,proto3" json:"reqID,omitempty"`
	Payload              []byte   `protobuf:"bytes,4,opt,name=payload,proto3" json:"payload,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *MessageWrapper) Reset()         { *m = MessageWrapper{} }
func (m *MessageWrapper) String() string { return proto.CompactTextString(m) }
func (*MessageWrapper) ProtoMessage()    {}
func (*MessageWrapper) Descriptor() ([]byte, []int) {
	return fileDescriptor_33c57e4bae7b9afd, []int{4}
}

func (m *MessageWrapper) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_MessageWrapper.Unmarshal(m, b)
}
func (m *MessageWrapper) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_MessageWrapper.Marshal(b, m, deterministic)
}
func (m *MessageWrapper) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MessageWrapper.Merge(m, src)
}
func (m *MessageWrapper) XXX_Size() int {
	return xxx_messageInfo_MessageWrapper.Size(m)
}
func (m *MessageWrapper) XXX_DiscardUnknown() {
	xxx_messageInfo_MessageWrapper.DiscardUnknown(m)
}

var xxx_messageInfo_MessageWrapper proto.InternalMessageInfo

func (m *MessageWrapper) GetType() uint32 {
	if m != nil {
		return m.Type
	}
	return 0
}

func (m *MessageWrapper) GetReq() bool {
	if m != nil {
		return m.Req
	}
	return false
}

func (m *MessageWrapper) GetReqID() uint64 {
	if m != nil {
		return m.ReqID
	}
	return 0
}

func (m *MessageWrapper) GetPayload() []byte {
	if m != nil {
		return m.Payload
	}
	return nil
}

func init() {
	proto.RegisterType((*CommonMessageData)(nil), "pb.CommonMessageData")
	proto.RegisterType((*HandshakeData)(nil), "pb.HandshakeData")
	proto.RegisterType((*ProtocolMessage)(nil), "pb.ProtocolMessage")
	proto.RegisterType((*Metadata)(nil), "pb.Metadata")
	proto.RegisterType((*MessageWrapper)(nil), "pb.MessageWrapper")
}

func init() { proto.RegisterFile("message.proto", fileDescriptor_33c57e4bae7b9afd) }

var fileDescriptor_33c57e4bae7b9afd = []byte{
	// 412 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xb4, 0x91, 0xc1, 0x8e, 0xd3, 0x30,
	0x10, 0x86, 0xe5, 0x34, 0xc9, 0xa6, 0xb3, 0xcd, 0x02, 0x16, 0x5a, 0x59, 0x08, 0xa1, 0x28, 0xe2,
	0x90, 0x53, 0x0f, 0xf0, 0x06, 0xd0, 0x03, 0x15, 0x5a, 0x69, 0x65, 0x04, 0x48, 0xdc, 0x9c, 0xc6,
	0xb4, 0xd6, 0x36, 0xb1, 0xd7, 0xf6, 0x2e, 0xea, 0xeb, 0x71, 0xe3, 0xad, 0x90, 0x27, 0x69, 0xd3,
	0x82, 0xca, 0x8d, 0xdb, 0xfc, 0xdf, 0xc4, 0xf6, 0xe4, 0x1b, 0xc8, 0x5b, 0xe9, 0x9c, 0x58, 0xcb,
	0xb9, 0xb1, 0xda, 0x6b, 0x1a, 0x99, 0xba, 0x54, 0xf0, 0xec, 0xbd, 0x6e, 0x5b, 0xdd, 0xdd, 0xf4,
	0xad, 0x85, 0xf0, 0x82, 0xbe, 0x84, 0xa9, 0x93, 0xce, 0x29, 0xdd, 0x2d, 0x1b, 0x46, 0x0a, 0x52,
	0xcd, 0xf8, 0x08, 0x28, 0x83, 0x0b, 0x23, 0x76, 0x5b, 0x2d, 0x1a, 0x16, 0x61, 0x6f, 0x1f, 0xc3,
	0x39, 0xaf, 0x5a, 0xe9, 0xbc, 0x68, 0x0d, 0x9b, 0x14, 0xa4, 0x9a, 0xf0, 0x11, 0x94, 0xbf, 0x22,
	0xc8, 0x3f, 0x88, 0xae, 0x71, 0x1b, 0x71, 0xf7, 0x1f, 0xdf, 0xa1, 0xaf, 0x21, 0x5f, 0x6d, 0x95,
	0xec, 0xfc, 0x17, 0x69, 0xc3, 0x55, 0x2c, 0x2e, 0x48, 0x35, 0xe5, 0xa7, 0x30, 0xdc, 0xd1, 0x49,
	0xff, 0x43, 0xdb, 0xbb, 0xe5, 0x82, 0x25, 0x05, 0xa9, 0x12, 0x3e, 0x02, 0xfa, 0x02, 0x32, 0x74,
	0xb4, 0xd2, 0x5b, 0x96, 0xe2, 0xf1, 0x43, 0xa6, 0xaf, 0x00, 0x3a, 0xdd, 0xc8, 0xdb, 0x87, 0xfa,
	0xa3, 0xdc, 0xb1, 0x0b, 0x1c, 0xed, 0x88, 0xd0, 0x2b, 0x88, 0xd4, 0x23, 0xcb, 0x90, 0x47, 0xea,
	0x91, 0x5e, 0x43, 0x6a, 0xfa, 0x6f, 0xa7, 0xc8, 0x86, 0x44, 0x29, 0xc4, 0x9b, 0x56, 0xac, 0x18,
	0x20, 0xc5, 0x3a, 0x30, 0xa7, 0xd6, 0x1d, 0xbb, 0xc4, 0x37, 0xb1, 0x0e, 0xcc, 0x68, 0xeb, 0xd9,
	0xac, 0x20, 0x55, 0xce, 0xb1, 0x2e, 0x3f, 0xc3, 0x93, 0xdb, 0x61, 0x9e, 0x61, 0x71, 0xb4, 0x82,
	0xac, 0x95, 0x5e, 0x34, 0xc2, 0x0b, 0x74, 0x79, 0xf9, 0x66, 0x36, 0x37, 0xf5, 0xfc, 0x66, 0x60,
	0xfc, 0xd0, 0x3d, 0x2f, 0xb6, 0xfc, 0x49, 0x20, 0xdb, 0x1f, 0x38, 0x71, 0x40, 0xfe, 0x70, 0xf0,
	0x97, 0xe3, 0xe8, 0x8c, 0xe3, 0x7f, 0xec, 0xe9, 0x1a, 0xd2, 0xb5, 0x76, 0x4e, 0x19, 0x5c, 0x50,
	0xc6, 0x87, 0x14, 0xfc, 0x8a, 0x07, 0xbf, 0x19, 0xfc, 0x26, 0xbd, 0xdf, 0x91, 0xec, 0xfb, 0xda,
	0x7e, 0x0a, 0xa6, 0xfa, 0xed, 0x1c, 0x91, 0xf2, 0x3b, 0x5c, 0x0d, 0x4e, 0xbe, 0x5a, 0x61, 0x8c,
	0xb4, 0xc1, 0xa0, 0xdf, 0x19, 0x89, 0x7f, 0x91, 0x73, 0xac, 0xe9, 0x53, 0x98, 0x58, 0x79, 0x8f,
	0x73, 0x67, 0x3c, 0x94, 0xf4, 0x39, 0x24, 0x56, 0xde, 0x2f, 0x17, 0x38, 0x69, 0xcc, 0xfb, 0x70,
	0x2c, 0x2b, 0x3e, 0x91, 0xf5, 0x2e, 0xfe, 0x16, 0x99, 0xba, 0x4e, 0xd1, 0xc9, 0xdb, 0xdf, 0x01,
	0x00, 0x00, 0xff, 0xff, 0xfc, 0x43, 0x08, 0x68, 0x5c, 0x03, 0x00, 0x00,
}
