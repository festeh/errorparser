package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Enter log lines (Ctrl+D to end):")

	var lastPythonFileRef *PythonFileRef // Holds context between lines for Python errors

	for scanner.Scan() {
		line := scanner.Text()
		currentPythonFileRef := lastPythonFileRef // Preserve ref from previous line for this iteration
		lastPythonFileRef = nil                   // Reset context for the *next* iteration by default

		if line == "" {
			continue
		}

		entry, err := ParseLogLine(line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Parser internal error: %v\n", err)
			// Print the raw line if parsing fails completely
			fmt.Printf("Raw Line: %s\n", line)
			lastPythonFileRef = nil // Reset context on error
			continue
		}

		// --- Handle Parsed Entry ---
		info, isCompleteError := entry.GetErrorInfo()

		if isCompleteError {
			// If it's a Python error line, try to combine with the context from the previous line
			if entry.PythonError != nil && currentPythonFileRef != nil {
				info.Filename = currentPythonFileRef.Filename
				info.Line = currentPythonFileRef.Line
				fmt.Printf("Parsed Error (Python Context): %+v\n", info)
			} else {
				// Print standard parsed errors (Flutter, Go, Go Panic, or Python without context)
				fmt.Printf("Parsed Error: %+v\n", info)
			}
		} else if entry.PythonFile != nil {
			// Store Python File context for the *next* line
			lastPythonFileRef = entry.PythonFile // Override the default nil reset
			fmt.Printf("Context (Python File): %s, Line %d\n", entry.PythonFile.Filename, entry.PythonFile.Line)
		} else if entry.Unmatched != nil {
			// Print lines that didn't match any specific error pattern
			fmt.Printf("Unmatched Line: %s\n", *entry.Unmatched)
		} else {
			// This case means parsing succeeded structurally but didn't match known types
			// and wasn't caught by Unmatched. Could indicate a grammar issue or an unhandled LogEntry field.
			fmt.Printf("Parsed but Unrecognized Structure: %+v\n", entry)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}
}
