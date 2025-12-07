/*
   Copyright 2025 The DIRPX Authors

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

package git_test

import (
	"encoding/json"
	"strings"
	"testing"

	"dirpx.dev/dxrel/dxcore/model/git"
	"gopkg.in/yaml.v3"
)

// ============================================================================
// FileChangeKind Tests
// ============================================================================

func TestFileChangeKind_String(t *testing.T) {
	tests := []struct {
		name string
		kind git.FileChangeKind
		want string
	}{
		{"unknown", git.FileChangeUnknown, "unknown"},
		{"added", git.FileChangeAdded, "added"},
		{"modified", git.FileChangeModified, "modified"},
		{"deleted", git.FileChangeDeleted, "deleted"},
		{"renamed", git.FileChangeRenamed, "renamed"},
		{"copied", git.FileChangeCopied, "copied"},
		{"type-changed", git.FileChangeType, "type-changed"},
		{"invalid_value", git.FileChangeKind(99), "FileChangeKind(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.kind.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFileChangeKind_Redacted(t *testing.T) {
	tests := []struct {
		name string
		kind git.FileChangeKind
		want string
	}{
		{"unknown", git.FileChangeUnknown, "unknown"},
		{"added", git.FileChangeAdded, "added"},
		{"modified", git.FileChangeModified, "modified"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.kind.Redacted()
			if got != tt.want {
				t.Errorf("Redacted() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFileChangeKind_TypeName(t *testing.T) {
	kind := git.FileChangeUnknown
	if got := kind.TypeName(); got != "FileChangeKind" {
		t.Errorf("TypeName() = %q, want %q", got, "FileChangeKind")
	}
}

func TestFileChangeKind_IsZero(t *testing.T) {
	tests := []struct {
		name string
		kind git.FileChangeKind
		want bool
	}{
		{"unknown_is_zero", git.FileChangeUnknown, true},
		{"added_not_zero", git.FileChangeAdded, false},
		{"modified_not_zero", git.FileChangeModified, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.kind.IsZero()
			if got != tt.want {
				t.Errorf("IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileChangeKind_Equal(t *testing.T) {
	tests := []struct {
		name  string
		k1    git.FileChangeKind
		k2    git.FileChangeKind
		want  bool
	}{
		{"both_unknown", git.FileChangeUnknown, git.FileChangeUnknown, true},
		{"same_added", git.FileChangeAdded, git.FileChangeAdded, true},
		{"different_kinds", git.FileChangeAdded, git.FileChangeModified, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.k1.Equal(tt.k2)
			if got != tt.want {
				t.Errorf("Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileChangeKind_Validate(t *testing.T) {
	tests := []struct {
		name    string
		kind    git.FileChangeKind
		wantErr bool
	}{
		{"valid_unknown", git.FileChangeUnknown, false},
		{"valid_added", git.FileChangeAdded, false},
		{"valid_modified", git.FileChangeModified, false},
		{"valid_deleted", git.FileChangeDeleted, false},
		{"valid_renamed", git.FileChangeRenamed, false},
		{"valid_copied", git.FileChangeCopied, false},
		{"valid_type", git.FileChangeType, false},
		{"invalid_value", git.FileChangeKind(99), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.kind.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseFileChangeKind(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    git.FileChangeKind
		wantErr bool
	}{
		{"unknown", "unknown", git.FileChangeUnknown, false},
		{"added", "added", git.FileChangeAdded, false},
		{"modified", "modified", git.FileChangeModified, false},
		{"deleted", "deleted", git.FileChangeDeleted, false},
		{"renamed", "renamed", git.FileChangeRenamed, false},
		{"copied", "copied", git.FileChangeCopied, false},
		{"type-changed", "type-changed", git.FileChangeType, false},
		{"type_changed_alt", "type_changed", git.FileChangeType, false},
		{"typechanged_alt", "typechanged", git.FileChangeType, false},
		{"uppercase", "ADDED", git.FileChangeAdded, false},
		{"whitespace", "  modified  ", git.FileChangeModified, false},
		{"invalid", "invalid", git.FileChangeUnknown, true},
		{"empty", "", git.FileChangeUnknown, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := git.ParseFileChangeKind(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFileChangeKind() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.want) {
				t.Errorf("ParseFileChangeKind() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileChangeKind_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		kind    git.FileChangeKind
		want    string
		wantErr bool
	}{
		{"unknown", git.FileChangeUnknown, `"unknown"`, false},
		{"added", git.FileChangeAdded, `"added"`, false},
		{"modified", git.FileChangeModified, `"modified"`, false},
		{"invalid", git.FileChangeKind(99), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.kind)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(got) != tt.want {
				t.Errorf("MarshalJSON() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestFileChangeKind_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    git.FileChangeKind
		wantErr bool
	}{
		{"unknown", `"unknown"`, git.FileChangeUnknown, false},
		{"added", `"added"`, git.FileChangeAdded, false},
		{"modified", `"modified"`, git.FileChangeModified, false},
		{"invalid", `"invalid"`, git.FileChangeUnknown, true},
		{"bad_json", `{invalid}`, git.FileChangeUnknown, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got git.FileChangeKind
			err := json.Unmarshal([]byte(tt.json), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.want) {
				t.Errorf("UnmarshalJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileChangeKind_MarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		kind    git.FileChangeKind
		wantErr bool
	}{
		{"valid_added", git.FileChangeAdded, false},
		{"invalid", git.FileChangeKind(99), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := yaml.Marshal(tt.kind)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFileChangeKind_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		want    git.FileChangeKind
		wantErr bool
	}{
		{"unknown", "unknown", git.FileChangeUnknown, false},
		{"added", "added", git.FileChangeAdded, false},
		{"invalid", "invalid", git.FileChangeUnknown, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got git.FileChangeKind
			err := yaml.Unmarshal([]byte(tt.yaml), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.want) {
				t.Errorf("UnmarshalYAML() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileChangeKind_JSON_RoundTrip(t *testing.T) {
	kinds := []git.FileChangeKind{
		git.FileChangeUnknown,
		git.FileChangeAdded,
		git.FileChangeModified,
		git.FileChangeDeleted,
		git.FileChangeRenamed,
		git.FileChangeCopied,
		git.FileChangeType,
	}

	for _, kind := range kinds {
		t.Run(kind.String(), func(t *testing.T) {
			data, err := json.Marshal(kind)
			if err != nil {
				t.Fatalf("Marshal() failed: %v", err)
			}

			var decoded git.FileChangeKind
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal() failed: %v", err)
			}

			if !decoded.Equal(kind) {
				t.Errorf("Round-trip failed: got %v, want %v", decoded, kind)
			}
		})
	}
}

func TestFileChangeKind_YAML_RoundTrip(t *testing.T) {
	kinds := []git.FileChangeKind{
		git.FileChangeUnknown,
		git.FileChangeAdded,
		git.FileChangeModified,
		git.FileChangeDeleted,
		git.FileChangeRenamed,
		git.FileChangeCopied,
		git.FileChangeType,
	}

	for _, kind := range kinds {
		t.Run(kind.String(), func(t *testing.T) {
			data, err := yaml.Marshal(kind)
			if err != nil {
				t.Fatalf("Marshal() failed: %v", err)
			}

			var decoded git.FileChangeKind
			if err := yaml.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal() failed: %v", err)
			}

			if !decoded.Equal(kind) {
				t.Errorf("Round-trip failed: got %v, want %v", decoded, kind)
			}
		})
	}
}

// ============================================================================
// FileChange Tests
// ============================================================================

func TestFileChange_String(t *testing.T) {
	tests := []struct {
		name string
		fc   git.FileChange
		want string
	}{
		{
			name: "simple_modified",
			fc: git.FileChange{
				Path: "main.go",
				Kind: git.FileChangeModified,
			},
			want: "FileChange{Path:main.go, Kind:modified}",
		},
		{
			name: "with_old_path",
			fc: git.FileChange{
				Path:    "new.go",
				OldPath: "old.go",
				Kind:    git.FileChangeRenamed,
			},
			want: "FileChange{Path:new.go, OldPath:old.go, Kind:renamed}",
		},
		{
			name: "zero_value",
			fc:   git.FileChange{},
			want: "FileChange{Path:, Kind:unknown}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fc.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFileChange_Redacted(t *testing.T) {
	fc := git.FileChange{
		Path: "src/main.go",
		Kind: git.FileChangeModified,
	}

	redacted := fc.Redacted()
	expected := "FileChange{Path:src/main.go, Kind:modified}"

	if redacted != expected {
		t.Errorf("Redacted() = %q, want %q", redacted, expected)
	}
}

func TestFileChange_TypeName(t *testing.T) {
	fc := git.FileChange{}
	if got := fc.TypeName(); got != "FileChange" {
		t.Errorf("TypeName() = %q, want %q", got, "FileChange")
	}
}

func TestFileChange_IsZero(t *testing.T) {
	tests := []struct {
		name string
		fc   git.FileChange
		want bool
	}{
		{
			name: "zero_value",
			fc:   git.FileChange{},
			want: true,
		},
		{
			name: "with_path",
			fc: git.FileChange{
				Path: "main.go",
			},
			want: false,
		},
		{
			name: "with_kind",
			fc: git.FileChange{
				Kind: git.FileChangeAdded,
			},
			want: false,
		},
		{
			name: "complete",
			fc: git.FileChange{
				Path: "main.go",
				Kind: git.FileChangeModified,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fc.IsZero()
			if got != tt.want {
				t.Errorf("IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileChange_Equal(t *testing.T) {
	tests := []struct {
		name string
		fc1  git.FileChange
		fc2  git.FileChange
		want bool
	}{
		{
			name: "both_zero",
			fc1:  git.FileChange{},
			fc2:  git.FileChange{},
			want: true,
		},
		{
			name: "same_complete",
			fc1: git.FileChange{
				Path: "main.go",
				Kind: git.FileChangeModified,
			},
			fc2: git.FileChange{
				Path: "main.go",
				Kind: git.FileChangeModified,
			},
			want: true,
		},
		{
			name: "different_path",
			fc1: git.FileChange{
				Path: "main.go",
				Kind: git.FileChangeModified,
			},
			fc2: git.FileChange{
				Path: "other.go",
				Kind: git.FileChangeModified,
			},
			want: false,
		},
		{
			name: "different_kind",
			fc1: git.FileChange{
				Path: "main.go",
				Kind: git.FileChangeAdded,
			},
			fc2: git.FileChange{
				Path: "main.go",
				Kind: git.FileChangeModified,
			},
			want: false,
		},
		{
			name: "different_old_path",
			fc1: git.FileChange{
				Path:    "new.go",
				OldPath: "old1.go",
				Kind:    git.FileChangeRenamed,
			},
			fc2: git.FileChange{
				Path:    "new.go",
				OldPath: "old2.go",
				Kind:    git.FileChangeRenamed,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fc1.Equal(tt.fc2)
			if got != tt.want {
				t.Errorf("Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileChange_Validate(t *testing.T) {
	longPath := strings.Repeat("a/", 2049) // > 4096 chars

	tests := []struct {
		name    string
		fc      git.FileChange
		wantErr bool
	}{
		{
			name: "valid_added",
			fc: git.FileChange{
				Path: "src/main.go",
				Kind: git.FileChangeAdded,
			},
			wantErr: false,
		},
		{
			name: "valid_modified",
			fc: git.FileChange{
				Path: "README.md",
				Kind: git.FileChangeModified,
			},
			wantErr: false,
		},
		{
			name: "valid_deleted",
			fc: git.FileChange{
				Path: "old/file.txt",
				Kind: git.FileChangeDeleted,
			},
			wantErr: false,
		},
		{
			name: "valid_renamed",
			fc: git.FileChange{
				Path:    "new/path.go",
				OldPath: "old/path.go",
				Kind:    git.FileChangeRenamed,
			},
			wantErr: false,
		},
		{
			name: "valid_copied",
			fc: git.FileChange{
				Path:    "copy.txt",
				OldPath: "template.txt",
				Kind:    git.FileChangeCopied,
			},
			wantErr: false,
		},
		{
			name: "valid_type_changed",
			fc: git.FileChange{
				Path: "symlink",
				Kind: git.FileChangeType,
			},
			wantErr: false,
		},
		{
			name:    "invalid_zero_value",
			fc:      git.FileChange{},
			wantErr: true,
		},
		{
			name: "invalid_empty_path",
			fc: git.FileChange{
				Path: "",
				Kind: git.FileChangeAdded,
			},
			wantErr: true,
		},
		{
			name: "invalid_path_too_long",
			fc: git.FileChange{
				Path: longPath,
				Kind: git.FileChangeAdded,
			},
			wantErr: true,
		},
		{
			name: "invalid_absolute_path",
			fc: git.FileChange{
				Path: "/absolute/path.go",
				Kind: git.FileChangeAdded,
			},
			wantErr: true,
		},
		{
			name: "invalid_old_path_for_modified",
			fc: git.FileChange{
				Path:    "new.go",
				OldPath: "old.go",
				Kind:    git.FileChangeModified,
			},
			wantErr: true,
		},
		{
			name: "invalid_old_path_absolute",
			fc: git.FileChange{
				Path:    "new.go",
				OldPath: "/old.go",
				Kind:    git.FileChangeRenamed,
			},
			wantErr: true,
		},
		{
			name: "invalid_kind",
			fc: git.FileChange{
				Path: "main.go",
				Kind: git.FileChangeKind(99),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fc.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFileChange_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		fc      git.FileChange
		wantErr bool
	}{
		{
			name: "valid",
			fc: git.FileChange{
				Path: "main.go",
				Kind: git.FileChangeModified,
			},
			wantErr: false,
		},
		{
			name:    "invalid",
			fc:      git.FileChange{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := json.Marshal(tt.fc)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFileChange_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    git.FileChange
		wantErr bool
	}{
		{
			name: "valid_simple",
			json: `{"path":"main.go","kind":"modified"}`,
			want: git.FileChange{
				Path: "main.go",
				Kind: git.FileChangeModified,
			},
			wantErr: false,
		},
		{
			name: "valid_with_old_path",
			json: `{"path":"new.go","old_path":"old.go","kind":"renamed"}`,
			want: git.FileChange{
				Path:    "new.go",
				OldPath: "old.go",
				Kind:    git.FileChangeRenamed,
			},
			wantErr: false,
		},
		{
			name:    "invalid_zero",
			json:    `{"path":"","kind":"unknown"}`,
			wantErr: true,
		},
		{
			name:    "invalid_json",
			json:    `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got git.FileChange
			err := json.Unmarshal([]byte(tt.json), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.want) {
				t.Errorf("UnmarshalJSON() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestFileChange_MarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		fc      git.FileChange
		wantErr bool
	}{
		{
			name: "valid",
			fc: git.FileChange{
				Path: "main.go",
				Kind: git.FileChangeModified,
			},
			wantErr: false,
		},
		{
			name:    "invalid",
			fc:      git.FileChange{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := yaml.Marshal(tt.fc)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFileChange_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		want    git.FileChange
		wantErr bool
	}{
		{
			name: "valid_simple",
			yaml: `path: main.go
kind: modified`,
			want: git.FileChange{
				Path: "main.go",
				Kind: git.FileChangeModified,
			},
			wantErr: false,
		},
		{
			name: "valid_with_old_path",
			yaml: `path: new.go
old_path: old.go
kind: renamed`,
			want: git.FileChange{
				Path:    "new.go",
				OldPath: "old.go",
				Kind:    git.FileChangeRenamed,
			},
			wantErr: false,
		},
		{
			name: "invalid_zero",
			yaml: `path: ""
kind: unknown`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got git.FileChange
			err := yaml.Unmarshal([]byte(tt.yaml), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.want) {
				t.Errorf("UnmarshalYAML() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestFileChange_JSON_RoundTrip(t *testing.T) {
	fc := git.FileChange{
		Path: "src/main.go",
		Kind: git.FileChangeModified,
	}

	data, err := json.Marshal(fc)
	if err != nil {
		t.Fatalf("Marshal() failed: %v", err)
	}

	var decoded git.FileChange
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() failed: %v", err)
	}

	if !decoded.Equal(fc) {
		t.Errorf("Round-trip failed: got %+v, want %+v", decoded, fc)
	}
}

func TestFileChange_YAML_RoundTrip(t *testing.T) {
	fc := git.FileChange{
		Path:    "new/path.go",
		OldPath: "old/path.go",
		Kind:    git.FileChangeRenamed,
	}

	data, err := yaml.Marshal(fc)
	if err != nil {
		t.Fatalf("Marshal() failed: %v", err)
	}

	var decoded git.FileChange
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() failed: %v", err)
	}

	if !decoded.Equal(fc) {
		t.Errorf("Round-trip failed: got %+v, want %+v", decoded, fc)
	}
}

func TestNewFileChange(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		kind      git.FileChangeKind
		wantErr   bool
		wantEqual git.FileChange
	}{
		{
			name:    "valid_added",
			path:    "src/main.go",
			kind:    git.FileChangeAdded,
			wantErr: false,
			wantEqual: git.FileChange{
				Path: "src/main.go",
				Kind: git.FileChangeAdded,
			},
		},
		{
			name:    "invalid_empty_path",
			path:    "",
			kind:    git.FileChangeAdded,
			wantErr: true,
		},
		{
			name:    "invalid_absolute_path",
			path:    "/absolute/path.go",
			kind:    git.FileChangeAdded,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := git.NewFileChange(tt.path, tt.kind)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFileChange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.wantEqual) {
				t.Errorf("NewFileChange() = %+v, want %+v", got, tt.wantEqual)
			}
		})
	}
}

func TestFileChange_CommonScenarios(t *testing.T) {
	scenarios := []struct {
		name  string
		fc    git.FileChange
		valid bool
	}{
		{
			name: "added_new_file",
			fc: git.FileChange{
				Path: "src/feature.go",
				Kind: git.FileChangeAdded,
			},
			valid: true,
		},
		{
			name: "modified_existing",
			fc: git.FileChange{
				Path: "README.md",
				Kind: git.FileChangeModified,
			},
			valid: true,
		},
		{
			name: "deleted_old_file",
			fc: git.FileChange{
				Path: "deprecated/old.go",
				Kind: git.FileChangeDeleted,
			},
			valid: true,
		},
		{
			name: "renamed_refactor",
			fc: git.FileChange{
				Path:    "internal/config/settings.go",
				OldPath: "pkg/config/settings.go",
				Kind:    git.FileChangeRenamed,
			},
			valid: true,
		},
		{
			name: "copied_template",
			fc: git.FileChange{
				Path:    "service2/handler.go",
				OldPath: "service1/handler.go",
				Kind:    git.FileChangeCopied,
			},
			valid: true,
		},
		{
			name: "deep_nested_path",
			fc: git.FileChange{
				Path: "a/b/c/d/e/f/g/h/i/j/k/file.go",
				Kind: git.FileChangeAdded,
			},
			valid: true,
		},
	}

	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fc.Validate()
			if tt.valid && err != nil {
				t.Errorf("Expected valid FileChange, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("Expected invalid FileChange, but validation passed")
			}

			// Test round-trip
			if tt.valid {
				data, err := json.Marshal(tt.fc)
				if err != nil {
					t.Fatalf("JSON marshal failed: %v", err)
				}

				var decoded git.FileChange
				if err := json.Unmarshal(data, &decoded); err != nil {
					t.Fatalf("JSON unmarshal failed: %v", err)
				}

				if !decoded.Equal(tt.fc) {
					t.Errorf("JSON round-trip failed: got %+v, want %+v", decoded, tt.fc)
				}
			}
		})
	}
}
