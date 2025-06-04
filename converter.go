package main

import (
	"flag"
	"fmt"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"io"
	"log"
	"os"
	"strings"
)

const protoIndexKey = iota

// Some fields, messages, etc have comments etc that are not useful
type customProtoData struct {
	customComment *string
	customType    *string
	customName    *string
}

func newCustomProtoData(comment string, customType string, name string) *customProtoData {
	cpd := customProtoData{}

	if comment != "" {
		cpd.customComment = &comment
	}
	if customType != "" {
		cpd.customType = &customType
	}
	if name != "" {
		cpd.customName = &name
	}
	return &cpd
}

// currently only custom comment fetching is implemented
var customProtoDataMap = map[string]*customProtoData{
	".google.protobuf.Timestamp": newCustomProtoData(
		"In JSON format, the Timestamp type is encoded as a string in the RFC 3339 format.",
		"",
		"",
	),
	//".google.protobuf.Timestamp": {
	//	customComment: "In JSON format, the Timestamp type is encoded as a string in the RFC 3339 format.",
	//	customType:    "",
	//	customName:    "",
	//},
	//".tutorial.PhoneType": {
	//	customComment: "abcdef",
	//	customType:    "",
	//	customName:    "",
	//},
}

var (
	descriptorSetPath = flag.String(
		"descriptor_set",
		"",
		"path to a FileDescriptorSet (.binpb). If non‐empty, run in standalone mode; otherwise act as a protoc plugin.",
	)
	// Allow user to override where to write the single .fdhschema in standalone mode:
	outSchemaPath = flag.String(
		"out",
		"",
		"path to the .fdhschema file to write when using --descriptor_set. Defaults to replacing .binpb with .fdhschema.",
	)
)

// --descriptor_set="./gen/addressbook.binpb" --out="./gen/addressbook.fdhschema"

type protoPath []int32

type descriptorConstraint interface {
	*descriptorpb.DescriptorProto |
		*descriptorpb.EnumDescriptorProto
}

type indexMetadata[T descriptorConstraint] struct {
	descriptor T
	// file descriptor of the file this message belongs to
	// useful for picking out comments or anything else we can't get at the message level
	fileDescriptor *descriptorpb.FileDescriptorProto

	// path of this field in sourceInfo from fileDescriptor
	path protoPath

	// key in parent map - is this needed?
	key string

	customData *customProtoData
}

type messageIndexType map[string]*indexMetadata[*descriptorpb.DescriptorProto]
type enumIndexType map[string]*indexMetadata[*descriptorpb.EnumDescriptorProto]

type protoIndex struct {
	messageIndex messageIndexType
	enumIndex    enumIndexType
}

func addMetadata[T descriptorConstraint](
	indexMap map[string]*indexMetadata[T],
	key string,
	descriptor T,
	file *descriptorpb.FileDescriptorProto,
	path protoPath,
	data *customProtoData,
) {
	md, ok := indexMap[key]
	if !ok {
		md = &indexMetadata[T]{}
		indexMap[key] = md
	}
	md.descriptor = descriptor
	md.fileDescriptor = file
	md.path = path
	md.key = key
	md.customData = data
}

func (p protoIndex) addMessage(key string, path protoPath, file *descriptorpb.FileDescriptorProto, message *descriptorpb.DescriptorProto, data *customProtoData) {
	addMetadata(p.messageIndex, key, message, file, path, data)
}

func (p protoIndex) addEnum(key string, path protoPath, file *descriptorpb.FileDescriptorProto, message *descriptorpb.EnumDescriptorProto, data *customProtoData) {
	addMetadata(p.enumIndex, key, message, file, path, data)
}

func newProtoIndex() *protoIndex {
	return &protoIndex{
		messageIndex: make(messageIndexType),
		enumIndex:    make(enumIndexType),
	}
}

type SchemaWriter interface {
	// Writef write formatted line with indentation level
	Writef(level int, format string, args ...interface{})
	// WriteLine write raw line (with new line) with indentation level
	WriteLine(level int, line string)
	// Writefln adds a new line to Writef call
	Writefln(level int, format string, args ...interface{})
}

type fileWriter struct {
	writer io.Writer
	logger *log.Logger
}

func (fw *fileWriter) Writef(level int, format string, args ...interface{}) {
	indent := strings.Repeat("  ", level)
	_, err := fmt.Fprintf(fw.writer, indent+format, args...)
	if err != nil {
		fw.logger.Println(err)
	}
}

