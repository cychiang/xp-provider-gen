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

package templates

import (
	"fmt"
	"io/fs"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

type CrossplaneTemplateFactory struct {
	config         config.Config
	initRegistry   map[TemplateType]TemplateBuilder
	apiRegistry    map[TemplateType]TemplateBuilder
	staticRegistry map[TemplateType]TemplateBuilder
}

func NewFactory(cfg config.Config) TemplateFactory {
	factory := &CrossplaneTemplateFactory{
		config:         cfg,
		initRegistry:   make(map[TemplateType]TemplateBuilder),
		apiRegistry:    make(map[TemplateType]TemplateBuilder),
		staticRegistry: make(map[TemplateType]TemplateBuilder),
	}

	factory.discoverAndRegisterTemplates()
	return factory
}

func (f *CrossplaneTemplateFactory) discoverAndRegisterTemplates() {
	fs.WalkDir(templateFS, "scaffolds", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		info := AnalyzeTemplatePath(path)
		templateType := TemplateType(info.GenerateTemplateType())

		switch info.Category {
		case "init":
			f.initRegistry[templateType] = NewInitTemplateBuilder(templateType)
		case "api":
			f.apiRegistry[templateType] = NewAPITemplateBuilder(templateType)
		case "static":
			f.staticRegistry[templateType] = NewStaticTemplateBuilder(templateType)
		}

		return nil
	})
}
func (f *CrossplaneTemplateFactory) CreateInitTemplate(templateType TemplateType, opts ...Option) (TemplateProduct, error) {
	builder, exists := f.initRegistry[templateType]
	if !exists {
		return nil, fmt.Errorf("unsupported init template type: %s", templateType)
	}

	product, err := builder.Build(f.config, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to build init template %s: %w", templateType, err)
	}

	return product, nil
}

func (f *CrossplaneTemplateFactory) CreateAPITemplate(templateType TemplateType, opts ...Option) (TemplateProduct, error) {
	builder, exists := f.apiRegistry[templateType]
	if !exists {
		return nil, fmt.Errorf("unsupported API template type: %s", templateType)
	}

	product, err := builder.Build(f.config, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to build API template %s: %w", templateType, err)
	}

	return product, nil
}

func (f *CrossplaneTemplateFactory) CreateStaticTemplate(templateType TemplateType, opts ...Option) (TemplateProduct, error) {
	builder, exists := f.staticRegistry[templateType]
	if !exists {
		return nil, fmt.Errorf("unsupported static template type: %s", templateType)
	}

	product, err := builder.Build(f.config, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to build static template %s: %w", templateType, err)
	}

	return product, nil
}

func (f *CrossplaneTemplateFactory) GetSupportedTypes() []TemplateType {
	var types []TemplateType

	for templateType := range f.initRegistry {
		types = append(types, templateType)
	}

	for templateType := range f.apiRegistry {
		types = append(types, templateType)
	}

	for templateType := range f.staticRegistry {
		types = append(types, templateType)
	}

	return types
}

