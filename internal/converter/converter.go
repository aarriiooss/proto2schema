package converter

import (
	"fmt"
	"strings"

	"proto2schema/internal/index"
	"proto2schema/internal/types"
	"proto2schema/internal/writer"

	"google.golang.org/protobuf/types/descriptorpb"
)

// GetReadableTypeName extracts the readable type name from a full type name
func GetReadableTypeName(typeName string, splitter string) string {
	typeNameSplit := strings.Split(typeName, splitter)
	readableTypeName := typeNameSplit[len(typeNameSplit)-1]
	return readableTypeName
}

// FetchCommentIfAny fetches a comment for the given key and path
func FetchCommentIfAny(key string, fileDescriptor *descriptorpb.FileDescriptorProto, path types.ProtoPath) string {
	if comment, ok := types.CustomProtoDataMap[key]; ok {
		return *comment.CustomComment
	}

	if strings.HasPrefix(fileDescriptor.GetPackage(), "google.") == true {
		return ""
	}
	comment := index.LookupComment(path, fileDescriptor.SourceCodeInfo)
	return comment
}

// PrintMessage prints a message definition following the desired format
func PrintMessage(
	pIndex *types.ProtoIndex,
	w writer.SchemaWriter,
	msgKey string,
	level int,
	visited map[string]bool,
) {
	msgMetadata := pIndex.MessageIndex[msgKey]
	msg := msgMetadata.Descriptor
	path := msgMetadata.Path

	if visited[msgKey] {
		w.Writef(level, "<Circular Ref> %s\n", msg.GetName())
		return
	}
	visited[msgKey] = true
	defer delete(visited, msgKey)

	// If there is a comment on the message, print it.
	writer.PrintCommentIfAny(w, FetchCommentIfAny(msgKey, msgMetadata.FileDescriptor, path), level)

	writer.OpenBlock(w, level, msg.GetName())

	for i, field := range msg.Field {
		// For a field, the path is the message's path plus [2, field_index] (2 = message.field)
		fieldPath := append(append([]int32(nil), path...), 2, int32(i))

		typeName := field.GetTypeName()
		writer.PrintCommentIfAny(w, FetchCommentIfAny("", msgMetadata.FileDescriptor, fieldPath), level+1)
		// Depending on field type and label, print accordingly.
		if field.GetLabel() == descriptorpb.FieldDescriptorProto_LABEL_REPEATED {
			writer.OpenList(w, level+1, field.GetName())

			switch field.GetType() {
			case descriptorpb.FieldDescriptorProto_TYPE_MESSAGE:
				PrintMessage(pIndex, w, typeName, level+2, visited)

			case descriptorpb.FieldDescriptorProto_TYPE_ENUM:
				if enumMeta, ok := pIndex.EnumIndex[typeName]; ok {
					writer.PrintCommentIfAny(w, FetchCommentIfAny("", enumMeta.FileDescriptor, enumMeta.Path), level+2)
					writer.OpenBlock(w, level+2, "ENUM "+enumMeta.Descriptor.GetName())
					PrintEnum(pIndex, w, typeName, level+2)
					writer.CloseBlock(w, level+2)
				}
			}

			writer.CloseList(w, level+1)
		} else {
			// For non-repeated fields, if the type is a message or enum, print inline.
			if field.GetType() == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
				readableTypeName := GetReadableTypeName(typeName, ".")
				header := fmt.Sprintf("%s %s", field.GetName(), readableTypeName)
				writer.OpenBlock(w, level+1, header)
				if _, ok := pIndex.MessageIndex[typeName]; ok {
					PrintMessage(pIndex, w, typeName, level+2, visited)
				}
				writer.CloseBlock(w, level+1)
			} else if field.GetType() == descriptorpb.FieldDescriptorProto_TYPE_ENUM {
				readableTypeName := GetReadableTypeName(typeName, ".")
				if enumMeta, ok := pIndex.EnumIndex[typeName]; ok {
					writer.PrintCommentIfAny(w, FetchCommentIfAny(enumMeta.Key, enumMeta.FileDescriptor, enumMeta.Path), level+1)
					writer.OpenEnum(w, level+1, "ENUM "+readableTypeName)
					PrintEnum(pIndex, w, typeName, level+2)
					writer.CloseEnum(w, level+1)
				}
			} else {
				humanReadableTypeName := GetReadableTypeName(field.GetType().String(), "_")
				w.Writefln(level+1, "%s %s", humanReadableTypeName, field.GetName())
			}
		}
		if i != len(msg.Field)-1 {
			w.WriteLine(level, "")
		}
	}

	writer.CloseBlock(w, level)
}

// PrintEnum prints an enum definition with its values and comments
func PrintEnum(pIndex *types.ProtoIndex, w writer.SchemaWriter, key string, level int) {
	enumMetadata := pIndex.EnumIndex[key]
	enum := enumMetadata.Descriptor
	path := enumMetadata.Path

	// Optionally print a comment for the enum.
	for i, value := range enum.Value {
		// For an enum value, the path is the enum's path plus [2, value_index] (2 = enum.value)
		valuePath := append(append([]int32(nil), path...), 2, int32(i))
		writer.PrintCommentIfAny(w, FetchCommentIfAny("", enumMetadata.FileDescriptor, valuePath), level)
		w.WriteLine(level, value.GetName())
		if i == len(enum.Value)-1 {
			continue
		}
		w.WriteLine(level, "")
	}
}
