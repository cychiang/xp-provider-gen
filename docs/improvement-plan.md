# Project Improvement Plan

## Overview

This document outlines potential improvements for the xp-provider-gen project based on a comprehensive code review. The plan focuses on code quality, maintainability, testing, and developer experience.

---

## 1. Code Quality & Maintainability

### 1.1 Reduce Code Duplication in Builders

**Current State:**
- `builders.go` has 330 lines with significant duplication across `InitTemplateBuilder`, `APITemplateBuilder`, and `StaticTemplateBuilder`
- Each builder has nearly identical `findTemplateInfo()` methods (only category differs)
- Project name extraction logic duplicated 3 times

**Problems:**
- High maintenance cost - changes need to be applied in 3 places
- Risk of inconsistency between builders
- Violates DRY principle

**Proposed Solution:**
```go
// Extract common logic into shared functions
func extractProjectName(cfg config.Config) string {
    projectName := cfg.GetProjectName()
    if projectName == "" {
        repo := cfg.GetRepository()
        if repo != "" {
            parts := strings.Split(repo, "/")
            if len(parts) > 0 {
                projectName = parts[len(parts)-1]
            }
        }
    }
    return projectName
}

// Generic template finder
func findTemplateByType(category TemplateCategory, templateType TemplateType) (TemplateInfo, error) {
    var foundInfo TemplateInfo
    found := false

    err := walkTemplateFS("files", func(path string, isDir bool) error {
        if isDir || !strings.HasSuffix(path, ".tmpl") {
            return nil
        }

        info := AnalyzeTemplatePath(path)
        if info.Category == category && TemplateType(info.GenerateTemplateType()) == templateType {
            foundInfo = info
            found = true
        }
        return nil
    })

    if err != nil {
        return TemplateInfo{}, err
    }
    if !found {
        return TemplateInfo{}, fmt.Errorf("template not found for type: %s", templateType)
    }

    return foundInfo, nil
}
```

**Benefits:**
- Single source of truth
- Easier to maintain and test
- Reduced line count (~30% reduction in builders.go)

**Priority:** High
**Effort:** Medium (2-4 hours)

---

### 1.2 Simplify Factory Pattern

**Current State:**
- Factory has 3 separate registries (init, api, static)
- Methods like `GetInitTemplates()`, `GetAPITemplates()`, `GetStaticTemplates()` are nearly identical
- Total of 173 lines in `factory.go`

**Problems:**
- Unnecessary complexity
- More code to maintain
- Category-specific methods don't add significant value

**Proposed Solution:**
```go
type CrossplaneTemplateFactory struct {
    config   config.Config
    registry map[TemplateType]*TemplateRegistration
}

type TemplateRegistration struct {
    Category TemplateCategory
    Builder  TemplateBuilder
}

// Simplified methods
func (f *CrossplaneTemplateFactory) GetTemplatesByCategory(category TemplateCategory, opts ...Option) ([]TemplateProduct, error) {
    var templates []TemplateProduct

    for _, reg := range f.registry {
        if reg.Category == category {
            product, err := reg.Builder.Build(f.config, opts...)
            if err != nil {
                return nil, err
            }
            templates = append(templates, product)
        }
    }

    return templates, nil
}
```

**Benefits:**
- Cleaner, more maintainable code
- Single registry to manage
- Easier to extend with new categories

**Priority:** Medium
**Effort:** Medium (3-4 hours)

---

## 2. Testing & Quality Assurance

### 2.1 Improve Test Coverage

**Current State:**
- Overall coverage: ~28%
- Engine package: 0% coverage
- Scaffolders: 0% coverage
- Version package: 0% coverage
- cmd/: 0% coverage

**Problems:**
- Low confidence in refactoring
- Bugs harder to catch early
- No regression protection

**Proposed Solution:**

