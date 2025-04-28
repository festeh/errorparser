package main

import (
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

// --- Flutter Grammar ---
// Example: lib/main.dart:9:1: Error: Type 'oid' not found.
type FlutterError struct {
	Filename string `@Path`
	Line     int    `":" @Number`
	Column   int    `":" @Number`
	ErrType  string `":" @Word` // "Error", "Warning"
	Message  string `":" @Rest`

	Pos lexer.Position `parser:"-"`
}

func (e *FlutterError) ToErrorInfo() ErrorInfo {
	col := e.Column
	return ErrorInfo{
		Filename: e.Filename,
		Line:     e.Line,
		Column:   &col,
		Type:     e.ErrType,
		Message:  strings.TrimSpace(e.Message),
	}
}

// Flutter parser instance
var flutterParser = participle.MustBuild[FlutterError](commonParserOptions...)
