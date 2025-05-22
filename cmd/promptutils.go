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
	"strings"
	"text/template"
)

// PromptManager handles loading and processing prompt templates
type PromptManager struct {
	promptDirs []string
}

// NewPromptManager creates a new prompt manager with default search locations
func NewPromptManager() *PromptManager {
	pm := &PromptManager{
		promptDirs: []string{},
	}

	// Add current directory
	pm.promptDirs = append(pm.promptDirs, "prompts")

	// Add user home directory
	if home, err := os.UserHomeDir(); err == nil {
		pm.promptDirs = append(pm.promptDirs, filepath.Join(home, ".aws-bia", "prompts"))
	}

	// Add global directory if available
	pm.promptDirs = append(pm.promptDirs, "/usr/local/share/aws-bia/prompts")

	return pm
}

// GetAvailablePrompts returns a list of available prompts
func (pm *PromptManager) GetAvailablePrompts() []string {
	var prompts []string
	seen := make(map[string]bool)

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

			// Check for supported extensions
			name := file.Name()
			if strings.HasSuffix(name, ".txt") || strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".prompt") {
				// Strip extension
				baseName := strings.TrimSuffix(name, filepath.Ext(name))
				if !seen[baseName] {
					prompts = append(prompts, baseName)
					seen[baseName] = true
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

	// Convert vars slice to map
	varMap := make(map[string]string)
	for _, v := range vars {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid variable format '%s', expected 'key=value'", v)
		}
		varMap[parts[0]] = parts[1]
	}

	// Parse and execute the template
	tmpl, err := template.New("prompt").Parse(promptContent)
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

	for _, ext := range extensions {
		nameWithExt := name
		if ext != "" && !strings.HasSuffix(name, ext) {
			nameWithExt = name + ext
		}

		// Search in prompt directories
		for _, dir := range pm.promptDirs {
			path := filepath.Join(dir, nameWithExt)
			if _, err := os.Stat(path); err == nil {
				// Found the prompt file
				data, err := os.ReadFile(path)
				if err != nil {
					return "", err
				}
				return string(data), nil
			}
		}
	}

	return "", fmt.Errorf("prompt '%s' not found in %v", name, pm.promptDirs)
}
