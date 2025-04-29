package main

import (
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

// --- Rust Grammar ---
// Example 1: error[E0308]: mismatched types
// Example 2:  --> src/main.rs:5:5
// Example 3: warning: unused variable: `x`
// We'll focus on parsing the main error/warning line and the location line.
// Other lines (like notes, help) will likely be treated as Unmatched.

// RustError captures the primary information from a Rust compiler error or warning line.
type RustMsgLine struct {
	Level    string     `@("error" | "warning")` // "error" or "warning"
	Code     *string    `( LBracket @ErrorCode RBracket )?` // Optional error code like [E0308]
	Message  string     `":" @Rest EOL?`
	Location *RustLocation `( @@ )?` // Optional location line immediately following

	Pos lexer.Position `parser:"-"`
}

// RustLocation captures the file path, line, and column.
type RustLocation struct {
	Filename string `@Arrow @Path`
	Line     int    `":" @Number`
	Column   int    `":" @Number`

	Pos lexer.Position `parser:"-"`
}


// ToErrorInfo converts a parsed RustMsgLine into the common ErrorInfo format.
func (e *RustMsgLine) ToErrorInfo() ErrorInfo {
	info := ErrorInfo{
		Type:    strings.Title(e.Level), // Capitalize "error" -> "Error", "warning" -> "Warning"
		Message: strings.TrimSpace(e.Message),
	}
	if e.Code != nil {
		// Optionally include the code in the message or a separate field if ErrorInfo is extended
		info.Message = "[" + *e.Code + "] " + info.Message
	}
	if e.Location != nil {
		info.Filename = e.Location.Filename
		info.Line = e.Location.Line
		col := e.Location.Column // Assign to temp var to take address
		info.Column = &col
	}
	return info
}

// Rust parser instance - attempts to parse a RustMsgLine
// Note: This parser expects the error/warning and its optional location on consecutive lines
// or combined if the grammar allows. The current grammar assumes they might appear together
// or just the message line appears. Handling multi-line context might require adjustments
// in main.go similar to Python, or a more complex grammar.
// For simplicity, we'll parse the RustMsgLine which *can* include the location.
var rustParser = participle.MustBuild[RustMsgLine](
	append(commonParserOptions, participle.UseLookahead(2))..., // Lookahead might be needed
)
