package main

import (
	"flag"
	"os"

	"proto2schema/internal/app"
)

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
		"path to the .fdhschema file to write when using --descriptor_set.",
	)
)

func main() {
	flag.Parse()

	if *descriptorSetPath != "" {
		// Standalone mode: read a .binpb descriptor set, index it, write one .fdhschema.
		// Unlike plugin mode, all top level schemas regardless of which file they came
		// from will be output in the same file
		app.RunStandalone(*descriptorSetPath, *outSchemaPath)
		return
	}

	app.RunAsPlugin(os.Stdin, os.Stdout)
}