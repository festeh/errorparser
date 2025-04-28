package main

import (
	"fmt"
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

// ErrorInfo holds the common structured information extracted from an error message.
// Use pointers for optional fields like Column.
type ErrorInfo struct {
	Filename string
	Line     int
	Column   *int // Optional column
	Type     string // Error, Warning, Panic, etc.
	Message  string // The actual error message text
}

// --- Custom Lexer ---
// Define custom lexer rules to handle file paths and specific error keywords.
var logLexer = lexer.MustSimple([]lexer.SimpleRule{
	{"Whitespace", `[ \t]+`},
	{"EOL", `[\n\r]+`}, // End of line
	{"Number", `\d+`},
	{"Word", `[a-zA-Z_][a-zA-Z0-9_]*`}, // Identifiers, keywords like Error, panic
	// Path needs to handle various characters including '/', '.', '-', '_', and drive letters C: etc.
	// Stop before ':' followed by a number (line number).
	{"Path", `(?:[a-zA-Z]:)?(?:[\\/]?[\w\.\-_]+)+`},
	{"PanicStart", `panic:`}, // Specific token for Go panics
	{"FileStart", `File "`},   // Specific token for Python File lines
	{"String", `"(\\"|[^"])*"`}, // Standard string literal for Python filenames
	{"Colon", `:`},
	{"Comma", `,`},
	{"Other", `.`}, // Catch any other single character
})

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

// --- Python Grammar ---
// Python errors often span multiple lines. We'll parse key lines individually.
// Example 1: File "/home/dima/projects/errorparser/gcd.py", line 1
type PythonFileRef struct {
	Filename string `@FileStart @Path "\""` // Use Path inside quotes
	Line     int    `"," "line" @Number`

	Pos lexer.Position `parser:"-"`
}

// Example 2: ModuleNotFoundError: No module named 'foowe'
// Example 3: SyntaxError: '(' was never closed
type PythonErrorLine struct {
	ErrType string `@Word` // e.g., ModuleNotFoundError, SyntaxError
	Message string `":" @Rest`

	Pos lexer.Position `parser:"-"`
}

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

// --- Combined Grammar ---
// LogEntry represents a potential line of log output.
// We try parsing it as one of the known error formats.
// Use @lexer.EOL to ensure we parse whole lines where appropriate.
type LogEntry struct {
	Flutter      *FlutterError    `( @@ EOL?` // Try Flutter format
	GoCompile    *GoCompileError  `| @@ EOL?` // Try Go compile error
	GoPanic      *GoPanic         `| @@ EOL?` // Try Go panic (first line)
	PythonFile   *PythonFileRef   `| @@ EOL?` // Try Python File line
	PythonError  *PythonErrorLine `| @@ EOL?` // Try Python Error line
	Unmatched    *string          `| @Rest)`  // Catch-all for lines that don't match known patterns
}

// --- Parser Setup ---

// Common parser options used by both main and fallback parsers
var commonParserOptions = []participle.Option{
	participle.Lexer(lexer.NewTextScannerLexer(logLexer)),
	participle.Elide("Whitespace"), // Ignore whitespace tokens between meaningful tokens
	// Define Rest token explicitly to capture remaining line content
	participle.Map(func(t lexer.Token) (lexer.Token, error) {
		// This mapping seems intended to ignore EOL unless explicitly matched,
		// but the current implementation doesn't modify the token.
		// Keeping it as is for now, assuming it serves a purpose or was WIP.
		// If EOL should be generally ignored, Elide("EOL") might be simpler.
		return t, nil
	}, "EOL"), // Apply mapping to EOL tokens
	participle.Unquote("String"), // Automatically unquote string literals
	participle.Capture("Rest", `.*`), // Define how to capture the 'Rest' of a line
}

// Main parser with lookahead 2
var parser = participle.MustBuild[LogEntry](
	append(commonParserOptions, participle.UseLookahead(2))...,
)

// Fallback parser for unmatched lines (simpler, less lookahead)
var fallbackParser = participle.MustBuild[LogEntry](
	append(commonParserOptions, participle.UseLookahead(1))...,
	// Note: Fallback specifically targets capturing the whole line as Unmatched,
	// so the grammar structure implies it might not need all common options,
	// but reusing them is simpler for now.
)


// Parse function
func ParseLogLine(line string) (*LogEntry, error) {
	// Ensure the line ends with a newline for consistent EOL handling
	if !strings.HasSuffix(line, "\n") {
		line += "\n"
	}
	entry, err := parser.ParseString("", line)
	if err != nil {
		// Attempt to parse just as Unmatched using the pre-built fallback parser
		entryFallback, errFallback := fallbackParser.ParseString("", line)
		// If fallback parsed *something* and it landed in Unmatched, return that.
		if errFallback == nil && entryFallback != nil && entryFallback.Unmatched != nil {
			// Trim trailing newline added earlier
			*entryFallback.Unmatched = strings.TrimSuffix(*entryFallback.Unmatched, "\n")
			return entryFallback, nil
			// Trim trailing newline added earlier
			*entry.Unmatched = strings.TrimSuffix(*entry.Unmatched, "\n")
			return entry, nil
		}
		// Otherwise, return the original parsing error
		return nil, fmt.Errorf("parsing error: %w on line: %s", err, line)
	}

	// Trim trailing newline from captured fields centrally
	trimSuffixNewline := func(s string) string {
		return strings.TrimSuffix(s, "\n")
	}

	if entry.Unmatched != nil {
		*entry.Unmatched = trimSuffixNewline(*entry.Unmatched)
	}
	if entry.Flutter != nil {
		entry.Flutter.Message = trimSuffixNewline(entry.Flutter.Message)
	}
	if entry.GoCompile != nil {
		entry.GoCompile.Message = trimSuffixNewline(entry.GoCompile.Message)
	}
	if entry.GoPanic != nil {
		entry.GoPanic.Message = trimSuffixNewline(entry.GoPanic.Message)
	}
	if entry.PythonError != nil {
		entry.PythonError.Message = trimSuffixNewline(entry.PythonError.Message)
	}

	return entry, nil
}

// Helper to extract the common ErrorInfo
// Note: Python parsing is basic and might require combining info from multiple lines.
func (l *LogEntry) GetErrorInfo() (ErrorInfo, bool) {
	if l.Flutter != nil {
		return l.Flutter.ToErrorInfo(), true
	}
	if l.GoCompile != nil {
		return l.GoCompile.ToErrorInfo(), true
	}
	if l.GoPanic != nil {
		// Go Panic parsing is simplified, may lack file/line info here
		return l.GoPanic.ToErrorInfo(), true
	}
	// Python requires context (previous File line) - this basic version won't have full info
	if l.PythonError != nil {
		// We only have the error type and message from this line
		return ErrorInfo{Type: l.PythonError.ErrType, Message: strings.TrimSpace(l.PythonError.Message)}, true
	}
	if l.PythonFile != nil {
		// This line only contains file info, not the error itself
		// We could potentially store this context for the next line.
		// For now, return false as it's not a complete error message.
		return ErrorInfo{Filename: l.PythonFile.Filename, Line: l.PythonFile.Line}, false // Indicate not a full error message
	}
	return ErrorInfo{}, false
}
