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

package v2

import (
	"testing"

	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

func TestValidator_ValidateDomain(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name    string
		domain  string
		wantErr bool
	}{
		{
			name:    "valid domain",
			domain:  "example.com",
			wantErr: false,
		},
		{
			name:    "valid subdomain",
			domain:  "api.example.com",
			wantErr: false,
		},
		{
			name:    "empty domain",
			domain:  "",
			wantErr: true,
		},
		{
			name:    "invalid domain format",
			domain:  "invalid-domain",
			wantErr: true,
		},
		{
			name:    "local domain warning",
			domain:  "example.local",
			wantErr: true,
		},
		{
			name:    "valid org domain",
			domain:  "crossplane.io",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateDomain(tt.domain)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDomain() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_ValidateRepository(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name    string
		repo    string
		wantErr bool
	}{
		{
			name:    "valid github repo",
			repo:    "github.com/example/provider-test",
			wantErr: false,
		},
		{
			name:    "valid gitlab repo",
			repo:    "gitlab.com/example/provider-test",
			wantErr: false,
		},
		{
			name:    "empty repository",
			repo:    "",
			wantErr: true,
		},
		{
			name:    "missing host",
			repo:    "example/provider-test",
			wantErr: true,
		},
		{
			name:    "too short",
			repo:    "example.com/test",
			wantErr: true,
		},
		{
			name:    "valid private repo",
			repo:    "git.company.com/team/provider-internal",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateRepository(tt.repo)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRepository() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_ValidateResource(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name     string
		resource *resource.Resource
		wantErr  bool
	}{
		{
			name: "valid resource",
			resource: &resource.Resource{
				GVK: resource.GVK{
					Group:   "compute",
					Version: "v1alpha1",
					Kind:    "Instance",
				},
			},
			wantErr: false,
		},
		{
			name: "valid beta resource",
			resource: &resource.Resource{
				GVK: resource.GVK{
					Group:   "storage",
					Version: "v1beta1",
					Kind:    "Bucket",
				},
			},
			wantErr: false,
		},
		{
			name:     "nil resource",
			resource: nil,
			wantErr:  true,
		},
		{
			name: "empty group",
			resource: &resource.Resource{
				GVK: resource.GVK{
					Group:   "",
					Version: "v1alpha1",
					Kind:    "Instance",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid version",
			resource: &resource.Resource{
				GVK: resource.GVK{
					Group:   "compute",
					Version: "alpha1",
					Kind:    "Instance",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid kind",
			resource: &resource.Resource{
				GVK: resource.GVK{
					Group:   "compute",
					Version: "v1alpha1",
					Kind:    "instance",
				},
			},
			wantErr: true,
		},
		{
			name: "reserved kind",
			resource: &resource.Resource{
				GVK: resource.GVK{
					Group:   "compute",
					Version: "v1alpha1",
					Kind:    "Pod",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateResource(tt.resource)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateResource() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}