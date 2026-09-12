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

package main

import "testing"

// Sample Go versions reused across this file's test tables.
const (
	testGoVersionA = "1.26.8"
	testGoVersionB = "1.26.5"
)

func TestExtractGoDirective(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
		wantErr bool
	}{
		{
			name:    "this repo's go.mod shape",
			content: "module github.com/cychiang/xp-provider-gen\n\ngo " + testGoVersionA + "\n\nrequire (\n)\n",
			want:    testGoVersionA,
		},
		{
			name:    "two-component directive (older go.mod convention)",
			content: "module example.com/x\n\ngo 1.17\n",
			want:    "1.17",
		},
		{
			name:    "directive with a patch and a toolchain line following",
			content: "module example.com/x\n\ngo " + testGoVersionB + "\n\ntoolchain go" + testGoVersionB + "\n",
			want:    testGoVersionB,
		},
		{
			name:    "no go directive at all (pre-modules-era go.mod)",
			content: "module github.com/pkg/errors\n",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractGoDirective(tt.content)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("extractGoDirective() = %q, want an error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("extractGoDirective() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("extractGoDirective() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractDockerGoTag(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
		wantErr bool
	}{
		{
			name:    "this repo's Dockerfile shape",
			content: "FROM golang:" + testGoVersionA + "-alpine AS builder\n\nENV GOTOOLCHAIN=auto\n",
			want:    testGoVersionA,
		},
		{
			name:    "no golang FROM line",
			content: "FROM alpine:3.24\n",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractDockerGoTag(tt.content)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("extractDockerGoTag() = %q, want an error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("extractDockerGoTag() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("extractDockerGoTag() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCompareGoVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{testGoVersionA, testGoVersionA, 0},
		{testGoVersionB, testGoVersionA, -1},
		{testGoVersionA, testGoVersionB, 1},
		{"1.26.0", "1.27.0", -1},
		{"1.9.0", "1.10.0", -1}, // numeric, not lexicographic, comparison
		{"1.26", "1.26.0", 0},   // a missing patch component means 0
		{"1.26", "1.26.1", -1},
	}
	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			got, err := compareGoVersions(tt.a, tt.b)
			if err != nil {
				t.Fatalf("compareGoVersions(%q, %q) error: %v", tt.a, tt.b, err)
			}
			if got != tt.want {
				t.Errorf("compareGoVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestEscapeModulePath(t *testing.T) {
	tests := []struct{ in, want string }{
		{"github.com/pkg/errors", "github.com/pkg/errors"},
		{"github.com/crossplane/crossplane-tools", "github.com/crossplane/crossplane-tools"},
		{"dario.cat/mergo", "dario.cat/mergo"},
		{"k8s.io/apimachinery", "k8s.io/apimachinery"},
		// The one case that needs escaping among this repo's actual
		// dependencies: nothing in dependencies.yaml has an uppercase
		// letter today, so this pins the mechanism against a synthetic
		// example per the module proxy's documented case-encoding.
		{"github.com/Masterminds/semver", "github.com/!masterminds/semver"},
	}
	for _, tt := range tests {
		if got := escapeModulePath(tt.in); got != tt.want {
			t.Errorf("escapeModulePath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
