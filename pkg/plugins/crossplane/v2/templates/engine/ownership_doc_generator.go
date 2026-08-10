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
	"fmt"
	"io/fs"
	"sort"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
	"github.com/cychiang/xp-provider-gen/pkg/templates"
)

// docPlaceholders rewrite the scaffolding placeholders into reader-friendly
// form for the generated ownership map.
var docPlaceholders = map[string]string{
	"GROUP":              "<group>",
	"VERSION":            "<version>",
	"KIND":               "<kind>",
	placeholderImageName: "<provider>",
}

// ownershipDocPath is where the generated ownership map lives in a provider.
const ownershipDocPath = "docs/ownership.md"

// OwnershipDocGenerator renders docs/ownership.md: the authoritative list of
// which files a provider's owner may edit and which `update` overwrites.
//
// It is generated rather than written by hand so the published contract cannot
// drift from the enforced one — it reads the same template bodies the ownership
// gate reads.
type OwnershipDocGenerator struct {
	machinery.TemplateMixin

	ToolOwned []string
	UserOwned []string
}

var _ machinery.Template = &OwnershipDocGenerator{}

// NewOwnershipDocGenerator builds the ownership doc generator. siblings are
// the other generator-emitted files (invisible to the template-FS walk); their
// path and ownership are read from the generators themselves — OverwriteFile
// means tool-owned, SkipFile means seeded once and then the user's.
func NewOwnershipDocGenerator(siblings ...machinery.Template) *OwnershipDocGenerator {
	g := &OwnershipDocGenerator{}
	processor := core.NewTemplatePathProcessor()

	// Walk the template FS directly. Template base names are not unique
	// (Makefile.tmpl exists twice), so any name-keyed map would drop a file.
	//
	// GenerateOutputPath, not GetOutputPath: only the former strips the
	// "project/" prefix and applies the path placeholders, so the doc lists
	// paths that actually exist in a generated provider.
	err := fs.WalkDir(templates.TemplateFS, "files", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !processor.IsTemplateFile(path) {
			return err
		}
		body, readErr := templates.TemplateFS.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		g.add(processor.GenerateOutputPath(path, docPlaceholders), core.IsToolOwned(body))
		return nil
	})
	if err != nil {
		// The template FS is embedded at compile time, so a walk failure is a
		// build defect — fail loudly rather than emit an incomplete doc.
		panic(fmt.Errorf("enumerating embedded templates for the ownership doc: %w", err))
	}

	g.add(ownershipDocPath, true)
	for _, sib := range siblings {
		if err := sib.SetTemplateDefaults(); err != nil {
			panic(fmt.Errorf("deriving generator outputs for the ownership doc: %w", err))
		}
		g.add(sib.GetPath(), sib.GetIfExistsAction() == machinery.OverwriteFile)
	}

	sort.Strings(g.ToolOwned)
	sort.Strings(g.UserOwned)
	return g
}

// add records one output path in the appropriate bucket.
func (f *OwnershipDocGenerator) add(path string, toolOwned bool) {
	if toolOwned {
		f.ToolOwned = append(f.ToolOwned, path)
		return
	}
	f.UserOwned = append(f.UserOwned, path)
}

func (f *OwnershipDocGenerator) SetTemplateDefaults() error {
	f.Path = ownershipDocPath
	f.IfExistsAction = machinery.OverwriteFile
	f.TemplateBody = ownershipDocTemplate
	return nil
}

// The generated header must appear literally for core.IsToolOwned to match it,
// but Markdown has no comment syntax — an HTML comment satisfies both.
const ownershipDocTemplate = `<!-- ` + core.GeneratedHeader + ` -->

# File ownership

This provider is scaffolded by ` + "`xp-provider-gen`" + `. Every file falls into exactly
one bucket, decided by whether it carries this header:

    ` + core.GeneratedHeader + `

**This file is generated.** Editing it has no effect — it is rewritten by
` + "`xp-provider-gen update`" + `.

## Tool-owned — overwritten by ` + "`update`" + `

Do not edit these. Your changes are lost on the next update, and everything in
here is framework wiring you should not need to touch.
{{ range .ToolOwned }}
- ` + "`{{ . }}`" + `
{{- end }}

## Yours — never touched

The generator creates these once and then leaves them alone forever.
{{ range .UserOwned }}
- ` + "`{{ . }}`" + `
{{- end }}

## Also generated

` + "`zz_generated.*.go`" + ` and ` + "`package/crds/*`" + ` are produced by ` + "`make generate`" + `
(controller-gen and angryjet), not by ` + "`xp-provider-gen`" + `. Do not edit them either.

## The seam names

Tool-owned code calls these by name. Renaming any of them breaks the build:

| Name | Where you define it |
|---|---|
| ` + "`Client`" + `, ` + "`NewClient`" + ` | ` + "`internal/provider/client.go`" + ` |
| ` + "`Flags`" + `, ` + "`Configure`" + ` | ` + "`internal/provider/options.go`" + ` |
| ` + "`NewExternal`" + `, ` + "`ReconcilerOptions`" + ` | ` + "`internal/controller/<kind>/external.go`" + ` |
`
