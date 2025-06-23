/*
Copyright © 2025 AWS-BIA Contributors

This file implements centralized logging using go.uber.org/zap library.
*/
package cmd

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger *zap.Logger
	sugar  *zap.SugaredLogger
)

// InitLogger initializes the global logger based on verbose mode
func InitLogger(verbose bool) {
	config := zap.NewProductionConfig()

	if verbose {
		// In verbose mode, use development config for more readable output
		config = zap.NewDevelopmentConfig()
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
		config.Development = true
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	} else {
		// In non-verbose mode, only show errors and warnings
		config.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
		config.Development = false
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		config.DisableCaller = true
		config.DisableStacktrace = true
	}

	// Build the logger
	var err error
	logger, err = config.Build()
	if err != nil {
		// Fallback to basic logger if build fails
		logger = zap.NewNop()
	}

	sugar = logger.Sugar()
}

// GetLogger returns the global zap logger
func GetLogger() *zap.Logger {
	if logger == nil {
		InitLogger(false) // Default to non-verbose
	}
	return logger
}

// GetSugar returns the global zap sugared logger for easier printf-style logging
func GetSugar() *zap.SugaredLogger {
	if sugar == nil {
		InitLogger(false) // Default to non-verbose
	}
	return sugar
}

// LogVerbose logs a debug message using zap (replacement for the old logVerbose function)
func LogVerbose(opts AgentOptions, format string, args ...interface{}) {
	if sugar == nil {
		InitLogger(opts.Verbose)
	}

	if opts.Verbose {
		sugar.Debugf(format, args...)
	}
}

// LogError logs an error message using zap (replacement for the old logError function)
func LogError(message string, err error) {
	if sugar == nil {
		InitLogger(false)
	}

	sugar.Errorw(message, "error", err)
}

// LogInfo logs an info message
func LogInfo(format string, args ...interface{}) {
	if sugar == nil {
		InitLogger(false)
	}

	sugar.Infof(format, args...)
}

// LogWarn logs a warning message
func LogWarn(format string, args ...interface{}) {
	if sugar == nil {
		InitLogger(false)
	}

	sugar.Warnf(format, args...)
}

// Sync flushes the logger (should be called before program exit)
func SyncLogger() {
	if logger != nil {
		_ = logger.Sync()
	}
}