func (fw *fileWriter) WriteLine(level int, line string) {
	indent := strings.Repeat("  ", level)
	_, err := fmt.Fprintf(fw.writer, "%s%s\n", indent, line)
	if err != nil {
		fw.logger.Println(err)
	}
}

func (fw *fileWriter) Writefln(level int, format string, args ...interface{}) {
	fw.Writef(level, format, args...)
	fw.WriteLine(0, "")
}

func NewFileWriter(w io.Writer, l *log.Logger) SchemaWriter {
	return &fileWriter{
		writer: w,
		logger: l,
	}
}

func openBlock(w SchemaWriter, level int, header string) {
	w.Writefln(level, "%s {", header)
}

func closeBlock(w SchemaWriter, level int) {
	w.Writefln(level, "}")
}

func openList(w SchemaWriter, level int, header string) {
	w.Writefln(level, "%s [", header)
}

func closeList(w SchemaWriter, level int) {
	w.Writefln(level, "]")
}

func openEnum(w SchemaWriter, level int, header string) {
	w.Writefln(level, "%s (", header)
}

func closeEnum(w SchemaWriter, level int) {
	w.Writefln(level, ")")
}

func printCommentIfAny(w SchemaWriter, comment string, level int) {
	if comment != "" {
		w.Writefln(level, "// %s", comment)
	}
}

func getReadableTypeName(typeName string, splitter string) string {
	typeNameSplit := strings.Split(typeName, splitter)
	readableTypeName := typeNameSplit[len(typeNameSplit)-1]
	return readableTypeName
}

func fetchCommentIfAny(key string, fileDescriptor *descriptorpb.FileDescriptorProto, path protoPath) string {
	if comment, ok := customProtoDataMap[key]; ok {
		return *comment.customComment
	}

	if strings.HasPrefix(fileDescriptor.GetPackage(), "google.") == true {
		return ""
	}
	comment := lookupComment(path, fileDescriptor.SourceCodeInfo)
	return comment
}

//func fetchComment[T descriptorConstraint](metadata indexMetadata[T], path protoPath) string {
//	if metadata.customData != nil && metadata.customData.customComment != nil {
//		return *metadata.customData.customComment
//	}
//
//	if strings.HasPrefix(metadata.fileDescriptor.GetPackage(), "google.") == true {
//		return ""
//	}
//	comment := lookupComment(path, metadata.fileDescriptor.SourceCodeInfo)
//	return comment
//}

