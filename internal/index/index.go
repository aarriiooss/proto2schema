package index

import (
	"strings"

	"proto2schema/internal/types"

	"google.golang.org/protobuf/types/descriptorpb"
)

// ParseNestedMessages recursively indexes nested messages and enums
func ParseNestedMessages(pIndex *types.ProtoIndex, parent string) {
	if msgMetadata, ok := pIndex.MessageIndex[parent]; ok {
		parentPath := msgMetadata.Path
		parentFile := msgMetadata.FileDescriptor
		for j, nested := range msgMetadata.Descriptor.NestedType {
			fqName := parent + "." + nested.GetName()
			nestedPath := append(append([]int32(nil), parentPath...), 3, int32(j))

			if cpd, ok := types.CustomProtoDataMap[fqName]; ok {
				pIndex.AddMessage(fqName, nestedPath, parentFile, nested, cpd)
			} else {
				pIndex.AddMessage(fqName, nestedPath, parentFile, nested, nil)
			}

			ParseNestedMessages(pIndex, fqName)
		}

		for j, enum := range msgMetadata.Descriptor.EnumType {
			fqName := parent + "." + enum.GetName()
			enumPath := append(append([]int32(nil), parentPath...), 3, int32(j))

			if cpd, ok := types.CustomProtoDataMap[fqName]; ok {
				pIndex.AddEnum(fqName, enumPath, parentFile, enum, cpd)
			} else {
				pIndex.AddEnum(fqName, enumPath, parentFile, enum, nil)
			}
		}
	}
}

// BuildProtoIndex builds a complete proto index from file descriptors
func BuildProtoIndex(files []*descriptorpb.FileDescriptorProto) *types.ProtoIndex {
	pIndex := types.NewProtoIndex()

	for _, file := range files {
		packageName := "." + file.GetPackage()

		// Build lookup maps for top-level messages and enums.
		for i, msg := range file.MessageType {
			fqName := packageName + "." + msg.GetName()
			topMsgPath := types.ProtoPath{4, int32(i)}

			if cpd, ok := types.CustomProtoDataMap[fqName]; ok {
				pIndex.AddMessage(fqName, topMsgPath, file, msg, cpd)
			} else {
				pIndex.AddMessage(fqName, topMsgPath, file, msg, nil)
			}

			ParseNestedMessages(pIndex, fqName)
		}
		for i, enum := range file.EnumType {
			fqName := packageName + "." + enum.GetName()
			enumPath := types.ProtoPath{5, int32(i)}

			if cpd, ok := types.CustomProtoDataMap[fqName]; ok {
				pIndex.AddEnum(fqName, enumPath, file, enum, cpd)
			} else {
				pIndex.AddEnum(fqName, enumPath, file, enum, nil)
			}
		}
	}

	return pIndex
}

// FindEntrypoints finds messages that don't appear in other fields
func FindEntrypoints(messageIndex types.MessageIndexType) []string {
	// Create a set to track used message types.
	used := make(map[string]bool)

	// Iterate through all messages in the messageIndex.
	for _, metadata := range messageIndex {
		// Process fields in this message.
		for _, field := range metadata.Descriptor.Field {
			if field.GetType() == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
				// Mark the type as used
				used[field.GetTypeName()] = true
			}
		}
	}

	// Now, any message in messageIndex that isn't in the 'used' map is an entry point.
	var entrypoints []string
	for name := range messageIndex {
		if !used[name] {
			entrypoints = append(entrypoints, name)
		}
	}
	return entrypoints
}

// LookupComment searches the sourceInfo locations for a comment at the given path
func LookupComment(path types.ProtoPath, sourceInfo *descriptorpb.SourceCodeInfo) string {
	if sourceInfo == nil {
		return ""
	}
	for _, loc := range sourceInfo.Location {
		if EqualPath(loc.Path, path) {
			comment := strings.TrimSpace(loc.GetLeadingComments())
			if comment == "" {
				comment = strings.TrimSpace(loc.GetTrailingComments())
			}
			return comment
		}
	}
	return ""
}

// EqualPath compares two proto paths for equality
func EqualPath(a, b types.ProtoPath) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if b[i] != v {
			return false
		}
	}
	return true
}
