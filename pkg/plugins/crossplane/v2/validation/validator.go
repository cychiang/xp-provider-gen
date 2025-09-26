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
	"regexp"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

// FieldValidationError represents a user input field validation error.
type FieldValidationError struct {
	Field   string
	Value   string
	Message string
}

func (e FieldValidationError) Error() string {
	return fmt.Sprintf("invalid %s '%s': %s", e.Field, e.Value, e.Message)
}

// Validator provides validation utilities that follow kubebuilder patterns.
type Validator struct{}

// NewValidator creates a new validator instance.
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateDomain validates the domain follows kubebuilder conventions.
func (v *Validator) ValidateDomain(domain string) error {
	if domain == "" {
		return FieldValidationError{
			Field:   "domain",
			Value:   domain,
			Message: "domain is required",
		}
	}

	// kubebuilder domain validation pattern
	domainPattern := `^[a-z0-9]+([-.][a-z0-9]+)*\.[a-z]{2,}$`
	matched, err := regexp.MatchString(domainPattern, domain)
	if err != nil {
		return fmt.Errorf("error validating domain: %w", err)
	}

	if !matched {
		return FieldValidationError{
			Field:   "domain",
			Value:   domain,
			Message: "must be a valid domain name (e.g., example.com)",
		}
	}

	// Prevent common mistakes
	if strings.HasSuffix(domain, ".local") {
		return FieldValidationError{
			Field:   "domain",
			Value:   domain,
			Message: ".local domains are not recommended for production use",
		}
	}

	return nil
}

// ValidateRepository validates the repository follows go module conventions.
func (v *Validator) ValidateRepository(repo string) error {
	if repo == "" {
		return FieldValidationError{
			Field:   "repository",
			Value:   repo,
			Message: "repository is required",
		}
	}

	// Go module validation pattern - follows kubebuilder's requirements
	repoPattern := `^[a-z0-9.-]+/[a-z0-9._-]+/[a-z0-9._-]+$`
	matched, err := regexp.MatchString(repoPattern, repo)
	if err != nil {
		return fmt.Errorf("error validating repository: %w", err)
	}

	if !matched {
		return FieldValidationError{
			Field:   "repository",
			Value:   repo,
			Message: "must be a valid go module name (e.g., github.com/example/provider-name)",
		}
	}

	// Validate common patterns
	if !strings.Contains(repo, "/") {
		return FieldValidationError{
			Field:   "repository",
			Value:   repo,
			Message: "must include hosting provider (e.g., github.com/user/repo)",
		}
	}

	parts := strings.Split(repo, "/")
	if len(parts) < 3 {
		return FieldValidationError{
			Field:   "repository",
			Value:   repo,
			Message: "must follow pattern: host/user/repository",
		}
	}

	// Recommend provider- prefix for Crossplane convention
	repoName := parts[len(parts)-1]
	if !strings.HasPrefix(repoName, "provider-") {
		// This is a warning, not an error - maintain kubebuilder flexibility
		fmt.Printf("Warning: Repository name '%s' doesn't follow Crossplane convention 'provider-*'\n", repoName)
	}

	return nil
}

// ValidateResource validates resource parameters following kubebuilder conventions.
func (v *Validator) ValidateResource(res *resource.Resource) error {
	if res == nil {
		return FieldValidationError{
			Field:   "resource",
			Value:   "<nil>",
			Message: "resource is required",
		}
	}

	// Validate group - follows kubebuilder patterns
	if err := v.validateGroup(res.Group); err != nil {
		return err
	}

	// Validate version - follows kubebuilder patterns
	if err := v.validateVersion(res.Version); err != nil {
		return err
	}

	// Validate kind - follows kubebuilder patterns
	return v.validateKind(res.Kind)
}

// validateGroup validates API group name.
func (v *Validator) validateGroup(group string) error {
	if group == "" {
		return FieldValidationError{
			Field:   "group",
			Value:   group,
			Message: "group is required",
		}
	}

	// kubebuilder group validation - DNS-1123 label format
	groupPattern := `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	matched, err := regexp.MatchString(groupPattern, group)
	if err != nil {
		return fmt.Errorf("error validating group: %w", err)
	}

	if !matched {
		return FieldValidationError{
			Field:   "group",
			Value:   group,
			Message: "must be lowercase alphanumeric with hyphens (e.g., compute, storage)",
		}
	}

	// Length validation following Kubernetes conventions
	if len(group) > 63 {
		return FieldValidationError{
			Field:   "group",
			Value:   group,
			Message: "must be 63 characters or less",
		}
	}

	return nil
}

// validateVersion validates API version.
func (v *Validator) validateVersion(version string) error {
	if version == "" {
		return FieldValidationError{
			Field:   "version",
			Value:   version,
			Message: "version is required",
		}
	}

	// kubebuilder version validation pattern
	versionPattern := `^v\d+(alpha\d+|beta\d+)?$`
	matched, err := regexp.MatchString(versionPattern, version)
	if err != nil {
		return fmt.Errorf("error validating version: %w", err)
	}

	if !matched {
		return FieldValidationError{
			Field:   "version",
			Value:   version,
			Message: "must follow Kubernetes version format (e.g., v1alpha1, v1beta1, v1)",
		}
	}

	return nil
}

// validateKind validates resource kind.
func (v *Validator) validateKind(kind string) error {
	if kind == "" {
		return FieldValidationError{
			Field:   "kind",
			Value:   kind,
			Message: "kind is required",
		}
	}

	// kubebuilder kind validation - PascalCase
	kindPattern := `^[A-Z][a-zA-Z0-9]*$`
	matched, err := regexp.MatchString(kindPattern, kind)
	if err != nil {
		return fmt.Errorf("error validating kind: %w", err)
	}

	if !matched {
		return FieldValidationError{
			Field:   "kind",
			Value:   kind,
			Message: "must be PascalCase (e.g., Instance, Bucket, Database)",
		}
	}

	// Length validation
	if len(kind) > 63 {
		return FieldValidationError{
			Field:   "kind",
			Value:   kind,
			Message: "must be 63 characters or less",
		}
	}

	// Prevent common reserved words
	reservedKinds := []string{
		"Node", "Pod", "Service", "Deployment", "ConfigMap",
		"Secret", "Namespace", "CustomResourceDefinition",
	}
	for _, reserved := range reservedKinds {
		if strings.EqualFold(kind, reserved) {
			return FieldValidationError{
				Field:   "kind",
				Value:   kind,
				Message: fmt.Sprintf("'%s' is a reserved Kubernetes resource name", reserved),
			}
		}
	}

	return nil
}
