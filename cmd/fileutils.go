/*
Copyright © 2025 AWS-BIA Contributors

This file contains utilities for file operations in the AWS Bedrock Intelligent Agents CLI.
It handles file upload, MIME type detection, and file output operations.
*/
package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime/types"
)

// FileHelper provides methods for file-related operations
type FileHelper struct {
	Options         AgentOptions
	hasUploadFiles  bool // Cache upload files check
	uploadFileCount int  // Cache count
}

// NewFileHelper creates a new FileHelper with the given options
func NewFileHelper(opts AgentOptions) *FileHelper {
	uploadCount := len(opts.UploadFiles)
	return &FileHelper{
		Options:         opts,
		hasUploadFiles:  uploadCount > 0,
		uploadFileCount: uploadCount,
	}
}

// PrepareInputFiles processes file paths from options and prepares them for upload
func (f *FileHelper) PrepareInputFiles() ([]types.InputFile, error) {
	if !f.hasUploadFiles {
		return nil, nil
	}

	// Pre-allocate with exact capacity
	inputFiles := make([]types.InputFile, 0, f.uploadFileCount)
	var totalSize int64
	const maxSize = 10 * 1024 * 1024 // 10MB limit

	for _, filePath := range f.Options.UploadFiles {
		// Get file info
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get file info for '%s': %w", filePath, err)
		}

		// Update total size and check early
		totalSize += fileInfo.Size()
		if totalSize > maxSize {
			return nil, fmt.Errorf("total upload file size exceeds 10MB limit (got %.2fMB)",
				float64(totalSize)/(1024*1024))
		}

		// Read the file content
		fileContent, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file '%s': %w", filePath, err)
		}

		// Cache base name to avoid repeated calls
		baseName := filepath.Base(filePath)

		// Detect MIME type
		mimeType := DetectMimeType(filePath, fileContent)

		// Create the input file
		inputFile := types.InputFile{
			Name: aws.String(baseName),
			Source: &types.FileSource{
				SourceType: types.FileSourceTypeByteContent,
				ByteContent: &types.ByteContentFile{
					Data:      fileContent,
					MediaType: aws.String(mimeType),
				},
			},
			UseCase: types.FileUseCase(f.Options.FileUseCase),
		}

		inputFiles = append(inputFiles, inputFile)
		logVerbose(f.Options, "Added file '%s' for upload (type: %s, size: %d bytes)",
			baseName, mimeType, len(fileContent))
	}

	return inputFiles, nil
}

// HandleFileOutput processes agent-generated files and optionally saves them to disk
func (f *FileHelper) HandleFileOutput(files []types.OutputFile) ([]string, error) {
	if len(files) == 0 || f.Options.FilesOutputDir == "" {
		return nil, nil
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(f.Options.FilesOutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory '%s': %w", f.Options.FilesOutputDir, err)
	}

	savedFiles := make([]string, 0, len(files))
	for i, file := range files {
		if file.Name == nil {
			continue // Skip files without names
		}

		// Clean the filename to avoid path traversal attacks
		safeFileName := filepath.Base(*file.Name)
		outputPath := filepath.Join(f.Options.FilesOutputDir, safeFileName)

		// Add index to filename if it already exists
		if _, err := os.Stat(outputPath); err == nil {
			ext := filepath.Ext(safeFileName)
			base := strings.TrimSuffix(safeFileName, ext)
			outputPath = filepath.Join(f.Options.FilesOutputDir, fmt.Sprintf("%s_%d%s", base, i+1, ext))
		}

		// Write the file
		if err := os.WriteFile(outputPath, file.Bytes, 0644); err != nil {
			return savedFiles, fmt.Errorf("failed to save file '%s': %w", outputPath, err)
		}

		savedFiles = append(savedFiles, outputPath)
	}

	return savedFiles, nil
}

// FormatUploadedFilesInfo formats uploaded files information for output
func (f *FileHelper) FormatUploadedFilesInfo() ([]map[string]interface{}, error) {
	if !f.hasUploadFiles {
		return nil, nil
	}

	uploadedFiles := make([]map[string]interface{}, 0, f.uploadFileCount)
	for _, file := range f.Options.UploadFiles {
		if fileInfo, err := os.Stat(file); err == nil {
			uploadedFiles = append(uploadedFiles, map[string]interface{}{
				"name": filepath.Base(file),
				"size": fileInfo.Size(),
				"path": file,
			})
		}
	}
	return uploadedFiles, nil
}

// PrepareOutput sets up the output destination based on the options
func PrepareOutput(outputFile string) (io.Writer, func(), error) {
	if outputFile == "" {
		return os.Stdout, nil, nil
	}

	// Check if the directory exists
	dir := filepath.Dir(outputFile)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// Try to create the directory if it doesn't exist
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, nil, fmt.Errorf("failed to create directory for output file: %w", err)
		}
	}

	// Try to open the file
	file, err := os.Create(outputFile)
	if err != nil {
		if os.IsPermission(err) {
			return nil, nil, fmt.Errorf("permission denied when creating output file '%s': %w", outputFile, err)
		}
		return nil, nil, fmt.Errorf("failed to create output file '%s': %w", outputFile, err)
	}

	return file, func() {
		if err := file.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to close output file: %v\n", err)
		}
	}, nil
}

// DetectMimeType returns the MIME type of a file based on its content and extension
func DetectMimeType(filePath string, content []byte) string {
	// First try to detect from content
	mimeType := http.DetectContentType(content)

	// For some common file types, use extension-based detection as fallback
	if mimeType == "application/octet-stream" {
		ext := strings.ToLower(filepath.Ext(filePath))
		switch ext {
		case ".csv":
			return "text/csv"
		case ".json":
			return "application/json"
		case ".txt":
			return "text/plain"
		case ".pdf":
			return "application/pdf"
		case ".xlsx", ".xls":
			return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		case ".docx", ".doc":
			return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
		case ".pptx", ".ppt":
			return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
		}
		return mimeType
	}

	// Optimize text MIME type processing
	if strings.HasPrefix(mimeType, "text/") {
		if idx := strings.IndexByte(mimeType, ';'); idx != -1 {
			return mimeType[:idx]
		}
	}

	return mimeType
}
