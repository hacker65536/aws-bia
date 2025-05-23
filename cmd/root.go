/*
Copyright © 2025 AWS-BIA Contributors

This file implements the root command for the AWS Bedrock Intelligent Agents CLI.
It sets up the base command structure, configuration loading, and global flags
that are shared across all subcommands.

The root command serves as the entry point for the CLI application and provides
the foundation for the command hierarchy.
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "aws-bia",
	Short: "CLI tool for interacting with AWS Bedrock Intelligent Agents",
	Long: `AWS-BIA is a command-line interface for interacting with AWS Bedrock Intelligent Agents.

It provides a convenient way to invoke Bedrock agents, manage sessions,
and test agent functionality from the command line.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.aws-bia.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
// This is called automatically by cobra on startup.
// We just do minimal setup here and let each command load their specific config.
func initConfig() {
	// Setup environment variables
	viper.AutomaticEnv() // read in environment variables that match

	// For global verbose mode, we don't need to load the config here.
	// Each command will load its own config as needed.
}

// LoadConfigForCommand loads configuration values from a file for any command
// and applies them to the provided options structure.
// This function should be used by all commands that need configuration values.
func LoadConfigForCommand(configPath string, verbose bool) (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigType("yaml")

	// If config file is explicitly specified, use it directly
	if configPath != "" {
		absPath, err := filepath.Abs(configPath)
		if err != nil {
			absPath = configPath
		}

		// Check if the file exists
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("specified config file not found: %s", absPath)
		}
		v.SetConfigFile(absPath)

		if verbose {
			fmt.Fprintf(os.Stderr, "Using specified config file: %s\n", absPath)
		}

		// Read the explicitly specified config file
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		// Return early for explicitly specified config files
		return v, nil
	}

	// Variables to track config discovery
	var configFound bool

	// Otherwise search in standard locations for both naming conventions
	searchPaths := []string{"."} // Current directory first

	// Add home directory paths if available
	homeDir, err := os.UserHomeDir()
	if err == nil {
		searchPaths = append(
			searchPaths,
			homeDir,                            // User's home directory
			filepath.Join(homeDir, ".aws-bia"), // .aws-bia in home directory
		)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Searching for config in: [current directory, home directory, ~/.aws-bia]\n")
	}

	// Try each naming convention, giving preference to non-dotfile
	configFound = false

	// First try aws-bia.yaml (without dot prefix)
	v1 := setupViperInstance("aws-bia", searchPaths)
	err = v1.ReadInConfig()
	if err == nil {
		configFound = true
		v = v1
		// We'll output the final config file path at the end, not here
	} else {
		// Then try .aws-bia.yaml (with dot prefix)
		v2 := setupViperInstance(".aws-bia", searchPaths)
		err = v2.ReadInConfig()
		if err == nil {
			configFound = true
			v = v2
			// We'll output the final config file path at the end, not here
		}
	}

	if !configFound {
		if verbose {
			fmt.Fprintln(os.Stderr, "No configuration file found, using command line options only")
		}
	}

	// If a config file was found, show where it was loaded from
	if configFound && verbose {
		fmt.Fprintf(os.Stderr, "Using config file: %s\n", v.ConfigFileUsed())
	}

	return v, nil
}

// setupViperInstance creates a new viper instance with the given name and search paths
func setupViperInstance(configName string, searchPaths []string) *viper.Viper {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigName(configName)

	// Add all paths to search
	for _, path := range searchPaths {
		v.AddConfigPath(path)
	}

	return v
}

func SetVersionInfo(version, commit, date string) {
	// バージョン表示からvプレフィックスを削除
	displayVersion := version
	if len(version) > 0 && version[0] == 'v' {
		displayVersion = version[1:]
	}
	rootCmd.Version = fmt.Sprintf("%s (Built on %s from Git SHA %s)", displayVersion, date, commit)
}
