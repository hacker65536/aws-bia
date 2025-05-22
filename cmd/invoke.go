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
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	invokeCmd.Flags().StringVar(&opts.InputText, "input", "", "The input text to send to the agent")

	// Only mark input as required - agent IDs can come from config
	invokeCmd.MarkFlagRequired("input")

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
	// Load configuration from file if specified
	if err := loadConfig(opts.ConfigFile); err != nil {
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
	if opts.InputText == "" {
		return fmt.Errorf("input text is required")
	}
	return nil
}

// validateOutputFormat validates the output format is supported
func validateOutputFormat(opts AgentOptions) error {
	validFormats := []string{OutputFormatText, OutputFormatJSON}
	for _, format := range validFormats {
		if opts.OutputFormat == format {
			return nil
		}
	}
	return fmt.Errorf("output format must be one of: %s, got '%s'",
		strings.Join(validFormats, ", "), opts.OutputFormat)
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
	if len(opts.UploadFiles) == 0 {
		return nil
	}

	// Check if files exist
	for _, filePath := range opts.UploadFiles {
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return fmt.Errorf("upload file '%s' does not exist", filePath)
		}
	}

	// Check file count limit
	if len(opts.UploadFiles) > MaxUploadFiles {
		return fmt.Errorf("maximum %d files can be uploaded, got %d", MaxUploadFiles, len(opts.UploadFiles))
	}

	// Validate file use case
	allowedUseCases := []string{FileUseCaseCodeInterpreter, FileUseCaseChat}
	for _, useCase := range allowedUseCases {
		if opts.FileUseCase == useCase {
			return nil
		}
	}

	return fmt.Errorf("file-use-case must be one of: %s, got '%s'",
		strings.Join(allowedUseCases, ", "), opts.FileUseCase)
}

// loadConfig loads agent configuration from a YAML file using Viper
func loadConfig(configPath string) error {
	v := viper.New()

	// Set default config name and locations
	v.SetConfigName("aws-bia")
	v.SetConfigType("yaml")

	// Add default search paths
	v.AddConfigPath(".") // Current directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		v.AddConfigPath(homeDir)                            // User's home directory
		v.AddConfigPath(filepath.Join(homeDir, ".aws-bia")) // .aws-bia in home directory
	}

	// If config file is explicitly specified, use it
	if configPath != "" {
		// Check if the file exists
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return fmt.Errorf("specified config file not found: %s", configPath)
		}

		v.SetConfigFile(configPath)
	}

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		// Only return error if a config file was explicitly specified
		if configPath != "" {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		// If no config was specified and we couldn't find one, that's okay
		return nil
	}

	// If a config file was found, read the agent and alias IDs
	if v.IsSet("agent_id") && opts.AgentID == "" {
		opts.AgentID = v.GetString("agent_id")
	}

	if v.IsSet("agent_alias_id") && opts.AgentAliasID == "" {
		opts.AgentAliasID = v.GetString("agent_alias_id")
	}

	// Also load region if set in config file and not provided via flag
	if v.IsSet("region") && opts.Region == "" {
		opts.Region = v.GetString("region")
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

	// If input text is already provided, consider the prompt as a prefix or template
	if opts.InputText != "" {
		// Append or replace depends on whether the prompt contains a placeholder
		if strings.Contains(processedPrompt, "{{input}}") {
			// Replace the placeholder with the input text
			processedPrompt = strings.ReplaceAll(processedPrompt, "{{input}}", opts.InputText)
		} else {
			// Otherwise, append the input to the prompt
			processedPrompt = processedPrompt + "\n" + opts.InputText
		}
	}

	// Set the processed prompt as the input text
	opts.InputText = processedPrompt

	return nil
}
