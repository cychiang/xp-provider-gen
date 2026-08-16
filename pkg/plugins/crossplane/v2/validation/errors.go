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

// PluginError reports a failed command step and, where we can infer one, how to
// fix it. The hints are the point: a scaffolding tool fails in a handful of
// predictable ways, and naming the fix beats making the user guess.
type PluginError struct {
	Component string // the command: "init", "createAPI"
	Operation string // the step that failed: "domain validation", "scaffolding"
	Cause     error
	Hints     []string
}

func (e PluginError) Error() string {
	msg := fmt.Sprintf("%s %s failed: %v", e.Component, e.Operation, e.Cause)
	if len(e.Hints) > 0 {
		msg += "\n\nSuggestions:\n  - " + strings.Join(e.Hints, "\n  - ")
	}
	return msg
}

// Unwrap returns the underlying cause for error chain compatibility.
func (e PluginError) Unwrap() error {
	return e.Cause
}

// hintRule maps a substring of the cause to the advice it warrants. Matching on
// the message keeps the hint table next to the wording it reacts to; the hints
// are advisory, so a miss costs nothing.
type hintRule struct {
	match string
	hints []string
}

var (
	initHints = []hintRule{
		{"domain", []string{"Ensure domain is a valid DNS name (e.g., example.com)"}},
		{"repository", []string{
			"Repository should be a valid go module name",
			"Example: github.com/example/provider-example",
		}},
		{"git", []string{
			"Ensure git is installed and configured",
			"Check if you have write permissions in the directory",
		}},
		{"submodule", []string{
			"You can manually add the build submodule later:",
			"git submodule add https://github.com/crossplane/build build",
		}},
	}

	createAPIHints = []hintRule{
		{"group", []string{"Group should be lowercase with hyphens (e.g., compute, storage)"}},
		{"version", []string{"Version should follow Kubernetes format (e.g., v1alpha1, v1beta1)"}},
		{"kind", []string{"Kind should be PascalCase (e.g., Instance, Bucket)"}},
		{"domain", []string{"Ensure the project is initialized with 'init' command first"}},
		{"template", []string{
			"Check if there are conflicting files in the target location",
			"Use --force flag to overwrite existing files",
		}},
	}
)

// InitError reports a failed `init` step.
func InitError(operation string, cause error) error {
	return newPluginError("init", operation, cause, initHints)
}

// CreateAPIError reports a failed `create api` step.
func CreateAPIError(operation string, cause error) error {
	return newPluginError("createAPI", operation, cause, createAPIHints)
}

// newPluginError builds the error, attaching the hints of the first rule whose
// substring appears in the cause.
func newPluginError(component, operation string, cause error, rules []hintRule) error {
	err := PluginError{Component: component, Operation: operation, Cause: cause}
	for _, rule := range rules {
		if strings.Contains(cause.Error(), rule.match) {
			err.Hints = rule.hints
			break
		}
	}
	return err
}
