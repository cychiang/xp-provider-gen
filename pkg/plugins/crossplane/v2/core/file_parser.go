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
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// FileReader provides unified file reading functionality.
type FileReader struct{}

// NewFileReader creates a new file reader.
func NewFileReader() *FileReader {
	return &FileReader{}
}

// ReadFile reads the content of a file and returns it as a string.
func (f *FileReader) ReadFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return string(content), nil
}

// ParseSection represents a section configuration for parsing.
type ParseSection struct {
	Name        string
	StartMarker string
	EndMarker   string
	Pattern     *regexp.Regexp
	ExtractFunc func(line string, pattern *regexp.Regexp) string
}

// ParseConfig holds the configuration for file parsing.
type ParseConfig struct {
	Sections []ParseSection
}

// ParseResult holds the results of parsing.
type ParseResult struct {
	Results map[string][]string
}

// NewParseResult creates a new parse result.
func NewParseResult() *ParseResult {
	return &ParseResult{
		Results: make(map[string][]string),
	}
}

// Get returns the results for a specific section.
func (r *ParseResult) Get(sectionName string) []string {
	return r.Results[sectionName]
}

// UnifiedFileParser provides a unified approach to parsing Go files.
type UnifiedFileParser struct {
	content string
	config  ParseConfig
}

// NewUnifiedFileParser creates a new unified file parser.
func NewUnifiedFileParser(content string, config ParseConfig) *UnifiedFileParser {
	return &UnifiedFileParser{
		content: content,
		config:  config,
	}
}

// Parse performs the parsing according to the configuration.
func (p *UnifiedFileParser) Parse() *ParseResult {
	result := NewParseResult()

	// Initialize result maps for each section
	for _, section := range p.config.Sections {
		result.Results[section.Name] = []string{}
	}

	scanner := bufio.NewScanner(strings.NewReader(p.content))
	state := p.createParseState()

	for scanner.Scan() {
		line := scanner.Text()

		p.updateParseState(state, line)
		p.processLine(result, state, line)
	}

	return result
}

// parseState tracks which sections we're currently in.
type parseState struct {
	activeSections map[string]bool
}

func (p *UnifiedFileParser) createParseState() *parseState {
	state := &parseState{
		activeSections: make(map[string]bool),
	}

	// Initialize all sections as inactive
	for _, section := range p.config.Sections {
		state.activeSections[section.Name] = false
	}

	return state
}

func (p *UnifiedFileParser) updateParseState(state *parseState, line string) {
	for _, section := range p.config.Sections {
		// Check for section start
		if section.StartMarker != "" && strings.Contains(line, section.StartMarker) {
			state.activeSections[section.Name] = true
			continue
		}

		// Check for section end
		if section.EndMarker != "" && state.activeSections[section.Name] &&
			strings.Contains(line, section.EndMarker) {
			state.activeSections[section.Name] = false
		}
	}
}

func (p *UnifiedFileParser) processLine(result *ParseResult, state *parseState, line string) {
	for _, section := range p.config.Sections {
		if state.activeSections[section.Name] && section.Pattern != nil {
			if extracted := section.ExtractFunc(line, section.Pattern); extracted != "" {
				result.Results[section.Name] = append(result.Results[section.Name], extracted)
			}
		}
	}
}

// Common extraction functions

// ExtractImportLine extracts import statements.
func ExtractImportLine(line string, pattern *regexp.Regexp) string {
	if pattern.MatchString(line) {
		return strings.TrimSpace(line)
	}
	return ""
}

// ExtractMatchGroup extracts the first capturing group from a regex match.
func ExtractMatchGroup(line string, pattern *regexp.Regexp) string {
	if matches := pattern.FindStringSubmatch(line); len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// ExtractFullMatch extracts the full matched string.
func ExtractFullMatch(line string, pattern *regexp.Regexp) string {
	if pattern.MatchString(line) {
		return strings.TrimSpace(line)
	}
	return ""
}

// FileParserBuilder helps build common parsing configurations.
type FileParserBuilder struct {
	sections []ParseSection
}

// NewFileParserBuilder creates a new file parser builder.
func NewFileParserBuilder() *FileParserBuilder {
	return &FileParserBuilder{
		sections: []ParseSection{},
	}
}

// AddImportSection adds an import section configuration.
func (b *FileParserBuilder) AddImportSection(
	name, startMarker, endMarker string, pattern *regexp.Regexp,
) *FileParserBuilder {
	b.sections = append(b.sections, ParseSection{
		Name:        name,
		StartMarker: startMarker,
		EndMarker:   endMarker,
		Pattern:     pattern,
		ExtractFunc: ExtractImportLine,
	})
	return b
}

// AddMatchGroupSection adds a section that extracts regex match groups.
func (b *FileParserBuilder) AddMatchGroupSection(
	name, startMarker, endMarker string, pattern *regexp.Regexp,
) *FileParserBuilder {
	b.sections = append(b.sections, ParseSection{
		Name:        name,
		StartMarker: startMarker,
		EndMarker:   endMarker,
		Pattern:     pattern,
		ExtractFunc: ExtractMatchGroup,
	})
	return b
}

// AddFullMatchSection adds a section that extracts full pattern matches.
func (b *FileParserBuilder) AddFullMatchSection(
	name, startMarker, endMarker string, pattern *regexp.Regexp,
) *FileParserBuilder {
	b.sections = append(b.sections, ParseSection{
		Name:        name,
		StartMarker: startMarker,
		EndMarker:   endMarker,
		Pattern:     pattern,
		ExtractFunc: ExtractFullMatch,
	})
	return b
}

// Build creates the parse configuration.
func (b *FileParserBuilder) Build() ParseConfig {
	return ParseConfig{
		Sections: b.sections,
	}
}

// ParseFileWithConfig is a utility function for common file parsing operations.
func ParseFileWithConfig(filePath string, config ParseConfig) (map[string][]string, error) {
	reader := NewFileReader()
	content, err := reader.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	parser := NewUnifiedFileParser(content, config)
	result := parser.Parse()

	return result.Results, nil
}
