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
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".aws-bia" (without extension).
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".aws-bia")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func SetVersionInfo(version, commit, date string) {
	// バージョン表示からvプレフィックスを削除
	displayVersion := version
	if len(version) > 0 && version[0] == 'v' {
		displayVersion = version[1:]
	}
	rootCmd.Version = fmt.Sprintf("%s (Built on %s from Git SHA %s)", displayVersion, date, commit)
}
