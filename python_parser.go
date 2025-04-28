package main

import (
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

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

// --- Python Specific Grammar ---
// PythonParseResult holds the result of parsing a single line of Python output.
type PythonParseResult struct {
	FileRef *PythonFileRef   `( @@ EOL?`
	Error   *PythonErrorLine `| @@ EOL? )`
}

// Python parser instance
var pythonParser = participle.MustBuild[PythonParseResult](
	append(commonParserOptions, participle.UseLookahead(1))..., // Python grammar might be simpler
)
