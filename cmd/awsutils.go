/*
Copyright © 2025 AWS-BIA Contributors

This file contains AWS-specific utilities for the AWS Bedrock Intelligent Agents CLI.
It handles AWS configuration, client creation, and error handling.
*/
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime/types"
	"github.com/google/uuid"
)

// AWSHelper provides AWS-specific functionality
type AWSHelper struct {
	Options    AgentOptions
	FileHelper *FileHelper
}

// NewAWSHelper creates a new AWSHelper
func NewAWSHelper(opts AgentOptions) *AWSHelper {
	return &AWSHelper{
		Options:    opts,
		FileHelper: NewFileHelper(opts),
	}
}

// LoadConfig loads the AWS SDK configuration with the specified region
func (a *AWSHelper) LoadConfig(ctx context.Context) (aws.Config, error) {
	configOptions := []func(*config.LoadOptions) error{}

	if a.Options.Region != "" {
		configOptions = append(configOptions, config.WithRegion(a.Options.Region))
	}

	return config.LoadDefaultConfig(ctx, configOptions...)
}

// CreateClient creates a Bedrock Agent runtime client
func (a *AWSHelper) CreateClient(ctx context.Context) (*bedrockagentruntime.Client, error) {
	cfg, err := a.LoadConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return bedrockagentruntime.NewFromConfig(cfg), nil
}

// PrepareInvokeInput creates the InvokeAgentInput struct from the options
func (a *AWSHelper) PrepareInvokeInput() (*bedrockagentruntime.InvokeAgentInput, error) {
	input := &bedrockagentruntime.InvokeAgentInput{
		AgentId:      aws.String(a.Options.AgentID),
		AgentAliasId: aws.String(a.Options.AgentAliasID),
		InputText:    aws.String(a.Options.InputText),
	}

	// Add session ID if provided, otherwise generate a random UUID
	if a.Options.SessionID != "" {
		input.SessionId = aws.String(a.Options.SessionID)
	} else {
		// Generate a random UUID as session ID
		randomSessionID := uuid.New().String()
		input.SessionId = aws.String(randomSessionID)

		// Log the generated session ID if verbose mode is enabled
		logVerbose(a.Options, "Generated random session ID: %s", randomSessionID)
	}

	// Add file uploads if specified
	if len(a.Options.UploadFiles) > 0 {
		// Initialize the session state if nil
		if input.SessionState == nil {
			input.SessionState = &types.SessionState{}
		}

		// Process files
		inputFiles, err := a.FileHelper.PrepareInputFiles()
		if err != nil {
			return nil, err
		}

		// Add the files to the session state
		input.SessionState.Files = inputFiles
	}

	j, _ := json.Marshal(input)
	logVerbose(a.Options, "Prepared InvokeAgentInput: %s", string(j))
	return input, nil
}

// HandleAWSError provides more detailed error information based on the AWS error type
func HandleAWSError(err error) error {
	if err == nil {
		return nil
	}

	// Check for specific error types to provide better error messages
	if strings.Contains(err.Error(), "operation error Bedrock Agent Runtime") {
		if strings.Contains(err.Error(), "InvalidAgentAliasId") {
			return fmt.Errorf("agent alias ID not found or not accessible with your credentials: %w", err)
		}
		if strings.Contains(err.Error(), "InvalidAgentId") {
			return fmt.Errorf("agent ID not found or not accessible with your credentials: %w", err)
		}
		if strings.Contains(err.Error(), "ThrottlingException") {
			return fmt.Errorf("request was throttled - please reduce request rate and try again: %w", err)
		}
		if strings.Contains(err.Error(), "ValidationException") {
			return fmt.Errorf("validation error - please check your input parameters: %w", err)
		}
		if strings.Contains(err.Error(), "AccessDeniedException") {
			return fmt.Errorf("access denied - check your IAM permissions for Bedrock services: %w", err)
		}
		if strings.Contains(err.Error(), "ResourceNotFoundException") {
			return fmt.Errorf("resource not found - verify your agent ID and alias ID are correct: %w", err)
		}
		if strings.Contains(err.Error(), "context deadline exceeded") ||
			strings.Contains(err.Error(), "timeout") {
			return fmt.Errorf("operation timed out - try increasing the timeout value: %w", err)
		}
		if strings.Contains(err.Error(), "ValidationException") && strings.Contains(err.Error(), "file") {
			return fmt.Errorf("file validation error - check file sizes and formats: %w", err)
		}
	}

	// For network-related errors
	if strings.Contains(err.Error(), "dial tcp") && strings.Contains(err.Error(), "i/o timeout") {
		return fmt.Errorf("network connection timed out - check your internet connection: %w", err)
	}

	// For context cancellation (user interrupt)
	if strings.Contains(err.Error(), "context canceled") {
		return fmt.Errorf("operation canceled by user: %w", err)
	}

	// Return the original error with a general message if no specific case matches
	return fmt.Errorf("AWS Bedrock error: %w", err)
}
