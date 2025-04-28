package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Enter log lines (Ctrl+D to end):")

	// Variable to potentially hold context from previous lines (e.g., Python File line)
	var lastPythonFileRef *PythonFileRef

	for scanner.Scan() {
		line := scanner.Text()
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
			// If it's a Python error line, try to combine with previous FileRef
			if entry.PythonError != nil && lastPythonFileRef != nil {
				info.Filename = lastPythonFileRef.Filename
				info.Line = lastPythonFileRef.Line
				fmt.Printf("Parsed Error (Python Context): %+v\n", info)
			} else {
				// Print standard parsed errors
				fmt.Printf("Parsed Error: %+v\n", info)
			}
			lastPythonFileRef = nil // Reset context after using it
		} else if entry.PythonFile != nil {
			// Store Python File context for the next line
			lastPythonFileRef = entry.PythonFile
			fmt.Printf("Context (Python File): %s, Line %d\n", entry.PythonFile.Filename, entry.PythonFile.Line)
		} else if entry.Unmatched != nil {
			// Print lines that didn't match any specific error pattern
			fmt.Printf("Unmatched Line: %s\n", *entry.Unmatched)
			lastPythonFileRef = nil // Reset context on unmatched lines
		} else {
			// This case means parsing succeeded structurally but didn't match known types
			// and wasn't caught by Unmatched. Could indicate a grammar issue.
			fmt.Printf("Parsed but Unrecognized Structure: %+v\n", entry)
			lastPythonFileRef = nil // Reset context
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}
}
