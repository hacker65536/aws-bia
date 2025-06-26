/*
Copyright © 2025 AWS-BIA Contributors

This file contains utilities for formatting and displaying responses in the AWS Bedrock Intelligent Agents CLI.
It handles different output formats (text and JSON) and manages the display of streaming responses.
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime/types"
)

// ResponseFormatter handles formatting and output of agent responses
type ResponseFormatter struct {
	Options        AgentOptions
	Writer         io.Writer
	FileHelper     *FileHelper
	isJSONFormat   bool // Cache format check
	hasUploadFiles bool // Cache upload files check
}

// NewResponseFormatter creates a new ResponseFormatter
func NewResponseFormatter(opts AgentOptions, writer io.Writer) *ResponseFormatter {
	return &ResponseFormatter{
		Options:        opts,
		Writer:         writer,
		FileHelper:     NewFileHelper(opts),
		isJSONFormat:   opts.OutputFormat == "json",
		hasUploadFiles: len(opts.UploadFiles) > 0,
	}
}

// FormatAndWriteResponse formats the response based on the output format and writes it to the writer
func (rf *ResponseFormatter) FormatAndWriteResponse(output *bedrockagentruntime.InvokeAgentOutput) error {
	if rf.isJSONFormat {
		return rf.writeJSONResponse(output)
	}
	return rf.writeTextResponse(output)
}

// writeTextResponse formats the response as text and writes it to the writer
func (rf *ResponseFormatter) writeTextResponse(output *bedrockagentruntime.InvokeAgentOutput) error {
	// Write header
	fmt.Fprintln(rf.Writer, "Agent Response:")

	// Show uploaded files info if any
	if rf.hasUploadFiles {
		fmt.Fprintf(rf.Writer, "[Uploaded %d file(s) to agent]\n", len(rf.Options.UploadFiles))
		for i, file := range rf.Options.UploadFiles {
			baseName := filepath.Base(file)
			if fileInfo, err := fileInfo(file); err == nil {
				fmt.Fprintf(rf.Writer, "  %d. %s (%.2f KB)\n", i+1, baseName, float64(fileInfo.Size())/1024)
			} else {
				fmt.Fprintf(rf.Writer, "  %d. %s\n", i+1, baseName)
			}
		}
		fmt.Fprintln(rf.Writer)
	}

	// Process the event stream if available
	stream := output.GetStream()
	if stream != nil {
		// Process the stream and write output in real-time
		processor := NewStreamProcessor(rf.Options, rf.Writer, true)
		_, citations, outputFiles, _, err := processor.ProcessStream(stream)
		if err != nil {
			return err
		}

		// Save any generated files if specified
		if len(outputFiles) > 0 && rf.Options.FilesOutputDir != "" {
			savedFiles, err := rf.FileHelper.HandleFileOutput(outputFiles)
			if err != nil {
				logError("Warning: Error saving files", err)
			} else if len(savedFiles) > 0 {
				fmt.Fprintf(rf.Writer, "\n[Saved %d files to %s]\n", len(savedFiles), rf.Options.FilesOutputDir)
				for i, file := range savedFiles {
					fmt.Fprintf(rf.Writer, "  %d. %s\n", i+1, file)
				}
			}
		}

		// Print session ID if returned
		rf.writeSessionInfo(output)

		// Print citation information if available
		rf.writeCitationsTextOutput(citations)
	} else {
		fmt.Fprintln(rf.Writer, "[No response content available]")
		rf.writeSessionInfo(output)
	}

	return nil
}

// writeJSONResponse formats the response as JSON and writes it to the writer
func (rf *ResponseFormatter) writeJSONResponse(output *bedrockagentruntime.InvokeAgentOutput) error {
	// Process stream content if available
	var textContent string
	var citations []types.Citation
	var outputFiles []types.OutputFile
	var hasReturnControl bool

	stream := output.GetStream()
	if stream != nil {
		processor := NewStreamProcessor(rf.Options, rf.Writer, false)
		var err error
		textContent, citations, outputFiles, hasReturnControl, err = processor.ProcessStream(stream)
		if err != nil {
			return err
		}
	}

	// Save any generated files if specified in the options
	var savedFiles []string
	if len(outputFiles) > 0 && rf.Options.FilesOutputDir != "" {
		var err error
		savedFiles, err = rf.FileHelper.HandleFileOutput(outputFiles)
		if err != nil {
			logError("Warning: Error saving files", err)
		} else if len(savedFiles) > 0 {
			logVerbose(rf.Options, "Saved %d files to %s", len(savedFiles), rf.Options.FilesOutputDir)
		}
	}

	// Create the base response
	response := map[string]interface{}{
		"content":          textContent,
		"wasStreamingUsed": rf.Options.EnableStreaming,
		"timestamp":        time.Now().Format(time.RFC3339),
	}

	// Add metadata to the response
	rf.addResponseMetadata(response, output, citations, outputFiles, hasReturnControl)

	// Add saved files information if any
	if len(savedFiles) > 0 {
		response["savedFiles"] = savedFiles
	}

	// Marshal and write the JSON response
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal response to JSON: %w", err)
	}

	// Write the JSON to the writer
	_, err = rf.Writer.Write(jsonData)
	return err
}

// addResponseMetadata adds metadata fields to the JSON response
func (rf *ResponseFormatter) addResponseMetadata(
	response map[string]interface{},
	output *bedrockagentruntime.InvokeAgentOutput,
	citations []types.Citation,
	files []types.OutputFile,
	hasReturnControl bool) {

	// Add content type
	if output.ContentType != nil {
		response["contentType"] = output.ContentType
	} else {
		response["contentType"] = "text/plain" // Default content type
	}

	// Add session information
	if output.SessionId != nil {
		response["sessionId"] = output.SessionId
		response["generatedSessionId"] = rf.Options.SessionID == ""
	}

	// Add memory ID if available
	if output.MemoryId != nil {
		response["memoryId"] = output.MemoryId
	}

	// Add uploaded files information if any
	if rf.hasUploadFiles {
		uploadedFiles, err := rf.FileHelper.FormatUploadedFilesInfo()
		if err == nil && len(uploadedFiles) > 0 {
			response["uploadedFiles"] = uploadedFiles
		}
	}

	// Add citations if available
	if len(citations) > 0 {
		response["citations"] = rf.formatCitationsForJSON(citations)
	}

	// Add files if available
	if len(files) > 0 {
		// Include metadata about files but not the binary content
		fileInfos := make([]map[string]interface{}, 0, len(files))
		for _, file := range files {
			fileInfo := map[string]interface{}{
				"name": *file.Name,
				"size": len(file.Bytes),
			}
			if file.Type != nil {
				fileInfo["type"] = *file.Type
			}
			fileInfos = append(fileInfos, fileInfo)
		}
		response["files"] = fileInfos
	}

	// Add control return info if available
	if hasReturnControl {
		response["returnedControl"] = true
	}
}

// formatCitationsForJSON formats citations for JSON output
func (rf *ResponseFormatter) formatCitationsForJSON(citations []types.Citation) []map[string]interface{} {
	if len(citations) == 0 {
		return nil
	}

	formattedCitations := make([]map[string]interface{}, 0, len(citations))
	for _, citation := range citations {
		citationInfo := make(map[string]interface{})

		// Check for generated response part text
		if citation.GeneratedResponsePart != nil &&
			citation.GeneratedResponsePart.TextResponsePart != nil &&
			citation.GeneratedResponsePart.TextResponsePart.Text != nil {
			citationInfo["text"] = *citation.GeneratedResponsePart.TextResponsePart.Text
		}

		// Process retrieved references
		if numRefs := len(citation.RetrievedReferences); numRefs > 0 {
			refs := make([]map[string]interface{}, 0, numRefs)
			for _, ref := range citation.RetrievedReferences {
				refInfo := make(map[string]interface{})

				if ref.Location != nil {
					refInfo["locationType"] = ref.Location.Type
				}

				if ref.Content != nil && ref.Content.Text != nil {
					refInfo["contentText"] = *ref.Content.Text
				}

				if len(refInfo) > 0 {
					refs = append(refs, refInfo)
				}
			}
			if len(refs) > 0 {
				citationInfo["references"] = refs
			}
		}

		if len(citationInfo) > 0 {
			formattedCitations = append(formattedCitations, citationInfo)
		}
	}
	return formattedCitations
}

// writeSessionInfo writes session information in text format
func (rf *ResponseFormatter) writeSessionInfo(output *bedrockagentruntime.InvokeAgentOutput) {
	// Print session ID if returned
	if output.SessionId != nil {
		fmt.Fprintf(rf.Writer, "\n\nSession ID: %s (Use this ID for follow-up questions)\n", *output.SessionId)
	}

	// Print content type
	if output.ContentType != nil {
		fmt.Fprintf(rf.Writer, "Content Type: %s\n", *output.ContentType)
	}

	// Print memory ID if any
	if output.MemoryId != nil {
		fmt.Fprintf(rf.Writer, "Memory ID: %s\n", *output.MemoryId)
	}
}

// writeCitationsTextOutput writes citation information in text format
func (rf *ResponseFormatter) writeCitationsTextOutput(citations []types.Citation) {
	if len(citations) == 0 {
		return
	}

	fmt.Fprintln(rf.Writer, "\nCitations:")
	for i, citation := range citations {
		fmt.Fprintf(rf.Writer, "  %d. ", i+1)
		if citation.GeneratedResponsePart != nil &&
			citation.GeneratedResponsePart.TextResponsePart != nil &&
			citation.GeneratedResponsePart.TextResponsePart.Text != nil {
			fmt.Fprintf(rf.Writer, "Text: %s", *citation.GeneratedResponsePart.TextResponsePart.Text)
		}
		if len(citation.RetrievedReferences) > 0 {
			for j, ref := range citation.RetrievedReferences {
				fmt.Fprintf(rf.Writer, "\n     Ref %d:", j+1)

				if ref.Location != nil {
					fmt.Fprintf(rf.Writer, " Type: %s", ref.Location.Type)
				}

				if ref.Content != nil && ref.Content.Text != nil {
					fmt.Fprintf(rf.Writer, ", Text: %s", *ref.Content.Text)
				}
			}
		}
		fmt.Fprintln(rf.Writer)
	}
}

// Helper function to get file info
func fileInfo(filePath string) (os.FileInfo, error) {
	return os.Stat(filePath)
}