func (f *CrossplaneTemplateFactory) FindTemplateTypeByPath(pathPattern string) (TemplateType, error) {
	patternMapping := map[string]string{
		"gomod":                    "RootGoMod",
		"makefile":                 "RootMakefile",
		"readme":                   "RootReadmeMd",
		"gitignore":                "RootGitignore",
		"maingo":                   "CmdProviderMainGo",
		"providerconfigtypes":      "ApisV1alpha1ProviderconfigTypes",
		"providerconfigregister":   "ApisV1alpha1Register",
		"crossplanepackage":        "PackageCrossplaneYaml",
		"configcontroller":         "InternalControllerConfigConfig",
		"controllerregister":       "InternalControllerRegister",
		"clusterdockerfile":        "ClusterImagesImageNameDockerfile",
		"clustermakefile":          "ClusterImagesImageNameMakefile",
		"versiongo":                "InternalVersionVersion",
		"license":                  "License",
		"docgo":                    "ApisDocGo",
		"examplesproviderconfig":   "ExamplesProviderConfigYaml",
		"apitypes":                 "ApisGroupVersionTypes",
		"apigroup":                 "ApisGroupVersionGroupversionInfo",
		"controller":               "InternalControllerKindController",
		"examplesmanagedresource":  "ExamplesGroupKindYaml",
		"boilerplate":              "HackBoilerplateGoTxt",
		"generatego":               "HackGenerateGo",
		"apis":                     "ApisRegister",
	}

	if mappedPattern, exists := patternMapping[strings.ToLower(pathPattern)]; exists {
		for templateType := range f.initRegistry {
			typeStr := string(templateType)
			if strings.Contains(typeStr, mappedPattern) {
				return templateType, nil
			}
		}
		for templateType := range f.apiRegistry {
			typeStr := string(templateType)
			if strings.Contains(typeStr, mappedPattern) {
				return templateType, nil
			}
		}
		for templateType := range f.staticRegistry {
			typeStr := string(templateType)
			if strings.Contains(typeStr, mappedPattern) {
				return templateType, nil
			}
		}
	}
	for templateType := range f.initRegistry {
		typeStr := string(templateType)
		if strings.Contains(strings.ToLower(typeStr), strings.ToLower(pathPattern)) {
			return templateType, nil
		}
	}
	for templateType := range f.apiRegistry {
		typeStr := string(templateType)
		if strings.Contains(strings.ToLower(typeStr), strings.ToLower(pathPattern)) {
			return templateType, nil
		}
	}
	for templateType := range f.staticRegistry {
		typeStr := string(templateType)
		if strings.Contains(strings.ToLower(typeStr), strings.ToLower(pathPattern)) {
			return templateType, nil
		}
	}
	return "", fmt.Errorf("template type not found for path pattern: %s", pathPattern)
}

func (f *CrossplaneTemplateFactory) GoMod() (TemplateProduct, error) {
	templateType, err := f.FindTemplateTypeByPath("gomod")
	if err != nil {
		return nil, err
	}
	return f.CreateInitTemplate(templateType)
}

func (f *CrossplaneTemplateFactory) Makefile() (TemplateProduct, error) {
	templateType, err := f.FindTemplateTypeByPath("makefile")
	if err != nil {
		return nil, err
	}
	return f.CreateInitTemplate(templateType)
}

func (f *CrossplaneTemplateFactory) README() (TemplateProduct, error) {
	templateType, err := f.FindTemplateTypeByPath("readme")
	if err != nil {
		return nil, err
	}
	return f.CreateInitTemplate(templateType)
}

func (f *CrossplaneTemplateFactory) GitIgnore() (TemplateProduct, error) {
	templateType, err := f.FindTemplateTypeByPath("gitignore")
	if err != nil {
		return nil, err
	}
	return f.CreateInitTemplate(templateType)
}

func (f *CrossplaneTemplateFactory) MainGo() (TemplateProduct, error) {
	templateType, err := f.FindTemplateTypeByPath("maingo")
	if err != nil {
		return nil, err
	}
	return f.CreateInitTemplate(templateType)
}

func (f *CrossplaneTemplateFactory) ProviderConfigTypes() (TemplateProduct, error) {
	templateType, err := f.FindTemplateTypeByPath("providerconfigtypes")
	if err != nil {
		return nil, err
	}
	return f.CreateInitTemplate(templateType)
}

func (f *CrossplaneTemplateFactory) ProviderConfigRegister() (TemplateProduct, error) {
	templateType, err := f.FindTemplateTypeByPath("providerconfigregister")
	if err != nil {
		return nil, err
	}
	return f.CreateInitTemplate(templateType)
}

func (f *CrossplaneTemplateFactory) CrossplanePackage() (TemplateProduct, error) {
	templateType, err := f.FindTemplateTypeByPath("crossplanepackage")
	if err != nil {
		return nil, err
	}
	return f.CreateInitTemplate(templateType)
}

