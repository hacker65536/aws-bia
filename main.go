/*
Copyright © 2025 AWS-BIA Contributors

This program provides a command-line interface for interacting with AWS Bedrock
Intelligent Agents. It allows users to invoke Bedrock agents with text inputs and
receive responses in both streaming and non-streaming modes.

The tool supports multiple output formats (text and JSON), saving responses to files,
and handling of all response types including text, citations, and generated files.
*/
package main

import (
	"time"

	"github.com/carlmjohnson/versioninfo"
	"github.com/hacker65536/aws-bia/cmd"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if version == "dev" {
		version = versioninfo.Version
		commit = versioninfo.Revision
		date = versioninfo.LastCommit.Format(time.RFC3339)
	} else {
		// Do not add the 'v' prefix if it is already present in the version string
		if len(version) == 0 || version[0] != 'v' {
			version = "v" + version
		}
	}
	cmd.SetVersionInfo(version, commit, date)
	cmd.Execute()
}
