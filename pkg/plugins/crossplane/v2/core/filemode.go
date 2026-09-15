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
	"io/fs"
	"strings"
)

const (
	// ScriptMode is the permission scaffolded shell scripts carry: they are
	// exec'd directly (uptest runs test/setup.sh), so machinery's 0644 is wrong.
	ScriptMode fs.FileMode = 0o755 // #nosec G302 -- executable script by design
	fileMode   fs.FileMode = 0o644
)

// FileMode returns the permission a scaffolded file is written with. Both the
// update write path and the post-scaffold chmod step use it, so the rule has
// one home.
func FileMode(path string) fs.FileMode {
	if strings.HasSuffix(path, ".sh") {
		return ScriptMode
	}
	return fileMode
}