func (f *CrossplaneTemplateFactory) ConfigController() (TemplateProduct, error) {
	templateType, err := f.FindTemplateTypeByPath("configcontroller")
	if err != nil {
		return nil, err
	}
	return f.CreateInitTemplate(templateType)
}

func (f *CrossplaneTemplateFactory) ControllerRegister() (TemplateProduct, error) {
	templateType, err := f.FindTemplateTypeByPath("controllerregister")
	if err != nil {
		return nil, err
	}
	return f.CreateInitTemplate(templateType)
}

func (f *CrossplaneTemplateFactory) ClusterDockerfile() (TemplateProduct, error) {
	templateType, err := f.FindTemplateTypeByPath("clusterdockerfile")
	if err != nil {
		return nil, err
	}
	return f.CreateInitTemplate(templateType)
}

func (f *CrossplaneTemplateFactory) ClusterMakefile() (TemplateProduct, error) {
	templateType, err := f.FindTemplateTypeByPath("clustermakefile")
	if err != nil {
		return nil, err
	}
	return f.CreateInitTemplate(templateType)
}

func (f *CrossplaneTemplateFactory) VersionGo() (TemplateProduct, error) {
	templateType, err := f.FindTemplateTypeByPath("versiongo")
	if err != nil {
		return nil, err
	}
	return f.CreateInitTemplate(templateType)
}

func (f *CrossplaneTemplateFactory) License() (TemplateProduct, error) {
	templateType, err := f.FindTemplateTypeByPath("license")
	if err != nil {
		return nil, err
	}
	return f.CreateStaticTemplate(templateType)
}

func (f *CrossplaneTemplateFactory) DocGo() (TemplateProduct, error) {
	templateType, err := f.FindTemplateTypeByPath("docgo")
	if err != nil {
		return nil, err
	}
	return f.CreateInitTemplate(templateType)
}

func (f *CrossplaneTemplateFactory) APITypes(force bool, res interface{}) (TemplateProduct, error) {
	templateType, err := f.FindTemplateTypeByPath("apitypes")
	if err != nil {
		return nil, err
	}
	opts := []Option{WithForce(force)}
	if res != nil {
		if r, ok := res.(*resource.Resource); ok {
			opts = append(opts, WithResource(r))
		}
	}
	return f.CreateAPITemplate(templateType, opts...)
}

func (f *CrossplaneTemplateFactory) APIGroup(res interface{}) (TemplateProduct, error) {
	templateType, err := f.FindTemplateTypeByPath("apigroup")
	if err != nil {
		return nil, err
	}
	opts := []Option{}
	if res != nil {
		if r, ok := res.(*resource.Resource); ok {
			opts = append(opts, WithResource(r))
		}
	}
	return f.CreateAPITemplate(templateType, opts...)
}

func (f *CrossplaneTemplateFactory) Controller(force bool, res interface{}) (TemplateProduct, error) {
	templateType, err := f.FindTemplateTypeByPath("controller")
	if err != nil {
		return nil, err
	}
	opts := []Option{WithForce(force)}
	if res != nil {
		if r, ok := res.(*resource.Resource); ok {
			opts = append(opts, WithResource(r))
		}
	}
	return f.CreateAPITemplate(templateType, opts...)
}

func (f *CrossplaneTemplateFactory) ExamplesProviderConfig() (TemplateProduct, error) {
	templateType, err := f.FindTemplateTypeByPath("examplesproviderconfig")
	if err != nil {
		return nil, err
	}
	return f.CreateInitTemplate(templateType)
}

func (f *CrossplaneTemplateFactory) ExamplesManagedResource(res interface{}) (TemplateProduct, error) {
	templateType, err := f.FindTemplateTypeByPath("examplesmanagedresource")
	if err != nil {
		return nil, err
	}
	opts := []Option{}
	if res != nil {
		if r, ok := res.(*resource.Resource); ok {
			opts = append(opts, WithResource(r))
		}
	}
	return f.CreateAPITemplate(templateType, opts...)
}