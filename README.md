# AWS-BIA (AWS Bedrock Invoke Agent CLI)

A powerful command-line interface tool for interacting with AWS Bedrock Invoke Agent. AWS-BIA provides a comprehensive set of features including streaming responses, file uploads, prompt templates, and flexible configuration options.

## Installation

### Using Go Install (Recommended)

```bash
go install github.com/hacker65536/aws-bia@latest
```

### Manual Build

```bash
git clone https://github.com/hacker65536/aws-bia.git
cd aws-bia
go build -o aws-bia
```

### Using Pre-built Releases

Download the latest binary from the [releases page](https://github.com/hacker65536/aws-bia/releases).

## Configuration

### AWS Configuration

AWS-BIA uses the standard AWS configuration from your environment. Make sure you have:

1. **AWS Credentials**: Properly configured in `~/.aws/credentials` or via environment variables
2. **IAM Permissions**: The necessary IAM permissions to use AWS Bedrock services
3. **AWS Region**: Set via environment variable `AWS_REGION` or specify with `--region` flag

### Configuration File Support

AWS-BIA supports YAML configuration files to store commonly used settings:

**Default locations (searched in order):**
- `./aws-bia.yaml` (current directory)
- `~/.aws-bia.yaml` (home directory)
- `~/.aws-bia/aws-bia.yaml` (config directory)

**Example configuration file:**
```yaml
# ~/.aws-bia.yaml
agent_id: "your-default-agent-id"
agent_alias_id: "your-default-alias-id"
region: "us-west-2"
timeout: "60s"  # Request timeout (supports formats like "30s", "1m", "2h30m")
```

**Using a specific config file:**
```bash
aws-bia invoke --config /path/to/config.yaml --input "Your question"
```

## Usage

### Invoke a Bedrock Agent

```bash
# Basic usage
aws-bia invoke --agent-id your-agent-id --agent-alias-id your-alias-id --input "Your question to the agent"

# With session ID for conversation continuity
aws-bia invoke --agent-id your-agent-id --agent-alias-id your-alias-id --session-id your-session-id --input "Follow-up question"

# With streaming enabled (shows real-time responses)
aws-bia invoke --agent-id your-agent-id --agent-alias-id your-alias-id --input "Your question" --stream

# Specify AWS region
aws-bia invoke --agent-id your-agent-id --agent-alias-id your-alias-id --input "Your question" --region us-west-2

# Using JSON output format
aws-bia invoke --agent-id your-agent-id --agent-alias-id your-alias-id --input "Your question" --format json

# Save response to a file
aws-bia invoke --agent-id your-agent-id --agent-alias-id your-alias-id --input "Your question" --output-file response.txt

# With verbose logging
aws-bia invoke --agent-id your-agent-id --agent-alias-id your-alias-id --input "Your question" --verbose

# Upload files to the agent (for code interpreter)
aws-bia invoke --agent-id your-agent-id --agent-alias-id your-alias-id --input "Analyze this data" --upload-files data.csv,schema.json

# Upload files with specific use case
aws-bia invoke --agent-id your-agent-id --agent-alias-id your-alias-id --input "Analyze this data" --upload-files data.csv --file-use-case CODE_INTERPRETER

# Combining options
aws-bia invoke --agent-id your-agent-id --agent-alias-id your-alias-id --input "Your question" --stream --format json --output-file response.json
```

## Prompt Templates

AWS-BIA includes built-in prompt templates for common use cases, making it easy to apply structured prompts without writing them from scratch.

### Available Templates

- **aws-expert**: Expert-level AWS consulting and guidance
- **code-review**: Code review and analysis prompts
- **terraform**: Infrastructure as Code and Terraform-specific prompts
- **translation**: Language translation prompts

### Using Prompt Templates

```bash
# Use a predefined prompt template (input is optional when using prompts)
aws-bia invoke --agent-id abc123 --agent-alias-id def456 --prompt code-review

# Use a prompt template with variables
aws-bia invoke --agent-id abc123 --agent-alias-id def456 --prompt translation --var language=Japanese --var text="Hello world"

# Combine a prompt template with additional input
aws-bia invoke --agent-id abc123 --agent-alias-id def456 --prompt aws-expert --input "How do I optimize my Lambda functions?"

# Use a custom prompt file (no need for --input)
aws-bia invoke --agent-id abc123 --agent-alias-id def456 --prompt-file ./my-custom-prompt.txt --var project=MyApp
```

### Template Variables

Prompt templates support variable substitution using the `--var` flag:

```bash
# Single variable
aws-bia invoke --prompt translation --var language=Spanish --input "Translate this text"

# Multiple variables
aws-bia invoke --prompt terraform --var environment=production --var region=us-east-1 --input "Review this configuration"
```

### Custom Prompt Files

You can also use your own prompt files:

```bash
# Use a custom prompt file
aws-bia invoke --agent-id abc123 --agent-alias-id def456 --prompt-file ./prompts/custom-prompt.txt --input "Your question"
```

**Template placeholders:**
- `{{input}}`: Replaced with the `--input` text (optional, can be omitted)
- `{{variable_name}}`: Replaced with values from `--var variable_name=value`
- `{{#if variable}}...{{/if}}`: Conditional content based on variable existence
- Template functions: `toLowerCase`, `toUpperCase`, `replace`, etc.

## Advanced Features

### Output Management

```bash
# Save response to a specific file
aws-bia invoke --agent-id abc123 --agent-alias-id def456 --input "Generate a report" --output-file ./reports/analysis.txt

# Save generated files to a directory
aws-bia invoke --agent-id abc123 --agent-alias-id def456 --input "Create charts" --save-files ./output

# JSON format for programmatic use
aws-bia invoke --agent-id abc123 --agent-alias-id def456 --input "Your question" --format json --output-file response.json
```

### Timeout and Debugging

```bash
# Set custom timeout (default: 30s)
aws-bia invoke --agent-id abc123 --agent-alias-id def456 --input "Complex analysis" --timeout 60s

# Enable verbose logging for debugging
aws-bia invoke --agent-id abc123 --agent-alias-id def456 --input "Your question" --verbose
```

### Configuration-based Usage

```bash
# Using configuration file (agent IDs from config)
aws-bia invoke --config ~/.aws-bia.yaml --input "Your question"

# Override config settings with command line flags
aws-bia invoke --config ~/.aws-bia.yaml --region us-east-1 --input "Your question"
```

## Streaming Mode

When using the `--stream` flag, the CLI will display agent responses in real-time as they are received from the AWS Bedrock service. This provides a more interactive experience, especially for longer responses or when the agent generates files.

### Features of Streaming Mode

- **Real-time Text Output**: See the agent's response as it's being generated
- **File Generation**: Handles files generated by the agent (e.g., from Code Interpreter actions)
- **Citations**: Properly displays citation information when the agent references sources
- **Return Control**: Shows when an agent returns control for custom action flows

### JSON Output with Streaming

When combining `--stream` with `--format json`, the CLI will collect all streaming events and provide a comprehensive JSON response at the end that includes:

- Complete text response
- Session information
- Any generated files (with metadata)
- Citations and references
- Return control information

This is useful for programmatic integration with other tools and scripts.

## File Upload Support

AWS-BIA supports uploading files to your Bedrock Agent. This is particularly useful when working with agents that use the Code Interpreter capability.

### File Upload Features

- Upload up to 5 files at once (total size limit: 10MB)
- Automatic MIME type detection for proper handling by the agent
- Support for various file types including CSV, JSON, PDF, images, etc.
- Customizable file use case (e.g., CODE_INTERPRETER)

### Using File Upload

```bash
# Upload a single file
aws-bia invoke --agent-id abc123 --agent-alias-id def456 --input "Analyze this data" --upload-files data.csv

# Upload multiple files
aws-bia invoke --agent-id abc123 --agent-alias-id def456 --input "Compare these datasets" --upload-files data1.csv,data2.csv

# Specify file use case
aws-bia invoke --agent-id abc123 --agent-alias-id def456 --input "Analyze this data" --upload-files data.csv --file-use-case CODE_INTERPRETER
```

## Examples

Example 1: Simple agent interaction
```bash
aws-bia invoke --agent-id abc123 --agent-alias-id def456 --input "What's the weather like in Seattle?"
```

Example 2: Multi-turn conversation
```bash
# First interaction - note the session ID that's generated
aws-bia invoke --agent-id abc123 --agent-alias-id def456 --input "Find me flights to Tokyo"

# Follow-up using the same session
aws-bia invoke --agent-id abc123 --agent-alias-id def456 --session-id session123 --input "What about flights to Osaka instead?"
```

Example 3: Working with a data file
```bash
# Upload a CSV file for analysis
aws-bia invoke --agent-id abc123 --agent-alias-id def456 --input "Analyze this sales data and create a report" --upload-files sales_data.csv

# In a follow-up, you can reference the uploaded file
aws-bia invoke --agent-id abc123 --agent-alias-id def456 --session-id session123 --input "Now create a chart showing the top 5 products"
```

## License

See the [LICENSE](LICENSE) file for details.