// printMessage prints a message definition following the desired format.
// It handles scalar fields, nested message fields, and enum fields.
func printMessage(
	pIndex protoIndex,
	w SchemaWriter,
	msgKey string,
	level int,
	visited map[string]bool,
) {
	//protoIndexes := ctx.Value(protoIndexKey).(*protoIndex)

	msgMetadata := pIndex.messageIndex[msgKey]
	msg := msgMetadata.descriptor
	path := msgMetadata.path

	if visited[msgKey] {
		w.Writef(level, "<Circular Ref> %s\n", msg.GetName())
		return
	}
	visited[msgKey] = true
	defer delete(visited, msgKey)

	// If there is a comment on the message, print it.
	printCommentIfAny(w, fetchCommentIfAny(msgKey, msgMetadata.fileDescriptor, path), level)
	//printCommentIfAny(w, fetchComment(*msgMetadata, path), level)

	openBlock(w, level, msg.GetName())

	for i, field := range msg.Field {
		// For a field, the path is the message's path plus [2, field_index] (2 = message.field)
		fieldPath := append(append([]int32(nil), path...), 2, int32(i))

		typeName := field.GetTypeName()
		printCommentIfAny(w, fetchCommentIfAny("", msgMetadata.fileDescriptor, fieldPath), level+1)
		//printCommentIfAny(w, fetchComment(*msgMetadata, fieldPath), level+1)
		// Depending on field type and label, print accordingly.
		if field.GetLabel() == descriptorpb.FieldDescriptorProto_LABEL_REPEATED {

			openList(w, level+1, field.GetName())

			switch field.GetType() {
			case descriptorpb.FieldDescriptorProto_TYPE_MESSAGE:
				printMessage(pIndex, w, typeName, level+2, visited)

			case descriptorpb.FieldDescriptorProto_TYPE_ENUM:
				if enumMeta, ok := pIndex.enumIndex[typeName]; ok {
					printCommentIfAny(w, fetchCommentIfAny("", enumMeta.fileDescriptor, enumMeta.path), level+2)
					//printCommentIfAny(w, fetchComment(*enumMeta, enumMeta.path), level+2)
					openBlock(w, level+2, "ENUM "+enumMeta.descriptor.GetName())
					printEnum(pIndex, w, typeName, level+2)
					closeBlock(w, level+2)
				}
			}

			closeList(w, level+1)
		} else {
			// For non-repeated fields, if the type is a message or enum, print inline.
			if field.GetType() == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
				readableTypeName := getReadableTypeName(typeName, ".")
				header := fmt.Sprintf("%s %s", field.GetName(), readableTypeName)
				openBlock(w, level+1, header)
				if _, ok := pIndex.messageIndex[typeName]; ok {
					printMessage(pIndex, w, typeName, level+2, visited)
				}
				closeBlock(w, level+1)
			} else if field.GetType() == descriptorpb.FieldDescriptorProto_TYPE_ENUM {
				readableTypeName := getReadableTypeName(typeName, ".")
				if enumMeta, ok := pIndex.enumIndex[typeName]; ok {
					//printCommentIfAny(w, fetchCommentIfAny(typeName, enumMeta.fileDescriptor, enumMeta.path), level+1)
					printCommentIfAny(w, fetchCommentIfAny(enumMeta.key, enumMeta.fileDescriptor, enumMeta.path), level+1)
					//printCommentIfAny(w, fetchComment(*enumMeta, enumMeta.path), level+1)
					openEnum(w, level+1, "ENUM "+readableTypeName)
					printEnum(pIndex, w, typeName, level+2)
					closeEnum(w, level+1)
				}
			} else {
				humanReadableTypeName := getReadableTypeName(field.GetType().String(), "_")
				w.Writefln(level+1, "%s %s", humanReadableTypeName, field.GetName())
			}
		}
		if i != len(msg.Field)-1 {
			w.WriteLine(level, "")
		}
	}

	closeBlock(w, level)
}

// printEnum prints an enum definition with its values and comments.
func printEnum(pIndex protoIndex, w SchemaWriter, key string, level int) {
	//protoIndexes := ctx.Value(protoIndexKey).(*protoIndex)
	enumMetadata := pIndex.enumIndex[key]
	enum := enumMetadata.descriptor
	path := enumMetadata.path

	// Optionally print a comment for the enum.
	for i, value := range enum.Value {
		// For an enum value, the path is the enum's path plus [2, value_index] (2 = enum.value)
		valuePath := append(append([]int32(nil), path...), 2, int32(i))
		printCommentIfAny(w, fetchCommentIfAny("", enumMetadata.fileDescriptor, valuePath), level)
		//printCommentIfAny(w, fetchComment(*enumMetadata, valuePath), level)
		w.WriteLine(level, value.GetName())
		if i == len(enum.Value)-1 {
			continue
		}
		w.WriteLine(level, "")
	}
}

// lookupComment searches the sourceInfo locations for one whose Path matches the given path.
// It returns any leading comments (or, if absent, trailing comments) attached to that element.
func lookupComment(path protoPath, sourceInfo *descriptorpb.SourceCodeInfo) string {
	if sourceInfo == nil {
		return ""
	}
	for _, loc := range sourceInfo.Location {
		if equalPath(loc.Path, path) {
			comment := strings.TrimSpace(loc.GetLeadingComments())
			if comment == "" {
				comment = strings.TrimSpace(loc.GetTrailingComments())
			}
			return comment
		}
	}
	return ""
}

