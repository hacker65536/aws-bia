/*
Copyright © 2025 AWS-BIA Contributors

This file contains the StreamProcessor implementation for the AWS Bedrock Intelligent Agents CLI.
It handles the processing of event streams from AWS Bedrock agents, including text chunks,
file outputs, and control events.
*/
package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime/types"
)

// StreamProcessor handles processing of event streams from AWS Bedrock agents
type StreamProcessor struct {
	Options     AgentOptions
	Writer      io.Writer
	WriteOutput bool
}

// NewStreamProcessor creates a new StreamProcessor
func NewStreamProcessor(opts AgentOptions, writer io.Writer, writeOutput bool) *StreamProcessor {
	return &StreamProcessor{
		Options:     opts,
		Writer:      writer,
		WriteOutput: writeOutput,
	}
}

// ProcessStream processes an event stream and returns the collected content.
// This is a helper function to avoid code duplication between streaming and non-streaming handling.
func (sp *StreamProcessor) ProcessStream(stream *bedrockagentruntime.InvokeAgentEventStream) (
	string, []types.Citation, []types.OutputFile, bool, error) {

	var textResponse strings.Builder
	var citations []types.Citation
	var outputFiles []types.OutputFile
	var hasReturnControl bool

	// Process the streaming response
	for event := range stream.Events() {
		if sp.Options.Verbose {
			logVerbose(sp.Options, "Processing event type: %T", event)
		}

		switch v := event.(type) {
		case *types.ResponseStreamMemberChunk:
			// This is a text chunk from the agent
			if len(v.Value.Bytes) > 0 {
				chunk := string(v.Value.Bytes)
				textResponse.WriteString(chunk)

				// Write the output if requested (for streaming mode or text format)
				if sp.WriteOutput && sp.Options.OutputFormat == "text" {
					fmt.Fprint(sp.Writer, chunk)
				}
			}

			// Process citations if available
			if v.Value.Attribution != nil && len(v.Value.Attribution.Citations) > 0 {
				citations = append(citations, v.Value.Attribution.Citations...)
			}

		case *types.ResponseStreamMemberFiles:
			// Handle file output
			if len(v.Value.Files) > 0 {
				outputFiles = append(outputFiles, v.Value.Files...)

				if sp.WriteOutput && sp.Options.OutputFormat == "text" {
					fmt.Fprintf(sp.Writer, "\n\n[Generated %d file(s)]\n", len(v.Value.Files))
					for i, file := range v.Value.Files {
						fmt.Fprintf(sp.Writer, "  %d. %s", i+1, *file.Name)
						if file.Type != nil {
							fmt.Fprintf(sp.Writer, " (type: %s)", *file.Type)
						}
						fmt.Fprintf(sp.Writer, " (%d bytes)\n", len(file.Bytes))
					}
				}
			}

		case *types.ResponseStreamMemberTrace:
			// Just log trace events in verbose mode
			if sp.Options.Verbose {
				logVerbose(sp.Options, "Received trace event")
			}

		case *types.ResponseStreamMemberReturnControl:
			// When agent returns control (for custom control flows)
			hasReturnControl = true
			if sp.WriteOutput && sp.Options.OutputFormat == "text" {
				fmt.Fprintln(sp.Writer, "\n[Agent returned control]")
				if v.Value.InvocationId != nil {
					fmt.Fprintf(sp.Writer, "Invocation ID: %s\n", *v.Value.InvocationId)
				}
				if v.Value.InvocationInputs != nil {
					fmt.Fprintf(sp.Writer, "Invocation inputs: %d item(s)\n", len(v.Value.InvocationInputs))
				}
			}

		default:
			if sp.Options.Verbose {
				logVerbose(sp.Options, "Unknown event type: %T", event)
			}
		}
	}

	// Check for any errors that occurred during streaming
	if err := stream.Err(); err != nil {
		return textResponse.String(), citations, outputFiles, hasReturnControl,
			handleAWSError(fmt.Errorf("error during streaming: %w", err))
	}

	return textResponse.String(), citations, outputFiles, hasReturnControl, nil
}
