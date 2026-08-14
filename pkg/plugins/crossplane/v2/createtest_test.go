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
	"strings"
	"testing"

	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

const kindBucket = "Bucket"

func testKinds() []resource.Resource {
	return []resource.Resource{
		{GVK: resource.GVK{Group: "compute", Version: "v1alpha1", Kind: "Instance"}},
		{GVK: resource.GVK{Group: "storage", Version: "v1alpha1", Kind: kindBucket}},
	}
}

func TestResolveKind(t *testing.T) {
	kinds := testKinds()

	cases := map[string]struct {
		flag        string
		interactive bool
		stdin       string
		wantKind    string
		wantErr     string
	}{
		"flag exact":               {flag: kindBucket, wantKind: kindBucket},
		"flag case-insensitive":    {flag: "bucket", wantKind: kindBucket},
		"flag unknown":             {flag: "Nope", wantErr: "not in this project"},
		"multiple non-interactive": {wantErr: "--kind is required"},
		"interactive pick":         {interactive: true, stdin: "2\n", wantKind: kindBucket},
		"interactive bad input":    {interactive: true, stdin: "9\n", wantErr: "invalid choice"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var out strings.Builder
			res, err := resolveKind(kinds, tc.flag, tc.interactive, strings.NewReader(tc.stdin), &out)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("want error containing %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.Kind != tc.wantKind {
				t.Fatalf("want kind %q, got %q", tc.wantKind, res.Kind)
			}
		})
	}
}

func TestResolveKindSingleDefaults(t *testing.T) {
	one := testKinds()[:1]
	res, err := resolveKind(one, "", false, strings.NewReader(""), &strings.Builder{})
	if err != nil || res.Kind != "Instance" {
		t.Fatalf("single kind should be auto-picked: kind=%q err=%v", res.Kind, err)
	}
}

func TestResolveName(t *testing.T) {
	cases := map[string]struct {
		flag        string
		interactive bool
		stdin       string
		want        string
		wantErr     string
	}{
		"flag valid":              {flag: "drift-check", want: "drift-check"},
		"flag invalid chars":      {flag: "Drift_Check", wantErr: "invalid test name"},
		"missing non-interactive": {wantErr: "--name is required"},
		"interactive":             {interactive: true, stdin: "smoke-1\n", want: "smoke-1"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := resolveName(tc.flag, tc.interactive, strings.NewReader(tc.stdin), &strings.Builder{})
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("want error containing %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("want %q, got %q (err %v)", tc.want, got, err)
			}
		})
	}
}
