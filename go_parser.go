package main

import (
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

// --- Go Grammar ---
// Example 1: main.go:1:1: expected 'package', found 'EOF'
// Example 2: ./main.go:4:2: undefined: fmt
type GoCompileError struct {
	Filename string `@Path`
	Line     int    `":" @Number`
	Column   int    `":" @Number`
	Message  string `":" @Rest`

	Pos lexer.Position `parser:"-"`
}

func (e *GoCompileError) ToErrorInfo() ErrorInfo {
	col := e.Column
	return ErrorInfo{
		Filename: e.Filename,
		Line:     e.Line,
		Column:   &col,
		Type:     "Error", // Go compiler errors are typically just "Error"
		Message:  strings.TrimSpace(e.Message),
	}
}

// Example 3: panic: runtime error: integer divide by zero
// Followed by stack trace, e.g., /home/dima/projects/errorparser/main.go:9 +0x8d
type GoPanic struct {
	Message   string `@PanicStart @Rest` // Capture message after "panic:"
	StackFile string `(@Path ":")?`      // Optional stack file line (simplified)
	StackLine int    `@Number?`          // Optional stack line number

	Pos lexer.Position `parser:"-"`
}

func (e *GoPanic) ToErrorInfo() ErrorInfo {
	info := ErrorInfo{
		Type:    "Panic",
		Message: strings.TrimSpace(e.Message),
	}
	// Add file/line if parsed from stack (very basic parsing)
	if e.StackFile != "" {
		info.Filename = e.StackFile
	}
	if e.StackLine > 0 {
		info.Line = e.StackLine
	}
	return info
}

// --- Go Specific Grammar ---
// GoParseResult holds the result of parsing a single line of Go output.
type GoParseResult struct {
	CompileError *GoCompileError `( @@ EOL?`
	Panic        *GoPanic        `| @@ EOL? )`
}

// Go parser instance
var goParser = participle.MustBuild[GoParseResult](
	append(commonParserOptions, participle.UseLookahead(1))..., // Go grammar might be simpler
)
