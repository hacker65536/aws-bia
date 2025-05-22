/*
Copyright © 2025 AWS-BIA Contributors

This program provides a command-line interface for interacting with AWS Bedrock
Intelligent Agents. It allows users to invoke Bedrock agents with text inputs and
receive responses in both streaming and non-streaming modes.

The tool supports multiple output formats (text and JSON), saving responses to files,
and handling of all response types including text, citations, and generated files.
*/
package main

import "github.com/hacker65536/aws-bia/cmd"

func main() {
	cmd.Execute()
}
