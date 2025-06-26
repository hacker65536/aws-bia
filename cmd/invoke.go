/*
Copyright © 2025 AWS-BIA Contributors

This file implements the 'invoke' command for AWS Bedrock Intelligent Agents CLI.
It handles both streaming and non-streaming modes for agent invocation and integrates
the various helper modules for a more modular and maintainable codebase.
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// Constants for configuration
const (
	// Default timeout for agent invocation
	DefaultTimeout = 30 * time.Second

	// Output format options
	OutputFormatText = "text"
	OutputFormatJSON = "json"

	// File use case options
	FileUseCaseCodeInterpreter = "CODE_INTERPRETER"
	FileUseCaseChat            = "CHAT"

	// Maximum number of files that can be uploaded
	MaxUploadFiles = 5
)

// AgentOptions contains all options for invoking an agent
type AgentOptions struct {
	// Required options
	AgentID      string
	AgentAliasID string
	InputText    string

	// Optional options
	ConfigFile      string // New field for config file path
	SessionID       string
	Region          string
	EnableStreaming bool
	Timeout         time.Duration
	OutputFormat    string
	OutputFile      string
	FilesOutputDir  string
	Verbose         bool

	// File upload options
	UploadFiles []string
	FileUseCase string

	// Prompt options
	PromptFile string   // Path to a specific prompt file
	PromptName string   // Name of a prompt template from the prompt directory
	PromptVars []string // Variables to substitute in the prompt template (format: key=value)
}

var opts AgentOptions

// invokeCmd represents the invoke command
var invokeCmd = &cobra.Command{
	Use:   "invoke",
	Short: "Invoke AWS Bedrock agent",
	Long: `Invoke AWS Bedrock agent with the specified input.
    
This command allows you to interact with AWS Bedrock agents by providing 
text input and receiving the agent's response.

Examples:
  # Basic usage (session ID will be auto-generated)
  aws-bia invoke --agent-id abc123 --agent-alias-id def456 --input "What's the weather like in Seattle?"

  # Using a configuration file (with agent IDs defined in the file)
  aws-bia invoke --config ~/.aws-bia.yaml --input "What's the weather like in Seattle?"

  # With explicit session ID for multi-turn conversations
  aws-bia invoke --agent-id abc123 --agent-alias-id def456 --session-id session123 --input "Follow-up question"

  # With streaming enabled
  aws-bia invoke --agent-id abc123 --agent-alias-id def456 --input "Your question" --stream

  # Save output to a file
  aws-bia invoke --agent-id abc123 --agent-alias-id def456 --input "Generate a report" --output-file report.txt

  # Save generated files to a directory
  aws-bia invoke --agent-id abc123 --agent-alias-id def456 --input "Generate files" --save-files ./files

  # Upload files to the agent for analysis
  aws-bia invoke --agent-id abc123 --agent-alias-id def456 --input "Analyze this data" --upload-files data.csv,config.json
  
  # Use a predefined prompt template
  aws-bia invoke --agent-id abc123 --agent-alias-id def456 --prompt code-review
  
  # Use a prompt template with variables
  aws-bia invoke --agent-id abc123 --agent-alias-id def456 --prompt translation --var language=Japanese --var text="Hello world"
  
  # Combine a prompt with additional input
  aws-bia invoke --agent-id abc123 --agent-alias-id def456 --prompt system-prompt --input "Generate a Python script"
`,
	Run: func(cmd *cobra.Command, args []string) {
		// Handle interrupts gracefully
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()

		if err := runInvokeCommand(ctx, opts); err != nil {
			logError("Error invoking agent", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(invokeCmd)

	// Config file flag
	invokeCmd.Flags().StringVar(&opts.ConfigFile, "config", "", "Path to configuration file (yaml)")

	// Required flags
	invokeCmd.Flags().StringVar(&opts.AgentID, "agent-id", "", "The ID of the agent to invoke (can be set in config file)")
	invokeCmd.Flags().StringVar(&opts.AgentAliasID, "agent-alias-id", "", "The ID of the agent alias to invoke (can be set in config file)")
	invokeCmd.Flags().StringVar(&opts.InputText, "input", "", "The input text to send to the agent (can be omitted when using --prompt or --prompt-file)")

	// We'll validate input requirements in the validateOptions function
	// This allows us to make input optional when a prompt is provided

	// Optional flags
	invokeCmd.Flags().StringVar(&opts.SessionID, "session-id", "", "The session ID for the conversation (if not provided, a random ID will be generated)")
	invokeCmd.Flags().StringVar(&opts.Region, "region", "", "AWS region to use (defaults to AWS_REGION environment variable)")
	invokeCmd.Flags().BoolVar(&opts.EnableStreaming, "stream", false, "Enable streaming mode for the response")
	invokeCmd.Flags().DurationVar(&opts.Timeout, "timeout", DefaultTimeout, "Timeout for the request (default: 30s)")
	invokeCmd.Flags().StringVar(&opts.OutputFormat, "format", OutputFormatText, "Output format: text or json (default: text)")
	invokeCmd.Flags().StringVar(&opts.OutputFile, "output-file", "", "Save the response to a file")
	invokeCmd.Flags().StringVar(&opts.FilesOutputDir, "save-files", "", "Directory to save any files generated by the agent")
	invokeCmd.Flags().BoolVar(&opts.Verbose, "verbose", false, "Enable verbose output")
	invokeCmd.Flags().StringSliceVar(&opts.UploadFiles, "upload-files", []string{}, "File paths to upload to the agent (comma-separated)")
	invokeCmd.Flags().StringVar(&opts.FileUseCase, "file-use-case", FileUseCaseCodeInterpreter, "File use case: CODE_INTERPRETER or other supported values")

	// Prompt flags
	invokeCmd.Flags().StringVar(&opts.PromptFile, "prompt-file", "", "Path to a prompt file to use")
	invokeCmd.Flags().StringVar(&opts.PromptName, "prompt", "", "Name of a predefined prompt to use")
	invokeCmd.Flags().StringSliceVar(&opts.PromptVars, "var", []string{}, "Variables for prompt template (format: key=value)")
}

// runInvokeCommand handles the agent invocation based on the provided options
func runInvokeCommand(ctx context.Context, opts AgentOptions) error {
	// Initialize logger based on verbose flag
	InitLogger(opts.Verbose)
	defer SyncLogger()

	// Load configuration from file if specified
	if err := loadConfig(opts.ConfigFile, &opts); err != nil {
		return err
	}

	// Process prompt if specified
	if opts.PromptName != "" || opts.PromptFile != "" {
		if err := processPrompt(&opts); err != nil {
			return err
		}
	}

	// Validate inputs before proceeding
	if err := validateOptions(opts); err != nil {
		return err
	}

	// Setup context with timeout
	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	// Setup AWS helper and client
	awsHelper := NewAWSHelper(opts)
	client, err := awsHelper.CreateClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	// Prepare output writer
	writer, closer, err := PrepareOutput(opts.OutputFile)
	if err != nil {
		return fmt.Errorf("failed to prepare output: %w", err)
	}
	if closer != nil {
		defer closer()
	}

	// Create response formatter
	formatter := NewResponseFormatter(opts, writer)

	// Prepare the input for agent invocation
	input, err := awsHelper.PrepareInvokeInput()
	if err != nil {
		return fmt.Errorf("failed to prepare invoke input: %w", err)
	}

	logVerbose(opts, "Invoking agent with options: %+v", opts)

	// Invoke the agent and process response
	output, err := client.InvokeAgent(ctx, input)
	if err != nil {
		return HandleAWSError(fmt.Errorf("failed to invoke agent: %w", err))
	}

	// Format and write the response using the formatter
	return formatter.FormatAndWriteResponse(output)
}

// validateOptions validates the agent options before making API calls
func validateOptions(opts AgentOptions) error {
	// Validate required fields
	if err := validateRequiredFields(opts); err != nil {
		return err
	}

	// Validate timeout
	if opts.Timeout <= 0 {
		return fmt.Errorf("timeout must be a positive duration")
	}

	// Validate output format
	if err := validateOutputFormat(opts); err != nil {
		return err
	}

	// Validate file output directory if specified
	if err := validateFilesOutputDir(opts); err != nil {
		return err
	}

	// Validate file upload options if specified
	if err := validateFileUploadOptions(opts); err != nil {
		return err
	}

	return nil
}

// validateRequiredFields checks that all required fields have values
func validateRequiredFields(opts AgentOptions) error {
	if opts.AgentID == "" {
		return fmt.Errorf("agent ID is required")
	}
	if opts.AgentAliasID == "" {
		return fmt.Errorf("agent alias ID is required")
	}

	// Input is only required if no prompt or prompt file is specified
	if opts.InputText == "" && opts.PromptName == "" && opts.PromptFile == "" {
		return fmt.Errorf("input is required (or use --prompt/--prompt-file)")
	}
	return nil
}

// validateOutputFormat validates the output format is supported
func validateOutputFormat(opts AgentOptions) error {
	// Direct comparison instead of loop for better performance
	switch opts.OutputFormat {
	case OutputFormatText, OutputFormatJSON:
		return nil
	default:
		return fmt.Errorf("output format must be one of: %s, %s, got '%s'",
			OutputFormatText, OutputFormatJSON, opts.OutputFormat)
	}
}

// validateFilesOutputDir validates the directory for saving files
func validateFilesOutputDir(opts AgentOptions) error {
	if opts.FilesOutputDir == "" {
		return nil
	}

	stat, err := os.Stat(opts.FilesOutputDir)
	if err == nil && !stat.IsDir() {
		return fmt.Errorf("save-files path '%s' exists but is not a directory", opts.FilesOutputDir)
	}
	return nil
}

// validateFileUploadOptions validates file upload-related options
func validateFileUploadOptions(opts AgentOptions) error {
	uploadCount := len(opts.UploadFiles)
	if uploadCount == 0 {
		return nil
	}

	// Check file count limit early
	if uploadCount > MaxUploadFiles {
		return fmt.Errorf("maximum %d files can be uploaded, got %d", MaxUploadFiles, uploadCount)
	}

	// Check if files exist
	for _, filePath := range opts.UploadFiles {
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return fmt.Errorf("upload file '%s' does not exist", filePath)
		}
	}

	// Validate file use case with direct comparison instead of loop
	switch opts.FileUseCase {
	case FileUseCaseCodeInterpreter, FileUseCaseChat:
		return nil
	default:
		return fmt.Errorf("file-use-case must be one of: %s, %s, got '%s'",
			FileUseCaseCodeInterpreter, FileUseCaseChat, opts.FileUseCase)
	}
}

// loadConfig loads agent configuration from a YAML file using Viper
func loadConfig(configPath string, options *AgentOptions) error {
	// Use the centralized config loading function from root.go
	v, err := LoadConfigForCommand(configPath, options.Verbose)
	if err != nil {
		return err
	}

	// Check if we found any configuration values
	settingsFound := false

	// Load agent ID if set in config and not provided via flag
	if v.InConfig("agent_id") && options.AgentID == "" {
		settingsFound = true
		options.AgentID = v.GetString("agent_id")
		logVerbose(*options, "Loaded agent ID from config: %s", options.AgentID)
	}

	// Load agent alias ID if set in config and not provided via flag
	if v.InConfig("agent_alias_id") && options.AgentAliasID == "" {
		settingsFound = true
		options.AgentAliasID = v.GetString("agent_alias_id")
		logVerbose(*options, "Loaded agent alias ID from config: %s", options.AgentAliasID)
	}

	// Load region if set in config and not provided via flag
	if v.InConfig("region") && options.Region == "" {
		settingsFound = true
		options.Region = v.GetString("region")
		logVerbose(*options, "Loaded region from config: %s", options.Region)
	}

	// Load timeout if set in config and not provided via flag (check if it's still the default value)
	if v.InConfig("timeout") && options.Timeout == DefaultTimeout {
		settingsFound = true
		timeoutDuration := v.GetDuration("timeout")
		if timeoutDuration > 0 {
			options.Timeout = timeoutDuration
			logVerbose(*options, "Loaded timeout from config: %s", options.Timeout)
		} else {
			logVerbose(*options, "Invalid timeout value in config, using default: %s", DefaultTimeout)
		}
	}

	// If config file was found but had no relevant settings, show a warning
	if !settingsFound && v.ConfigFileUsed() != "" && options.Verbose {
		LogWarn("Config file found but no agent_id, agent_alias_id, region, or timeout settings found")
	}

	return nil
}

// processPrompt loads and processes a prompt template if specified
func processPrompt(opts *AgentOptions) error {
	// Don't do anything if neither prompt name nor file is provided
	if opts.PromptName == "" && opts.PromptFile == "" {
		return nil
	}

	// Initialize the prompt manager
	pm := NewPromptManager()

	// Load the prompt content
	promptContent, err := pm.LoadPrompt(opts.PromptName, opts.PromptFile)
	if err != nil {
		return err
	}

	// Apply template variables if any
	processedPrompt, err := pm.ProcessPromptTemplate(promptContent, opts.PromptVars)
	if err != nil {
		return err
	}

	// Check if we got any content
	if processedPrompt == "" {
		return fmt.Errorf("loaded prompt is empty")
	}

	// Cache input placeholder for efficiency
	const inputPlaceholder = "{{input}}"
	hasInputPlaceholder := strings.Contains(processedPrompt, inputPlaceholder)

	// If input text is already provided, consider the prompt as a prefix or template
	if opts.InputText != "" {
		// Append or replace depends on whether the prompt contains a placeholder
		if hasInputPlaceholder {
			// Replace the placeholder with the input text
			processedPrompt = strings.ReplaceAll(processedPrompt, inputPlaceholder, opts.InputText)
		} else {
			// Otherwise, append the input to the prompt (use strings.Builder for efficiency)
			var builder strings.Builder
			builder.Grow(len(processedPrompt) + 1 + len(opts.InputText)) // Pre-allocate
			builder.WriteString(processedPrompt)
			builder.WriteByte('\n')
			builder.WriteString(opts.InputText)
			processedPrompt = builder.String()
		}
	} else if hasInputPlaceholder {
		// If no input is provided, use the prompt as-is, but replace any {{input}} with empty string
		processedPrompt = strings.ReplaceAll(processedPrompt, inputPlaceholder, "")
	}

	// Set the processed prompt as the input text
	opts.InputText = processedPrompt

	return nil
}
