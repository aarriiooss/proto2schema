package writer

import (
	"fmt"
	"io"
	"log"
	"strings"
)

// SchemaWriter defines the interface for writing schema output
type SchemaWriter interface {
	// Writef write formatted line with indentation level
	Writef(level int, format string, args ...interface{})
	// WriteLine write raw line (with new line) with indentation level
	WriteLine(level int, line string)
	// Writefln adds a new line to Writef call
	Writefln(level int, format string, args ...interface{})
}

// CustomWriter implements SchemaWriter
type CustomWriter struct {
	writer io.Writer
	logger *log.Logger
}

// NewCustomWriter creates a new CustomWriter instance
func NewCustomWriter(w io.Writer, l *log.Logger) SchemaWriter {
	return &CustomWriter{
		writer: w,
		logger: l,
	}
}

// Writef writes formatted text with indentation
func (fw *CustomWriter) Writef(level int, format string, args ...interface{}) {
	indent := strings.Repeat("  ", level)
	_, err := fmt.Fprintf(fw.writer, indent+format, args...)
	if err != nil {
		fw.logger.Println(err)
	}
}

// WriteLine writes a line with indentation
func (fw *CustomWriter) WriteLine(level int, line string) {
	indent := strings.Repeat("  ", level)
	_, err := fmt.Fprintf(fw.writer, "%s%s\n", indent, line)
	if err != nil {
		fw.logger.Println(err)
	}
}

// Writefln writes formatted text with a newline
func (fw *CustomWriter) Writefln(level int, format string, args ...interface{}) {
	fw.Writef(level, format, args...)
	fw.WriteLine(0, "")
}

// OpenBlock writes an opening block with header
func OpenBlock(w SchemaWriter, level int, header string) {
	w.Writefln(level, "%s {", header)
}

// CloseBlock writes a closing block
func CloseBlock(w SchemaWriter, level int) {
	w.Writefln(level, "}")
}

// OpenList writes an opening list with header
func OpenList(w SchemaWriter, level int, header string) {
	w.Writefln(level, "%s [", header)
}

// CloseList writes a closing list
func CloseList(w SchemaWriter, level int) {
	w.Writefln(level, "]")
}

// OpenEnum writes an opening enum with header
func OpenEnum(w SchemaWriter, level int, header string) {
	w.Writefln(level, "%s (", header)
}

// CloseEnum writes a closing enum
func CloseEnum(w SchemaWriter, level int) {
	w.Writefln(level, ")")
}

// PrintCommentIfAny prints a comment if it's not empty
func PrintCommentIfAny(w SchemaWriter, comment string, level int) {
	if comment != "" {
		w.Writefln(level, "// %s", comment)
	}
}