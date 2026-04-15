// Package encoding provides custom JSON encoding for gRPC.
package encoding

import (
	"encoding/json"

	"google.golang.org/grpc/encoding"
)

// Name is the name of the JSON codec.
const Name = "json"

// jsonCodec implements the encoding.Codec interface using JSON.
type jsonCodec struct{}

func (jsonCodec) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (jsonCodec) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func (jsonCodec) Name() string {
	return Name
}

func (jsonCodec) String() string {
	return Name
}

// jsonCodecWithLenPrefix implements encoding.Codec with length-prefixed framing.
type jsonCodecWithLenPrefix struct{}

func (jsonCodecWithLenPrefix) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (jsonCodecWithLenPrefix) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func (jsonCodecWithLenPrefix) Name() string {
	return "proto" // Use "proto" name to override default protobuf codec
}

func (jsonCodecWithLenPrefix) String() string {
	return "json-proto"
}

// Register registers the JSON codec with gRPC.
// This should be called before starting the gRPC server or client.
func Register() {
	encoding.RegisterCodec(jsonCodec{})
}

// RegisterAsProto registers JSON codec as the default "proto" codec.
// This allows using JSON encoding without changing client/server code.
func RegisterAsProto() {
	encoding.RegisterCodec(jsonCodecWithLenPrefix{})
}