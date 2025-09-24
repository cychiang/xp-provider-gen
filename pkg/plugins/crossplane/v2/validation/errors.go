/*
Copyright 2025 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package validation

import (
	"fmt"
	"strings"
)

// PluginError represents structured error information for better user experience
// while maintaining compatibility with kubebuilder's error handling patterns
type PluginError struct {
	Component string // e.g., "init", "createAPI", "template"
	Operation string // e.g., "validation", "scaffolding", "configuration"
	Cause     error
	Hints     []string // User-friendly suggestions
}

// Error implements the error interface
func (e PluginError) Error() string {
	msg := fmt.Sprintf("%s %s failed: %v", e.Component, e.Operation, e.Cause)

	if len(e.Hints) > 0 {
		msg += "\n\nSuggestions:"
		for _, hint := range e.Hints {
			msg += fmt.Sprintf("\n  - %s", hint)
		}
	}

	return msg
}

// Unwrap returns the underlying cause for error chain compatibility
func (e PluginError) Unwrap() error {
	return e.Cause
}

// ErrorBuilder helps construct PluginError with fluent API
type ErrorBuilder struct {
	component string
	operation string
	cause     error
	hints     []string
}

// NewError creates a new error builder
func NewError(component string) *ErrorBuilder {
	return &ErrorBuilder{component: component}
}

// Operation sets the operation that failed
func (b *ErrorBuilder) Operation(op string) *ErrorBuilder {
	b.operation = op
	return b
}

// Cause sets the underlying error
func (b *ErrorBuilder) Cause(err error) *ErrorBuilder {
	b.cause = err
	return b
}

// Hint adds a user-friendly suggestion
func (b *ErrorBuilder) Hint(hint string) *ErrorBuilder {
	b.hints = append(b.hints, hint)
	return b
}

// Build creates the final PluginError
func (b *ErrorBuilder) Build() error {
	return PluginError{
		Component: b.component,
		Operation: b.operation,
		Cause:     b.cause,
		Hints:     b.hints,
	}
}

// Common error constructors for frequently used patterns

// ValidationError creates a validation error with helpful hints
func ValidationError(field, value, message string) error {
	return NewError("validation").
		Operation("field validation").
		Cause(fmt.Errorf("invalid %s '%s': %s", field, value, message)).
		Hint("Check the documentation for valid formats").
		Hint("Use --help flag to see examples").
		Build()
}

// InitError creates an init command error with context
func InitError(operation string, cause error) error {
	builder := NewError("init").
		Operation(operation).
		Cause(cause)

	// Add context-specific hints
	switch {
	case strings.Contains(cause.Error(), "domain"):
		builder = builder.Hint("Ensure domain is a valid DNS name (e.g., example.com)")
	case strings.Contains(cause.Error(), "repository"):
		builder = builder.Hint("Repository should be a valid go module name")
		builder = builder.Hint("Example: github.com/example/provider-example")
	case strings.Contains(cause.Error(), "git"):
		builder = builder.Hint("Ensure git is installed and configured")
		builder = builder.Hint("Check if you have write permissions in the directory")
	case strings.Contains(cause.Error(), "submodule"):
		builder = builder.Hint("You can manually add the build submodule later:")
		builder = builder.Hint("git submodule add https://github.com/crossplane/build build")
	}

	return builder.Build()
}

// CreateAPIError creates a create api command error with context
func CreateAPIError(operation string, cause error) error {
	builder := NewError("createAPI").
		Operation(operation).
		Cause(cause)

	// Add context-specific hints
	switch {
	case strings.Contains(cause.Error(), "group"):
		builder = builder.Hint("Group should be lowercase with hyphens (e.g., compute, storage)")
	case strings.Contains(cause.Error(), "version"):
		builder = builder.Hint("Version should follow Kubernetes format (e.g., v1alpha1, v1beta1)")
	case strings.Contains(cause.Error(), "kind"):
		builder = builder.Hint("Kind should be PascalCase (e.g., Instance, Bucket)")
	case strings.Contains(cause.Error(), "domain"):
		builder = builder.Hint("Ensure the project is initialized with 'init' command first")
	case strings.Contains(cause.Error(), "template"):
		builder = builder.Hint("Check if there are conflicting files in the target location")
		builder = builder.Hint("Use --force flag to overwrite existing files")
	}

	return builder.Build()
}

// TemplateError creates a template processing error with context
func TemplateError(templateName string, cause error) error {
	return NewError("template").
		Operation(fmt.Sprintf("processing %s", templateName)).
		Cause(cause).
		Hint("Check if all required template variables are provided").
		Hint("Verify the template syntax is correct").
		Build()
}

// ScaffoldError creates a scaffolding error with context
func ScaffoldError(operation string, cause error) error {
	return NewError("scaffold").
		Operation(operation).
		Cause(cause).
		Hint("Check if you have write permissions in the target directory").
		Hint("Ensure all required dependencies are available").
		Build()
}

// WrapWithContext wraps an error with additional context while preserving error chains
func WrapWithContext(err error, component, operation string) error {
	if err == nil {
		return nil
	}

	// If it's already a PluginError, don't double-wrap
	var pluginErr PluginError
	if As(err, &pluginErr) {
		return err
	}

	return NewError(component).
		Operation(operation).
		Cause(err).
		Build()
}

// As is a compatibility function for error unwrapping
func As(err error, target interface{}) bool {
	switch t := target.(type) {
	case *PluginError:
		if pe, ok := err.(PluginError); ok {
			*t = pe
			return true
		}
	}
	return false
}
