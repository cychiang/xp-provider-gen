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

package engine

import (
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

type TemplateType string

type TemplateProduct interface {
	machinery.Template
	machinery.Builder
	GetTemplateType() TemplateType
	Configure(cfg config.Config) error
	SetResource(res *resource.Resource) error
}

type TemplateFactory interface {
	CreateInitTemplate(templateType TemplateType, opts ...Option) (TemplateProduct, error)
	CreateAPITemplate(templateType TemplateType, opts ...Option) (TemplateProduct, error)
	CreateStaticTemplate(templateType TemplateType, opts ...Option) (TemplateProduct, error)
	GetSupportedTypes() []TemplateType
	GetInitTemplates(opts ...Option) ([]TemplateProduct, error)
	GetAPITemplates(opts ...Option) ([]TemplateProduct, error)
	GetStaticTemplates(opts ...Option) ([]TemplateProduct, error)
}

type TemplateBuilder interface {
	Build(cfg config.Config, opts ...Option) (TemplateProduct, error)
	GetTemplateType() TemplateType
}

type Option func(*TemplateOptions)

type TemplateOptions struct {
	Force      bool
	Resource   *resource.Resource
	CustomData map[string]interface{}
}

func WithForce(force bool) Option {
	return func(opts *TemplateOptions) { opts.Force = force }
}

func WithResource(resource *resource.Resource) Option {
	return func(opts *TemplateOptions) { opts.Resource = resource }
}

func WithCustomData(data map[string]interface{}) Option {
	return func(opts *TemplateOptions) { opts.CustomData = data }
}