**Phase 1: Core Engine Testing (Priority: High)**
```go
// Test template discovery
func TestTemplateDiscovery(t *testing.T) {
    factory := NewFactory(testConfig)
    types := factory.GetSupportedTypes()

    // Verify all expected templates are discovered
    assert.Contains(t, types, TemplateType("api.apis.group.version.kind_types"))
    assert.Contains(t, types, TemplateType("init.cmd.provider.main"))
}

// Test builder logic
func TestAPITemplateBuilder(t *testing.T) {
    builder := NewAPITemplateBuilder("api.apis.group.version.kind_types")

    product, err := builder.Build(testConfig, WithResource(testResource))

    assert.NoError(t, err)
    assert.Equal(t, "apis/compute/v1alpha1/instance_types.go", product.GetPath())
}

// Test path generation
func TestGenerateOutputPath(t *testing.T) {
    tests := []struct {
        name         string
        templatePath string
        replacements map[string]string
        expected     string
    }{
        {
            name:         "API type with group/version/kind",
            templatePath: "files/api/apis/GROUP/VERSION/KIND_types.go.tmpl",
            replacements: map[string]string{
                "GROUP":   "compute",
                "VERSION": "v1alpha1",
                "KIND":    "instance",
            },
            expected: "apis/compute/v1alpha1/instance_types.go",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            info := AnalyzeTemplatePath(tt.templatePath)
            result := generateOutputPath(info, tt.replacements)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

**Phase 2: Integration Tests (Priority: Medium)**
```go
func TestEndToEndInit(t *testing.T) {
    tmpDir := t.TempDir()

    // Run init command
    err := runInit(tmpDir, "example.com", "github.com/test/provider-test")
    assert.NoError(t, err)

    // Verify generated structure
    assert.FileExists(t, filepath.Join(tmpDir, "go.mod"))
    assert.FileExists(t, filepath.Join(tmpDir, "Makefile"))
    assert.DirExists(t, filepath.Join(tmpDir, "apis/v1alpha1"))
}
```

**Target Coverage:** 60-70% (realistic goal)

**Priority:** High
**Effort:** High (8-12 hours)

---

### 2.2 Add Linter Configuration

**Current State:**
- No `.golangci.yml` configuration
- No consistent linting rules across codebase
- Makefile references linter but no config exists

**Proposed Solution:**
```yaml
# .golangci.yml
linters:
  enable:
    - gofmt
    - goimports
    - govet
    - errcheck
    - staticcheck
    - unused
    - gosimple
    - ineffassign
    - misspell
    - unconvert
    - dupl
    - gocritic

linters-settings:
  dupl:
    threshold: 100
  gocritic:
    enabled-tags:
      - diagnostic
      - performance
      - style

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - dupl
        - errcheck

run:
  timeout: 5m
  tests: true
```

**Benefits:**
- Consistent code quality
- Catch common bugs automatically
- Better code review process

**Priority:** Medium
**Effort:** Low (1 hour)

---

## 3. Developer Experience

### 3.1 Add Integration Test Script

**Current State:**
- Makefile references `./scripts/integration-test.sh` but script doesn't exist
- No automated E2E testing
- Manual testing required

**Proposed Solution:**
```bash
#!/bin/bash
# scripts/integration-test.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BINARY="$PROJECT_ROOT/bin/xp-provider-gen"

echo "=== Integration Test Suite ==="

# Test 1: Init and single API
TEST_DIR=$(mktemp -d)
trap "rm -rf $TEST_DIR" EXIT

echo "Test 1: Basic init + create api"
cd "$TEST_DIR"
$BINARY init --domain=test.io --repo=github.com/test/provider-test
$BINARY create api --group=compute --version=v1alpha1 --kind=Instance

echo "Validating generated code..."
make generate
make build

echo "✅ Test 1 passed"

# Test 2: Multiple APIs
echo "Test 2: Multiple resources"
$BINARY create api --group=storage --version=v1 --kind=Bucket
$BINARY create api --group=network --version=v1beta1 --kind=VPC

make generate
make build

echo "✅ Test 2 passed"

