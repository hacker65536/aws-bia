/*
Copyright © 2025 AWS-BIA Contributors

This file implements helper utilities shared across the CLI commands.
*/
package cmd

// logVerbose logs a message if verbose mode is enabled
// This function is now a wrapper around LogVerbose for backward compatibility
func logVerbose(opts AgentOptions, format string, args ...interface{}) {
	LogVerbose(opts, format, args...)
}

// logError logs an error message
// This function is now a wrapper around LogError for backward compatibility
func logError(message string, err error) {
	LogError(message, err)
}

// For backward compatibility with existing code
var (
	handleAWSError = HandleAWSError
)
