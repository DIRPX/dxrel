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
	"testing"

	"dirpx.dev/dxrel/dxcore/model/git"
	"gopkg.in/yaml.v3"
)

func TestRefKind_String(t *testing.T) {
	tests := []struct {
		name string
		kind git.RefKind
		want string
	}{
		{
			name: "unknown",
			kind: git.RefKindUnknown,
			want: "unknown",
		},
		{
			name: "branch",
			kind: git.RefKindBranch,
			want: "branch",
		},
		{
			name: "remote_branch",
			kind: git.RefKindRemoteBranch,
			want: "remote-branch",
		},
		{
			name: "tag",
			kind: git.RefKindTag,
			want: "tag",
		},
		{
			name: "head",
			kind: git.RefKindHead,
			want: "head",
		},
		{
			name: "hash",
			kind: git.RefKindHash,
			want: "hash",
		},
		{
			name: "invalid_value",
			kind: git.RefKind(99),
			want: "RefKind(99)",
		},
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

func TestRefKind_Redacted(t *testing.T) {
	tests := []struct {
		name string
		kind git.RefKind
		want string
	}{
		{
			name: "unknown",
			kind: git.RefKindUnknown,
			want: "unknown",
		},
		{
			name: "branch",
			kind: git.RefKindBranch,
			want: "branch",
		},
		{
			name: "tag",
			kind: git.RefKindTag,
			want: "tag",
		},
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

func TestRefKind_TypeName(t *testing.T) {
	var rk git.RefKind
	got := rk.TypeName()
	want := "RefKind"
	if got != want {
		t.Errorf("TypeName() = %q, want %q", got, want)
	}
}

func TestRefKind_IsZero(t *testing.T) {
	tests := []struct {
		name string
		kind git.RefKind
		want bool
	}{
		{
			name: "unknown_is_zero",
			kind: git.RefKindUnknown,
			want: true,
		},
		{
			name: "branch_not_zero",
			kind: git.RefKindBranch,
			want: false,
		},
		{
			name: "tag_not_zero",
			kind: git.RefKindTag,
			want: false,
		},
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

func TestRefKind_Equal(t *testing.T) {
	tests := []struct {
		name string
		rk1  git.RefKind
		rk2  git.RefKind
		want bool
	}{
		{
			name: "both_unknown",
			rk1:  git.RefKindUnknown,
			rk2:  git.RefKindUnknown,
			want: true,
		},
		{
			name: "same_branch",
			rk1:  git.RefKindBranch,
			rk2:  git.RefKindBranch,
			want: true,
		},
		{
			name: "different_kinds",
			rk1:  git.RefKindBranch,
			rk2:  git.RefKindTag,
			want: false,
		},
		{
			name: "unknown_vs_branch",
			rk1:  git.RefKindUnknown,
			rk2:  git.RefKindBranch,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.rk1.Equal(tt.rk2)
			if got != tt.want {
				t.Errorf("Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRefKind_Validate(t *testing.T) {
	tests := []struct {
		name    string
		kind    git.RefKind
		wantErr bool
	}{
		{
			name:    "unknown_valid",
			kind:    git.RefKindUnknown,
			wantErr: false,
		},
		{
			name:    "branch_valid",
			kind:    git.RefKindBranch,
			wantErr: false,
		},
		{
			name:    "remote_branch_valid",
			kind:    git.RefKindRemoteBranch,
			wantErr: false,
		},
		{
			name:    "tag_valid",
			kind:    git.RefKindTag,
			wantErr: false,
		},
		{
			name:    "head_valid",
			kind:    git.RefKindHead,
			wantErr: false,
		},
		{
			name:    "hash_valid",
			kind:    git.RefKindHash,
			wantErr: false,
		},
		{
			name:    "invalid_value_6",
			kind:    git.RefKind(6),
			wantErr: true,
		},
		{
			name:    "invalid_value_99",
			kind:    git.RefKind(99),
			wantErr: true,
		},
		{
			name:    "invalid_value_255",
			kind:    git.RefKind(255),
			wantErr: true,
		},
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

func TestRefKind_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		kind    git.RefKind
		want    string
		wantErr bool
	}{
		{
			name:    "unknown",
			kind:    git.RefKindUnknown,
			want:    `"unknown"`,
			wantErr: false,
		},
		{
			name:    "branch",
			kind:    git.RefKindBranch,
			want:    `"branch"`,
			wantErr: false,
		},
		{
			name:    "remote_branch",
			kind:    git.RefKindRemoteBranch,
			want:    `"remote-branch"`,
			wantErr: false,
		},
		{
			name:    "tag",
			kind:    git.RefKindTag,
			want:    `"tag"`,
			wantErr: false,
		},
		{
			name:    "head",
			kind:    git.RefKindHead,
			want:    `"head"`,
			wantErr: false,
		},
		{
			name:    "hash",
			kind:    git.RefKindHash,
			want:    `"hash"`,
			wantErr: false,
		},
		{
			name:    "invalid_value",
			kind:    git.RefKind(99),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.kind.MarshalJSON()
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

func TestRefKind_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    git.RefKind
		wantErr bool
	}{
		{
			name:    "unknown",
			json:    `"unknown"`,
			want:    git.RefKindUnknown,
			wantErr: false,
		},
		{
			name:    "branch",
			json:    `"branch"`,
			want:    git.RefKindBranch,
			wantErr: false,
		},
		{
			name:    "branch_uppercase",
			json:    `"BRANCH"`,
			want:    git.RefKindBranch,
			wantErr: false,
		},
		{
			name:    "remote_branch",
			json:    `"remote-branch"`,
			want:    git.RefKindRemoteBranch,
			wantErr: false,
		},
		{
			name:    "remote_branch_underscore",
			json:    `"remote_branch"`,
			want:    git.RefKindRemoteBranch,
			wantErr: false,
		},
		{
			name:    "tag",
			json:    `"tag"`,
			want:    git.RefKindTag,
			wantErr: false,
		},
		{
			name:    "head",
			json:    `"head"`,
			want:    git.RefKindHead,
			wantErr: false,
		},
		{
			name:    "hash",
			json:    `"hash"`,
			want:    git.RefKindHash,
			wantErr: false,
		},
		{
			name:    "with_whitespace",
			json:    `"  tag  "`,
			want:    git.RefKindTag,
			wantErr: false,
		},
		{
			name:    "invalid_name",
			json:    `"invalid"`,
			want:    git.RefKindUnknown,
			wantErr: true,
		},
		{
			name:    "invalid_json",
			json:    `not-json`,
			want:    git.RefKindUnknown,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got git.RefKind
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

func TestRefKind_MarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		kind    git.RefKind
		want    string
		wantErr bool
	}{
		{
			name:    "unknown",
			kind:    git.RefKindUnknown,
			want:    "unknown\n",
			wantErr: false,
		},
		{
			name:    "branch",
			kind:    git.RefKindBranch,
			want:    "branch\n",
			wantErr: false,
		},
		{
			name:    "remote_branch",
			kind:    git.RefKindRemoteBranch,
			want:    "remote-branch\n",
			wantErr: false,
		},
		{
			name:    "tag",
			kind:    git.RefKindTag,
			want:    "tag\n",
			wantErr: false,
		},
		{
			name:    "head",
			kind:    git.RefKindHead,
			want:    "head\n",
			wantErr: false,
		},
		{
			name:    "hash",
			kind:    git.RefKindHash,
			want:    "hash\n",
			wantErr: false,
		},
		{
			name:    "invalid_value",
			kind:    git.RefKind(99),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := yaml.Marshal(tt.kind)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(got) != tt.want {
				t.Errorf("MarshalYAML() = %q, want %q", string(got), tt.want)
			}
		})
	}
}

func TestRefKind_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		want    git.RefKind
		wantErr bool
	}{
		{
			name:    "unknown",
			yaml:    "unknown",
			want:    git.RefKindUnknown,
			wantErr: false,
		},
		{
			name:    "branch",
			yaml:    "branch",
			want:    git.RefKindBranch,
			wantErr: false,
		},
		{
			name:    "branch_uppercase",
			yaml:    "BRANCH",
			want:    git.RefKindBranch,
			wantErr: false,
		},
		{
			name:    "remote_branch",
			yaml:    "remote-branch",
			want:    git.RefKindRemoteBranch,
			wantErr: false,
		},
		{
			name:    "tag",
			yaml:    "tag",
			want:    git.RefKindTag,
			wantErr: false,
		},
		{
			name:    "head",
			yaml:    "head",
			want:    git.RefKindHead,
			wantErr: false,
		},
		{
			name:    "hash",
			yaml:    "hash",
			want:    git.RefKindHash,
			wantErr: false,
		},
		{
			name:    "with_whitespace",
			yaml:    `"  tag  "`,
			want:    git.RefKindTag,
			wantErr: false,
		},
		{
			name:    "invalid_name",
			yaml:    "invalid",
			want:    git.RefKindUnknown,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got git.RefKind
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

func TestParseRefKind(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    git.RefKind
		wantErr bool
	}{
		{
			name:    "unknown",
			input:   "unknown",
			want:    git.RefKindUnknown,
			wantErr: false,
		},
		{
			name:    "branch",
			input:   "branch",
			want:    git.RefKindBranch,
			wantErr: false,
		},
		{
			name:    "branch_uppercase",
			input:   "BRANCH",
			want:    git.RefKindBranch,
			wantErr: false,
		},
		{
			name:    "branch_mixed_case",
			input:   "BrAnCh",
			want:    git.RefKindBranch,
			wantErr: false,
		},
		{
			name:    "remote_branch_hyphen",
			input:   "remote-branch",
			want:    git.RefKindRemoteBranch,
			wantErr: false,
		},
		{
			name:    "remote_branch_underscore",
			input:   "remote_branch",
			want:    git.RefKindRemoteBranch,
			wantErr: false,
		},
		{
			name:    "remote_branch_no_separator",
			input:   "remotebranch",
			want:    git.RefKindRemoteBranch,
			wantErr: false,
		},
		{
			name:    "tag",
			input:   "tag",
			want:    git.RefKindTag,
			wantErr: false,
		},
		{
			name:    "head",
			input:   "head",
			want:    git.RefKindHead,
			wantErr: false,
		},
		{
			name:    "hash",
			input:   "hash",
			want:    git.RefKindHash,
			wantErr: false,
		},
		{
			name:    "with_leading_whitespace",
			input:   "  branch",
			want:    git.RefKindBranch,
			wantErr: false,
		},
		{
			name:    "with_trailing_whitespace",
			input:   "tag  ",
			want:    git.RefKindTag,
			wantErr: false,
		},
		{
			name:    "with_surrounding_whitespace",
			input:   "  head  ",
			want:    git.RefKindHead,
			wantErr: false,
		},
		{
			name:    "invalid_empty",
			input:   "",
			want:    git.RefKindUnknown,
			wantErr: true,
		},
		{
			name:    "invalid_whitespace_only",
			input:   "   ",
			want:    git.RefKindUnknown,
			wantErr: true,
		},
		{
			name:    "invalid_name",
			input:   "invalid",
			want:    git.RefKindUnknown,
			wantErr: true,
		},
		{
			name:    "invalid_numeric",
			input:   "123",
			want:    git.RefKindUnknown,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := git.ParseRefKind(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRefKind() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !got.Equal(tt.want) {
				t.Errorf("ParseRefKind() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRefKind_JSON_RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		kind git.RefKind
	}{
		{
			name: "unknown",
			kind: git.RefKindUnknown,
		},
		{
			name: "branch",
			kind: git.RefKindBranch,
		},
		{
			name: "remote_branch",
			kind: git.RefKindRemoteBranch,
		},
		{
			name: "tag",
			kind: git.RefKindTag,
		},
		{
			name: "head",
			kind: git.RefKindHead,
		},
		{
			name: "hash",
			kind: git.RefKindHash,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.kind)
			if err != nil {
				t.Fatalf("Marshal() failed: %v", err)
			}

			var decoded git.RefKind
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal() failed: %v", err)
			}

			if !decoded.Equal(tt.kind) {
				t.Errorf("Round-trip failed: got %v, want %v", decoded, tt.kind)
			}
		})
	}
}

func TestRefKind_YAML_RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		kind git.RefKind
	}{
		{
			name: "unknown",
			kind: git.RefKindUnknown,
		},
		{
			name: "branch",
			kind: git.RefKindBranch,
		},
		{
			name: "remote_branch",
			kind: git.RefKindRemoteBranch,
		},
		{
			name: "tag",
			kind: git.RefKindTag,
		},
		{
			name: "head",
			kind: git.RefKindHead,
		},
		{
			name: "hash",
			kind: git.RefKindHash,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := yaml.Marshal(tt.kind)
			if err != nil {
				t.Fatalf("Marshal() failed: %v", err)
			}

			var decoded git.RefKind
			if err := yaml.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal() failed: %v", err)
			}

			if !decoded.Equal(tt.kind) {
				t.Errorf("Round-trip failed: got %v, want %v", decoded, tt.kind)
			}
		})
	}
}

func TestRefKind_AllValues(t *testing.T) {
	// Test that all defined RefKind constants are valid
	kinds := []git.RefKind{
		git.RefKindUnknown,
		git.RefKindBranch,
		git.RefKindRemoteBranch,
		git.RefKindTag,
		git.RefKindHead,
		git.RefKindHash,
	}

	for _, kind := range kinds {
		t.Run(kind.String(), func(t *testing.T) {
			if err := kind.Validate(); err != nil {
				t.Errorf("Validate() failed for %v: %v", kind, err)
			}

			// Test String() returns non-empty
			if kind.String() == "" {
				t.Errorf("String() returned empty for %v", kind)
			}

			// Test Redacted() returns non-empty
			if kind.Redacted() == "" {
				t.Errorf("Redacted() returned empty for %v", kind)
			}

			// Test JSON round-trip
			data, err := json.Marshal(kind)
			if err != nil {
				t.Errorf("MarshalJSON() failed for %v: %v", kind, err)
			}
			var decoded git.RefKind
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Errorf("UnmarshalJSON() failed for %v: %v", kind, err)
			}
			if !decoded.Equal(kind) {
				t.Errorf("JSON round-trip failed for %v: got %v", kind, decoded)
			}

			// Test YAML round-trip
			yamlData, err := yaml.Marshal(kind)
			if err != nil {
				t.Errorf("MarshalYAML() failed for %v: %v", kind, err)
			}
			var yamlDecoded git.RefKind
			if err := yaml.Unmarshal(yamlData, &yamlDecoded); err != nil {
				t.Errorf("UnmarshalYAML() failed for %v: %v", kind, err)
			}
			if !yamlDecoded.Equal(kind) {
				t.Errorf("YAML round-trip failed for %v: got %v", kind, yamlDecoded)
			}
		})
	}
}
