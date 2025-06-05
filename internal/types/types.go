package types

import (
	"google.golang.org/protobuf/types/descriptorpb"
)

// ProtoPath represents a path in the protobuf descriptor
type ProtoPath []int32

// CustomProtoData holds custom metadata for proto elements
type CustomProtoData struct {
	CustomComment *string
	CustomType    *string
	CustomName    *string
}

// NewCustomProtoData creates a new CustomProtoData instance
func NewCustomProtoData(comment string, customType string, name string) *CustomProtoData {
	cpd := CustomProtoData{}

	if comment != "" {
		cpd.CustomComment = &comment
	}
	if customType != "" {
		cpd.CustomType = &customType
	}
	if name != "" {
		cpd.CustomName = &name
	}
	return &cpd
}

// DescriptorConstraint defines the interface for descriptor types
type DescriptorConstraint interface {
	*descriptorpb.DescriptorProto |
		*descriptorpb.EnumDescriptorProto
}

// IndexMetadata holds metadata for indexed descriptors
type IndexMetadata[T DescriptorConstraint] struct {
	Descriptor     T
	FileDescriptor *descriptorpb.FileDescriptorProto
	Path           ProtoPath
	Key            string
	CustomData     *CustomProtoData
}

// MessageIndexType maps message names to their metadata
type MessageIndexType map[string]*IndexMetadata[*descriptorpb.DescriptorProto]

// EnumIndexType maps enum names to their metadata
type EnumIndexType map[string]*IndexMetadata[*descriptorpb.EnumDescriptorProto]

// ProtoIndex contains the indexed protobuf definitions
type ProtoIndex struct {
	MessageIndex MessageIndexType
	EnumIndex    EnumIndexType
}

// NewProtoIndex creates a new ProtoIndex instance
func NewProtoIndex() *ProtoIndex {
	return &ProtoIndex{
		MessageIndex: make(MessageIndexType),
		EnumIndex:    make(EnumIndexType),
	}
}

// AddMessage adds a message to the proto index
func (p *ProtoIndex) AddMessage(key string, path ProtoPath, file *descriptorpb.FileDescriptorProto, message *descriptorpb.DescriptorProto, data *CustomProtoData) {
	AddMetadata(p.MessageIndex, key, message, file, path, data)
}

// AddEnum adds an enum to the proto index
func (p *ProtoIndex) AddEnum(key string, path ProtoPath, file *descriptorpb.FileDescriptorProto, enum *descriptorpb.EnumDescriptorProto, data *CustomProtoData) {
	AddMetadata(p.EnumIndex, key, enum, file, path, data)
}

// AddMetadata adds metadata to an index map
func AddMetadata[T DescriptorConstraint](
	indexMap map[string]*IndexMetadata[T],
	key string,
	descriptor T,
	file *descriptorpb.FileDescriptorProto,
	path ProtoPath,
	data *CustomProtoData,
) {
	md, ok := indexMap[key]
	if !ok {
		md = &IndexMetadata[T]{}
		indexMap[key] = md
	}
	md.Descriptor = descriptor
	md.FileDescriptor = file
	md.Path = path
	md.Key = key
	md.CustomData = data
}
