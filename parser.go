package main

import (
	"fmt"
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

// Language represents the type of log being parsed.
type Language int

const (
	LangUnknown Language = iota
	LangFlutter
	LangPython
	LangGo
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

// --- Python Specific Grammar ---
// PythonParseResult holds the result of parsing a single line of Python output.
type PythonParseResult struct {
	FileRef *PythonFileRef   `( @@ EOL?`
	Error   *PythonErrorLine `| @@ EOL? )`
}

// --- Go Specific Grammar ---
// GoParseResult holds the result of parsing a single line of Go output.
type GoParseResult struct {
	CompileError *GoCompileError `( @@ EOL?`
	Panic        *GoPanic        `| @@ EOL? )`
}

// --- Unmatched Line ---
// Represents a line that did not match the expected grammar for the selected language.
type UnmatchedLine struct {
	Content string `@Rest`
}


// --- Parser Setup ---

// Common parser options used across different language parsers
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

// Individual parsers for each language
var (
	flutterParser = participle.MustBuild[FlutterError](commonParserOptions...)

	pythonParser = participle.MustBuild[PythonParseResult](
		append(commonParserOptions, participle.UseLookahead(1))..., // Python grammar might be simpler
	)

	goParser = participle.MustBuild[GoParseResult](
		append(commonParserOptions, participle.UseLookahead(1))..., // Go grammar might be simpler
	)

	// A simple parser to capture the entire line as 'Rest' for unmatched cases
	unmatchedLineParser = participle.MustBuild[UnmatchedLine](
		participle.Lexer(lexer.NewTextScannerLexer(logLexer)), // Use the same lexer
		participle.Elide("Whitespace"),
		participle.Capture("Rest", `.*`),
	)
)


// ParseLine parses a single line of text based on the provided language context.
// It returns the specific parsed struct (e.g., *FlutterError), *UnmatchedLine, or an error.
func ParseLine(line string, lang Language) (interface{}, error) {
	// Ensure the line ends with a newline for consistent EOL handling within grammars using EOL?
	// Although, with separate parsers, EOL might be less critical unless explicitly needed.
	// Let's keep it for now as grammars use EOL?.
	if !strings.HasSuffix(line, "\n") {
		line += "\n"
	}

	var result interface{}
	var err error

	switch lang {
	case LangFlutter:
		parsed := &FlutterError{}
		err = flutterParser.ParseString("", line, parsed)
		if err == nil {
			// Trim newline from message after successful parse
			parsed.Message = strings.TrimSuffix(parsed.Message, "\n")
			result = parsed
		}
	case LangPython:
		parsed := &PythonParseResult{}
		err = pythonParser.ParseString("", line, parsed)
		if err == nil {
			// Trim newline from message if PythonErrorLine was parsed
			if parsed.Error != nil {
				parsed.Error.Message = strings.TrimSuffix(parsed.Error.Message, "\n")
			}
			result = parsed
		}
	case LangGo:
		parsed := &GoParseResult{}
		err = goParser.ParseString("", line, parsed)
		if err == nil {
			// Trim newline from message after successful parse
			if parsed.CompileError != nil {
				parsed.CompileError.Message = strings.TrimSuffix(parsed.CompileError.Message, "\n")
			}
			if parsed.Panic != nil {
				parsed.Panic.Message = strings.TrimSuffix(parsed.Panic.Message, "\n")
			}
			result = parsed
		}
	default:
		return nil, fmt.Errorf("unknown language specified for parsing")
	}

	// If parsing for the specific language failed, try parsing as an UnmatchedLine
	if err != nil {
		// Use the original line without the potentially added newline for UnmatchedLine parsing
		originalLine := strings.TrimSuffix(line, "\n")
		unmatched := &UnmatchedLine{}
		// Use the dedicated unmatchedLineParser
		errUnmatched := unmatchedLineParser.ParseString("", originalLine, unmatched)
		if errUnmatched == nil {
			// Successfully parsed as unmatched, return this instead of the original error
			return unmatched, nil
		}
		// If even unmatched parsing failed (should be rare), return the original language parse error
		return nil, fmt.Errorf("parsing error for lang %v: %w (line: %s)", lang, err, originalLine)
	}

	return result, nil
}

// Note: The GetErrorInfo helper function is removed.
// Logic for converting parsed structs to ErrorInfo will now reside in main.go,
// allowing for context-specific handling (like Python's multi-line errors).
