package types

// CustomProtoDataMap contains predefined custom metadata for specific proto types
var CustomProtoDataMap = map[string]*CustomProtoData{
	".google.protobuf.Timestamp": NewCustomProtoData(
		"In JSON format, the Timestamp type is encoded as a string in the RFC 3339 format.",
		"",
		"",
	),
}
