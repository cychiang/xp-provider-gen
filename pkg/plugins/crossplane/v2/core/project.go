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

package core

import (
	"fmt"
	"os"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

type ProjectFile struct {
	config config.Config
}

func NewProjectFile(cfg config.Config) *ProjectFile {
	return &ProjectFile{config: cfg}
}

func (p *ProjectFile) Save() error {
	bytes, err := p.config.MarshalYAML()
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile("PROJECT", bytes, 0644); err != nil {
		return fmt.Errorf("write PROJECT file: %w", err)
	}

	return nil
}

func (p *ProjectFile) AddResource(res resource.Resource) error {
	if err := p.config.AddResource(res); err != nil {
		return fmt.Errorf("add resource to config: %w", err)
	}

	return p.Save()
}