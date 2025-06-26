/*
Copyright © 2025 AWS-BIA Contributors

This file implements prompt template handling functionality for the AWS Bedrock Intelligent Agents CLI.
*/
package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

// PromptManager handles loading and processing prompt templates
type PromptManager struct {
	promptDirs []string
	funcMap    template.FuncMap // Cache template functions
}

// NewPromptManager creates a new prompt manager with default search locations
func NewPromptManager() *PromptManager {
	// Pre-allocate with known capacity
	promptDirs := make([]string, 0, 3)

	// Add current directory
	promptDirs = append(promptDirs, "prompts")

	// Add user home directory
	if home, err := os.UserHomeDir(); err == nil {
		promptDirs = append(promptDirs, filepath.Join(home, ".aws-bia", "prompts"))
	}

	// Add global directory if available
	promptDirs = append(promptDirs, "/usr/local/share/aws-bia/prompts")

	// Pre-create function map to avoid recreation on each template processing
	funcMap := template.FuncMap{
		"toLowerCase": strings.ToLower,
		"toUpperCase": strings.ToUpper,
		"replace": func(old, new, s string) string {
			return strings.ReplaceAll(s, old, new)
		},
		"join":      strings.Join,
		"split":     strings.Split,
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"trim":      strings.TrimSpace,
	}

	return &PromptManager{
		promptDirs: promptDirs,
		funcMap:    funcMap,
	}
}

// GetAvailablePrompts returns a list of available prompts
func (pm *PromptManager) GetAvailablePrompts() []string {
	var prompts []string
	seen := make(map[string]struct{}) // Use struct{} instead of bool for memory efficiency

	// Pre-define supported extensions for faster lookup
	supportedExts := map[string]struct{}{
		".txt":    {},
		".md":     {},
		".prompt": {},
	}

	for _, dir := range pm.promptDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		files, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			// Check for supported extensions using map lookup (faster)
			name := file.Name()
			ext := filepath.Ext(name)
			if _, supported := supportedExts[ext]; supported {
				// Strip extension
				baseName := strings.TrimSuffix(name, ext)
				if _, exists := seen[baseName]; !exists {
					prompts = append(prompts, baseName)
					seen[baseName] = struct{}{}
				}
			}
		}
	}

	return prompts
}

// LoadPrompt loads a prompt by name or file path
func (pm *PromptManager) LoadPrompt(promptName, promptFile string) (string, error) {
	var promptContent string
	var err error

	if promptFile != "" {
		// Load from specified file
		promptContent, err = pm.loadPromptFromFile(promptFile)
		if err != nil {
			return "", fmt.Errorf("failed to load prompt file %s: %w", promptFile, err)
		}
	} else if promptName != "" {
		// Search for prompt in prompt directories
		promptContent, err = pm.findPromptByName(promptName)
		if err != nil {
			return "", fmt.Errorf("failed to find prompt '%s': %w", promptName, err)
		}
	} else {
		// No prompt specified
		return "", nil
	}

	return promptContent, nil
}

// ProcessPromptTemplate processes template variables in the prompt
func (pm *PromptManager) ProcessPromptTemplate(promptContent string, vars []string) (string, error) {
	// If no variables, return the original content
	if len(vars) == 0 {
		return promptContent, nil
	}

	// Convert vars slice to map with pre-allocated capacity
	varMap := make(map[string]interface{}, len(vars))
	for _, v := range vars {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid variable format '%s', expected 'key=value'", v)
		}

		key, value := parts[0], parts[1]

		// Allow boolean and numeric values for advanced templating
		switch strings.ToLower(value) {
		case "true":
			varMap[key] = true
		case "false":
			varMap[key] = false
		default:
			// Try to parse as number
			if num, err := strconv.ParseFloat(value, 64); err == nil {
				varMap[key] = num
			} else {
				// Default to string
				varMap[key] = value
			}
		}
	}

	// Parse and execute the template using cached function map
	tmpl, err := template.New("prompt").Funcs(pm.funcMap).Parse(promptContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse prompt template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, varMap); err != nil {
		return "", fmt.Errorf("failed to apply template variables: %w", err)
	}

	return buf.String(), nil
}

// loadPromptFromFile loads prompt content from a file path
func (pm *PromptManager) loadPromptFromFile(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// findPromptByName searches for a prompt by name in the prompt directories
func (pm *PromptManager) findPromptByName(name string) (string, error) {
	// Try with different extensions
	extensions := []string{"", ".txt", ".md", ".prompt"}

	// Pre-check if name already has an extension to avoid unnecessary suffix checks
	hasExtension := strings.Contains(name, ".")

	for _, ext := range extensions {
		var nameWithExt string
		if ext == "" || (hasExtension && strings.HasSuffix(name, ext)) {
			nameWithExt = name
		} else if !hasExtension {
			nameWithExt = name + ext
		} else {
			continue // Skip if name has extension but doesn't match current ext
		}

		// Search in prompt directories
		for _, dir := range pm.promptDirs {
			path := filepath.Join(dir, nameWithExt)
			if data, err := os.ReadFile(path); err == nil {
				// Found and read the prompt file in one step
				return string(data), nil
			}
		}
	}

	return "", fmt.Errorf("prompt '%s' not found in %v", name, pm.promptDirs)
}
