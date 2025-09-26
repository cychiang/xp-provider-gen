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
	"bufio"
	"regexp"
	"strings"
)

// FileParser defines the common interface for parsing Go files.
type FileParser interface {
	Parse() ([]string, []string)
}

// BaseParser provides common parsing functionality.
type BaseParser struct {
	content            string
	importPattern      *regexp.Regexp
	secondPattern      *regexp.Regexp
	importSectionStart string
	importSectionEnd   string
	secondSectionStart string
	secondSectionEnd   string
}

// NewBaseParser creates a new base parser.
func NewBaseParser(content string, importPattern, secondPattern *regexp.Regexp,
	importStart, importEnd, secondStart, secondEnd string,
) *BaseParser {
	return &BaseParser{
		content:            content,
		importPattern:      importPattern,
		secondPattern:      secondPattern,
		importSectionStart: importStart,
		importSectionEnd:   importEnd,
		secondSectionStart: secondStart,
		secondSectionEnd:   secondEnd,
	}
}

// Parse performs the common parsing logic.
func (p *BaseParser) Parse() ([]string, []string) {
	var imports, seconds []string

	scanner := bufio.NewScanner(strings.NewReader(p.content))
	state := &parseState{}

	for scanner.Scan() {
		line := scanner.Text()

		p.updateParseState(state, line)
		imports = p.processImportLine(imports, state, line)
		seconds = p.processSecondLine(seconds, state, line)
	}

	return imports, seconds
}

type parseState struct {
	inImports bool
	inSeconds bool
}

func (p *BaseParser) updateParseState(state *parseState, line string) {
	// Update state for imports
	if strings.Contains(line, p.importSectionStart) {
		state.inImports = true
		return
	}
	if state.inImports && strings.Contains(line, p.importSectionEnd) {
		state.inImports = false
		return
	}

	// Update state for second section
	if strings.Contains(line, p.secondSectionStart) {
		state.inSeconds = true
		return
	}
	if state.inSeconds && strings.Contains(line, p.secondSectionEnd) {
		state.inSeconds = false
	}
}

func (p *BaseParser) processImportLine(imports []string, state *parseState, line string) []string {
	if state.inImports {
		if importLine := p.parseImport(line); importLine != "" {
			imports = append(imports, importLine)
		}
	}
	return imports
}

func (p *BaseParser) processSecondLine(seconds []string, state *parseState, line string) []string {
	if state.inSeconds {
		if secondLine := p.parseSecond(line); secondLine != "" {
			seconds = append(seconds, secondLine)
		}
	}
	return seconds
}

func (p *BaseParser) parseImport(line string) string {
	if p.importPattern.MatchString(line) {
		return strings.TrimSpace(line)
	}
	return ""
}

func (p *BaseParser) parseSecond(line string) string {
	if matches := p.secondPattern.FindStringSubmatch(line); len(matches) > 1 {
		return matches[1]
	}
	return ""
}