echo "=== All tests passed ==="
```

**Priority:** High
**Effort:** Low (2 hours)

---

### 3.2 Improve Documentation Structure

**Current State:**
- 3 summary documents that overlap
- No architecture documentation
- No contributing guide
- No development guide

**Proposed Solution:**

**Reorganize docs/**
```
docs/
├── architecture/
│   ├── overview.md          # High-level architecture
│   ├── template-system.md   # Template discovery & rendering
│   └── plugin-integration.md # Kubebuilder plugin details
├── development/
│   ├── setup.md             # Development environment setup
│   ├── testing.md           # Testing guidelines
│   └── contributing.md      # Contribution guide
└── CHANGELOG.md             # Track changes
```

**Consolidate summaries:**
- Remove `cleanup-summary.md`, `final-summary.md`, `refactoring-complete.md`
- Replace with `CHANGELOG.md` for historical tracking
- Create focused architecture docs

**Priority:** Medium
**Effort:** Medium (4-6 hours)

---

### 3.3 Add Pre-commit Hooks

**Current State:**
- No automated checks before commits
- Quality issues caught late in CI/PR review

**Proposed Solution:**
```bash
# .git/hooks/pre-commit (or use pre-commit framework)
#!/bin/bash

echo "Running pre-commit checks..."

# Format check
if ! make fmt; then
    echo "❌ Code formatting failed"
    exit 1
fi

# Vet check
if ! make vet; then
    echo "❌ Go vet failed"
    exit 1
fi

# Tests
if ! make test; then
    echo "❌ Tests failed"
    exit 1
fi

echo "✅ Pre-commit checks passed"
```

**Priority:** Low
**Effort:** Low (1 hour)

---

## 4. Code Organization

### 4.1 Split Large Files

**Current State:**
- `builders.go`: 330 lines (3 builders)
- `factory.go`: 173 lines (1 factory + methods)
- Files mixing multiple concerns

**Proposed Solution:**
```
templates/engine/
├── factory.go                    # Factory only
├── builder.go                    # Common builder logic
├── builder_init.go               # Init-specific
├── builder_api.go                # API-specific
├── builder_static.go             # Static-specific
├── template_info.go              # Template analysis
└── path.go                       # Path generation
```

**Benefits:**
- Easier to navigate
- Clearer separation of concerns
- Better git history

**Priority:** Low
**Effort:** Medium (3-4 hours)

---

## 5. CI/CD Enhancements

### 5.1 Add GitHub Actions Workflow

**Current State:**
- No CI/CD automation
- Manual testing only

**Proposed Solution:**
```yaml
# .github/workflows/ci.yml
name: CI

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Run tests
        run: make test

      - name: Run linter
        run: make lint

      - name: Integration tests
        run: make integration-test

  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Build binary
        run: make build
```

**Priority:** High
**Effort:** Low (2 hours)

---

## Implementation Roadmap

### Phase 1: Foundation (Week 1)
- [ ] Add integration test script (3.1)
- [ ] Add linter configuration (2.2)
- [ ] Set up CI/CD (5.1)
- [ ] Start engine tests (2.1 Phase 1)

### Phase 2: Code Quality (Week 2)
- [ ] Reduce builder duplication (1.1)
- [ ] Simplify factory pattern (1.2)
- [ ] Continue test coverage (2.1)

### Phase 3: Organization (Week 3)
- [ ] Reorganize documentation (3.2)
- [ ] Split large files (4.1)
- [ ] Add pre-commit hooks (3.3)

### Phase 4: Polish (Week 4)
- [ ] Complete test coverage to 60%+
- [ ] Final documentation review
- [ ] Performance optimization if needed

---

## Success Metrics

- **Test Coverage:** 28% → 60%+
- **Code Duplication:** Reduce by ~40% in engine/
- **Documentation:** Complete architecture + dev guides
- **CI/CD:** Automated testing on all PRs
- **Developer Experience:** < 5 minutes to start contributing

---

## Non-Goals

- Changing the CLI interface (maintain backward compatibility)
- Rewriting template system (works well)
- Adding new features (focus on quality)

---

## Conclusion

This plan focuses on **quality, maintainability, and developer experience** without breaking existing functionality. All changes maintain 100% backward compatibility.

**Recommended Starting Point:** Phase 1 (Foundation) - establishes quality gates and testing infrastructure for safe refactoring.