// entrypoint is any message that doesn't appear in other fields (or types?)
func findEntrypoints(messageIndex messageIndexType) []string {
	// Create a set to track used message types.
	used := make(map[string]bool)

	// Iterate through all messages in the messageIndex.
	for _, metadata := range messageIndex {
		// Process fields in this message.
		for _, field := range metadata.descriptor.Field {
			if field.GetType() == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
				// Mark the type as used. The field type name should be fully qualified.
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

func indexNestedMessages(pIndex protoIndex, parent string) {
	//protoIndexes := ctx.Value(protoIndexKey).(*protoIndex)

	if msgMetadata, ok := pIndex.messageIndex[parent]; ok {
		parentPath := msgMetadata.path
		parentFile := msgMetadata.fileDescriptor
		for j, nested := range msgMetadata.descriptor.NestedType {
			fqName := parent + "." + nested.GetName()
			nestedPath := append(append([]int32(nil), parentPath...), 3, int32(j))

			if cpd, ok := customProtoDataMap[fqName]; ok {
				pIndex.addMessage(fqName, nestedPath, parentFile, nested, cpd)
			} else {
				pIndex.addMessage(fqName, nestedPath, parentFile, nested, nil)
			}

			//pIndex.addMessage(fqName, nestedPath, parentFile, nested)
			indexNestedMessages(pIndex, fqName)
		}

		for j, enum := range msgMetadata.descriptor.EnumType {
			fqName := parent + "." + enum.GetName()
			enumPath := append(append([]int32(nil), parentPath...), 3, int32(j))

			if cpd, ok := customProtoDataMap[fqName]; ok {
				pIndex.addEnum(fqName, enumPath, parentFile, enum, cpd)
			} else {
				pIndex.addEnum(fqName, enumPath, parentFile, enum, nil)
			}

			//pIndex.addEnum(fqName, enumPath, parentFile, enum)
		}
	}
}

// equalPath compares two slices of int32 for equality.
func equalPath(a, b protoPath) bool {
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

func runStandalone(descriptorPath *string, outPath *string) int {
	//descriptorPath := "gen/addressbook.binpb"
	//fdhSchemaPath := "gen/addressbook.fdhschema"
	logger := log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)
	pIndex := newProtoIndex()
	//ctx := context.WithValue(context.TODO(), protoIndexKey, pIndex)

	logger.Println("Attempting to read binbp")
	// Read the file descriptor set generated by protoc.
	data, err := os.ReadFile(*descriptorPath)
	if err != nil {
		logger.Fatalf("Failed to read descriptor set: %v", err)
	}
	logger.Printf("Read descriptor set: %s", *descriptorPath)

	// Unmarshal the data into a FileDescriptorSet.
	var fds descriptorpb.FileDescriptorSet
	if err := proto.Unmarshal(data, &fds); err != nil {
		logger.Fatalf("Failed to unmarshal descriptor set: %v", err)
	}
	logger.Println("Unmarshalled descriptor set.")

	for _, file := range fds.File {
		packageName := "." + file.GetPackage()

		// Build lookup maps for top-level messages and enums.
		for i, msg := range file.MessageType {
			fqName := packageName + "." + msg.GetName()
			topMsgPath := protoPath{4, int32(i)}

			if cpd, ok := customProtoDataMap[fqName]; ok {
				pIndex.addMessage(fqName, topMsgPath, file, msg, cpd)
			} else {
				pIndex.addMessage(fqName, topMsgPath, file, msg, nil)
			}

			//indexNestedMessages(fqName, msg, file, topMsgPath, pIndex)
			indexNestedMessages(*pIndex, fqName)

		}
		for i, enum := range file.EnumType {
			fqName := packageName + "." + enum.GetName()
			enumPath := protoPath{5, int32(i)}

			if cpd, ok := customProtoDataMap[fqName]; ok {
				pIndex.addEnum(fqName, enumPath, file, enum, cpd)
			} else {
				pIndex.addEnum(fqName, enumPath, file, enum, nil)
			}

		}
	}

	logger.Println("finish parsing descriptor set.")
	// Create or open the output file.
	outFile, err := os.Create(*outPath)
	sw := NewFileWriter(outFile, logger)
	if err != nil {
		logger.Fatalf("Failed to create output file: %v", err)
	}
	defer func(outFile *os.File) {
		err := outFile.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(outFile)

	entryPoints := findEntrypoints(pIndex.messageIndex)

	for i, ep := range entryPoints {
		if _, ok := pIndex.messageIndex[ep]; ok {
			printMessage(*pIndex, sw, ep, 0, make(map[string]bool))

			// make sure we have a blank line between top level messages
			if i != len(entryPoints)-1 {
				sw.WriteLine(0, "")
			}
		}
	}
	return 0
}

func main() {
	flag.Parse()

	if *descriptorSetPath != "" {
		// Standalone mode: read a .binpb descriptor set, index it, write one .fdhschema.
		_ = runStandalone(descriptorSetPath, outSchemaPath)
		return
	}

}
