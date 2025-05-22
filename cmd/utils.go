/*
Copyright © 2025 AWS-BIA Contributors

This file implements helper utilities shared across the CLI commands.
*/
package cmd

import (
	"fmt"
	"os"
)

// logVerbose logs a message if verbose mode is enabled
func logVerbose(opts AgentOptions, format string, args ...interface{}) {
	if opts.Verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] "+format+"\n", args...)
	}
}

// logError logs an error message
func logError(message string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", message, err)
}

// For backward compatibility with existing code
var (
	handleAWSError = HandleAWSError
)
