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

// Field names used in validation errors.
const (
	fieldDomain     = "domain"
	fieldRepository = "repository"
	fieldGroup      = "group"
	fieldVersion    = "version"
	fieldKind       = "kind"
)

// maxNameLength is the Kubernetes DNS label limit applied to groups and kinds.
const maxNameLength = 63

// Input patterns, compiled once. Each mirrors the kubebuilder/Kubernetes
// convention for the field it guards.
var (
	domainRe  = regexp.MustCompile(`^[a-z0-9]+([-.][a-z0-9]+)*\.[a-z]{2,}$`)
	repoRe    = regexp.MustCompile(`^[a-z0-9.-]+/[a-z0-9._-]+/[a-z0-9._-]+$`)
	groupRe   = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`) // DNS-1123 label
	versionRe = regexp.MustCompile(`^v\d+(alpha\d+|beta\d+)?$`)
	kindRe    = regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`) // PascalCase
)

// reservedKinds are Kubernetes core kinds a managed resource must not shadow.
var reservedKinds = []string{
	"Node", "Pod", "Service", "Deployment", "ConfigMap",
	"Secret", "Namespace", "CustomResourceDefinition",
}

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

// checkRequired rejects an empty value.
func checkRequired(field, value string) error {
	if value == "" {
		return FieldValidationError{Field: field, Value: value, Message: field + " is required"}
	}
	return nil
}

// checkPattern rejects a value that does not match its field's pattern. message
// describes the accepted form, so it doubles as the user-facing fix.
func checkPattern(field, value string, re *regexp.Regexp, message string) error {
	if !re.MatchString(value) {
		return FieldValidationError{Field: field, Value: value, Message: message}
	}
	return nil
}

// checkLength rejects a value longer than the Kubernetes name limit.
func checkLength(field, value string) error {
	if len(value) > maxNameLength {
		return FieldValidationError{
			Field:   field,
			Value:   value,
			Message: fmt.Sprintf("must be %d characters or less", maxNameLength),
		}
	}
	return nil
}

// ValidateDomain validates the domain follows kubebuilder conventions.
func (v *Validator) ValidateDomain(domain string) error {
	if err := checkRequired(fieldDomain, domain); err != nil {
		return err
	}
	if err := checkPattern(fieldDomain, domain, domainRe,
		"must be a valid domain name (e.g., example.com)"); err != nil {
		return err
	}
	if strings.HasSuffix(domain, ".local") {
		return FieldValidationError{
			Field:   fieldDomain,
			Value:   domain,
			Message: ".local domains are not recommended for production use",
		}
	}
	return nil
}

// ValidateRepository validates the repository follows go module conventions.
// The pattern already requires exactly host/user/repository, so no further
// structural checks are needed.
func (v *Validator) ValidateRepository(repo string) error {
	if err := checkRequired(fieldRepository, repo); err != nil {
		return err
	}
	if err := checkPattern(fieldRepository, repo, repoRe,
		"must be a valid go module name (e.g., github.com/example/provider-name)"); err != nil {
		return err
	}

	// A non-provider-* name is legal but unconventional for Crossplane; warn
	// rather than reject, matching kubebuilder's flexibility.
	parts := strings.Split(repo, "/")
	if repoName := parts[len(parts)-1]; !strings.HasPrefix(repoName, "provider-") {
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
	if err := v.validateGroup(res.Group); err != nil {
		return err
	}
	if err := v.validateVersion(res.Version); err != nil {
		return err
	}
	return v.validateKind(res.Kind)
}

// validateGroup validates API group name.
func (v *Validator) validateGroup(group string) error {
	if err := checkRequired(fieldGroup, group); err != nil {
		return err
	}
	if err := checkPattern(fieldGroup, group, groupRe,
		"must be lowercase alphanumeric with hyphens (e.g., compute, storage)"); err != nil {
		return err
	}
	return checkLength(fieldGroup, group)
}

// validateVersion validates API version.
func (v *Validator) validateVersion(version string) error {
	if err := checkRequired(fieldVersion, version); err != nil {
		return err
	}
	return checkPattern(fieldVersion, version, versionRe,
		"must follow Kubernetes version format (e.g., v1alpha1, v1beta1, v1)")
}

// validateKind validates resource kind.
func (v *Validator) validateKind(kind string) error {
	if err := checkRequired(fieldKind, kind); err != nil {
		return err
	}
	if err := checkPattern(fieldKind, kind, kindRe,
		"must be PascalCase (e.g., Instance, Bucket, Database)"); err != nil {
		return err
	}
	if err := checkLength(fieldKind, kind); err != nil {
		return err
	}
	for _, reserved := range reservedKinds {
		if strings.EqualFold(kind, reserved) {
			return FieldValidationError{
				Field:   fieldKind,
				Value:   kind,
				Message: fmt.Sprintf("'%s' is a reserved Kubernetes resource name", reserved),
			}
		}
	}
	return nil
}
