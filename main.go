package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	// --- Language Selection via Flag ---
	langFlag := flag.String("lang", "", "The language of the log output (flutter, python, go)")
	flag.Parse()

	var selectedLang Language
	switch strings.ToLower(*langFlag) {
	case "flutter":
		selectedLang = LangFlutter
	case "python":
		selectedLang = LangPython
	case "go":
		selectedLang = LangGo
	default:
		fmt.Fprintf(os.Stderr, "Error: Invalid or missing -lang flag. Please specify flutter, python, or go.\n")
		os.Exit(1)
	}

	// --- Input Processing ---
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Printf("Parsing for language: %s. Enter log lines (Ctrl+D to end):\n", *langFlag)

	var lastPythonFileRef *PythonFileRef // Holds context between lines specifically for Python errors

	for scanner.Scan() {
		line := scanner.Text()
		currentPythonFileRef := lastPythonFileRef // Preserve ref from previous line for this iteration (Python only)
		if selectedLang != LangPython {
			currentPythonFileRef = nil // Not needed for other languages
		}
		lastPythonFileRef = nil // Reset context for the *next* iteration by default

		if line == "" {
			continue
		}

		// --- Parsing ---
		parsedResult, err := ParseLine(line, selectedLang)
		if err != nil {
			// ParseLine now tries to return UnmatchedLine instead of error for non-matching lines.
			// An error here indicates a more fundamental parsing issue or unknown language.
			fmt.Fprintf(os.Stderr, "Parser internal error: %v\n", err)
			continue
		}

		// --- Handle Parsed Result ---
		switch v := parsedResult.(type) {
		case *FlutterError:
			info := v.ToErrorInfo()
			fmt.Printf("Parsed Error (Flutter): %+v\n", info)
		case *GoParseResult:
			if v.CompileError != nil {
				info := v.CompileError.ToErrorInfo()
				fmt.Printf("Parsed Error (Go Compile): %+v\n", info)
			} else if v.Panic != nil {
				info := v.Panic.ToErrorInfo()
				fmt.Printf("Parsed Error (Go Panic): %+v\n", info)
			} else {
				// Should not happen if parser logic is correct
				fmt.Printf("Parsed Go Structure (Empty): %+v\n", v)
			}
		case *PythonParseResult:
			if v.FileRef != nil {
				// Store Python File context for the *next* line
				lastPythonFileRef = v.FileRef // Override the default nil reset
				fmt.Printf("Context (Python File): %s, Line %d\n", v.FileRef.Filename, v.FileRef.Line)
			} else if v.Error != nil {
				// Construct ErrorInfo for the Python error line
				info := ErrorInfo{
					Type:    v.Error.ErrType,
					Message: strings.TrimSpace(v.Error.Message),
				}
				// Combine with context from the previous line if available
				if currentPythonFileRef != nil {
					info.Filename = currentPythonFileRef.Filename
					info.Line = currentPythonFileRef.Line
					fmt.Printf("Parsed Error (Python Context): %+v\n", info)
				} else {
					// Print Python error without file context
					fmt.Printf("Parsed Error (Python): %+v\n", info)
				}
			} else {
				// Should not happen if parser logic is correct
				fmt.Printf("Parsed Python Structure (Empty): %+v\n", v)
			}
		case *UnmatchedLine:
			// Print lines that didn't match the specific language's error patterns
			fmt.Printf("Unmatched Line: %s\n", v.Content)
		default:
			// This case should ideally not be reached if ParseLine handles all types
			fmt.Printf("Parsed but Unrecognized Type: %T %+v\n", v, v)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}
}
