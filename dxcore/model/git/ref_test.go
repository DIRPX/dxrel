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

func TestRef_String(t *testing.T) {
	tests := []struct {
		name string
		ref  git.Ref
		want string
	}{
		{
			name: "complete_branch_ref",
			ref: git.Ref{
				Name: "refs/heads/main",
				Kind: git.RefKindBranch,
				Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
			},
			want: "Ref{Name:refs/heads/main, Kind:branch, Hash:a1b2c3d4e5f67890abcdef1234567890abcdef12}",
		},
		{
			name: "complete_tag_ref",
			ref: git.Ref{
				Name: "v1.0.0",
				Kind: git.RefKindTag,
				Hash: "1234567890abcdef1234567890abcdef12345678",
			},
			want: "Ref{Name:v1.0.0, Kind:tag, Hash:1234567890abcdef1234567890abcdef12345678}",
		},
		{
			name: "zero_ref",
			ref:  git.Ref{},
			want: "Ref{Name:, Kind:unknown, Hash:}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ref.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRef_Redacted(t *testing.T) {
	tests := []struct {
		name string
		ref  git.Ref
		want string
	}{
		{
			name: "complete_ref_abbreviates_hash",
			ref: git.Ref{
				Name: "main",
				Kind: git.RefKindBranch,
				Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
			},
			want: "Ref{Name:main, Kind:branch, Hash:a1b2c3d}",
		},
		{
			name: "short_hash_not_abbreviated",
			ref: git.Ref{
				Name: "dev",
				Kind: git.RefKindBranch,
				Hash: "abc123",
			},
			want: "Ref{Name:dev, Kind:branch, Hash:abc123}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ref.Redacted()
			if got != tt.want {
				t.Errorf("Redacted() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRef_TypeName(t *testing.T) {
	ref := git.Ref{}
	got := ref.TypeName()
	want := "Ref"
	if got != want {
		t.Errorf("TypeName() = %q, want %q", got, want)
	}
}

func TestRef_IsZero(t *testing.T) {
	tests := []struct {
		name string
		ref  git.Ref
		want bool
	}{
		{
			name: "zero_ref",
			ref:  git.Ref{},
			want: true,
		},
		{
			name: "ref_with_name",
			ref: git.Ref{
				Name: "main",
			},
			want: false,
		},
		{
			name: "ref_with_kind",
			ref: git.Ref{
				Kind: git.RefKindBranch,
			},
			want: false,
		},
		{
			name: "ref_with_hash",
			ref: git.Ref{
				Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
			},
			want: false,
		},
		{
			name: "complete_ref",
			ref: git.Ref{
				Name: "main",
				Kind: git.RefKindBranch,
				Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ref.IsZero()
			if got != tt.want {
				t.Errorf("IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRef_Equal(t *testing.T) {
	tests := []struct {
		name string
		r1   git.Ref
		r2   git.Ref
		want bool
	}{
		{
			name: "both_zero",
			r1:   git.Ref{},
			r2:   git.Ref{},
			want: true,
		},
		{
			name: "same_complete_refs",
			r1: git.Ref{
				Name: "main",
				Kind: git.RefKindBranch,
				Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
			},
			r2: git.Ref{
				Name: "main",
				Kind: git.RefKindBranch,
				Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
			},
			want: true,
		},
		{
			name: "different_names",
			r1: git.Ref{
				Name: "main",
				Kind: git.RefKindBranch,
				Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
			},
			r2: git.Ref{
				Name: "develop",
				Kind: git.RefKindBranch,
				Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
			},
			want: false,
		},
		{
			name: "different_kinds",
			r1: git.Ref{
				Name: "v1.0.0",
				Kind: git.RefKindTag,
				Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
			},
			r2: git.Ref{
				Name: "v1.0.0",
				Kind: git.RefKindBranch,
				Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
			},
			want: false,
		},
		{
			name: "different_hashes",
			r1: git.Ref{
				Name: "main",
				Kind: git.RefKindBranch,
				Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
			},
			r2: git.Ref{
				Name: "main",
				Kind: git.RefKindBranch,
				Hash: "1234567890abcdef1234567890abcdef12345678",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.r1.Equal(tt.r2)
			if got != tt.want {
				t.Errorf("Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRef_Validate(t *testing.T) {
	tests := []struct {
		name    string
		ref     git.Ref
		wantErr bool
	}{
		{
			name: "valid_complete_branch",
			ref: git.Ref{
				Name: "refs/heads/main",
				Kind: git.RefKindBranch,
				Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
			},
			wantErr: false,
		},
		{
			name: "valid_complete_tag",
			ref: git.Ref{
				Name: "refs/tags/v1.0.0",
				Kind: git.RefKindTag,
				Hash: "1234567890abcdef1234567890abcdef12345678",
			},
			wantErr: false,
		},
		{
			name: "valid_with_name_and_hash_only",
			ref: git.Ref{
				Name: "main",
				Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
			},
			wantErr: false,
		},
		{
			name:    "invalid_zero_ref",
			ref:     git.Ref{},
			wantErr: true,
		},
		{
			name: "valid_name_only",
			ref: git.Ref{
				Name: "main",
			},
			wantErr: false, // Name only is valid (hash can be resolved later)
		},
		{
			name: "invalid_hash_format",
			ref: git.Ref{
				Name: "main",
				Kind: git.RefKindBranch,
				Hash: "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid_kind",
			ref: git.Ref{
				Name: "main",
				Kind: git.RefKind(99),
				Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
			},
			wantErr: true,
		},
		{
			name: "invalid_kind_mismatch_branch_as_tag",
			ref: git.Ref{
				Name: "refs/heads/main",
				Kind: git.RefKindTag,
				Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
			},
			wantErr: true, // refs/heads/* must be RefKindBranch
		},
		{
			name: "invalid_kind_mismatch_tag_as_branch",
			ref: git.Ref{
				Name: "refs/tags/v1.0.0",
				Kind: git.RefKindBranch,
				Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
			},
			wantErr: true, // refs/tags/* must be RefKindTag
		},
		{
			name: "invalid_kind_mismatch_remote_as_branch",
			ref: git.Ref{
				Name: "refs/remotes/origin/main",
				Kind: git.RefKindBranch,
				Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
			},
			wantErr: true, // refs/remotes/* must be RefKindRemoteBranch
		},
		{
			name: "invalid_kind_mismatch_HEAD_as_branch",
			ref: git.Ref{
				Name: "HEAD",
				Kind: git.RefKindBranch,
				Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
			},
			wantErr: true, // HEAD must be RefKindHead
		},
		{
			name: "invalid_kind_mismatch_hash_as_branch",
			ref: git.Ref{
				Name: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
				Kind: git.RefKindBranch,
				Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
			},
			wantErr: true, // 40 hex chars must be RefKindHash
		},
		{
			name: "valid_with_unknown_kind",
			ref: git.Ref{
				Name: "refs/heads/main",
				Kind: git.RefKindUnknown,
				Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
			},
			wantErr: false, // RefKindUnknown is allowed (skips consistency check)
		},
		{
			name: "valid_ambiguous_name_with_tag_kind",
			ref: git.Ref{
				Name: "v1.0.0", // Ambiguous - could be tag or short ref
				Kind: git.RefKindTag,
				Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
			},
			wantErr: false, // Ambiguous names don't trigger mismatch
		},
		{
			name: "valid_remote_branch",
			ref: git.Ref{
				Name: "refs/remotes/origin/develop",
				Kind: git.RefKindRemoteBranch,
				Hash: "1234567890abcdef1234567890abcdef12345678",
			},
			wantErr: false,
		},
		{
			name: "valid_HEAD",
			ref: git.Ref{
				Name: "HEAD",
				Kind: git.RefKindHead,
				Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
			},
			wantErr: false,
		},
		{
			name: "valid_hash_ref",
			ref: git.Ref{
				Name: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
				Kind: git.RefKindHash,
				Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ref.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRef_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		ref     git.Ref
		wantErr bool
	}{
		{
			name: "valid_ref",
			ref: git.Ref{
				Name: "main",
				Kind: git.RefKindBranch,
				Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
			},
			wantErr: false,
		},
		{
			name:    "invalid_ref",
			ref:     git.Ref{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("MarshalJSON() returned nil for valid ref")
			}
		})
	}
}

func TestRef_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    git.Ref
		wantErr bool
	}{
		{
			name: "valid_json",
			json: `{"name":"refs/heads/main","kind":"branch","hash":"a1b2c3d4e5f67890abcdef1234567890abcdef12"}`,
			want: git.Ref{
				Name: "refs/heads/main",
				Kind: git.RefKindBranch,
				Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
			},
			wantErr: false,
		},
		{
			name:    "invalid_json_zero_ref",
			json:    `{"name":"","kind":"unknown","hash":""}`,
			wantErr: true,
		},
		{
			name:    "invalid_json_format",
			json:    `not-json`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got git.Ref
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

func TestRef_MarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		ref     git.Ref
		wantErr bool
	}{
		{
			name: "valid_ref",
			ref: git.Ref{
				Name: "main",
				Kind: git.RefKindBranch,
				Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
			},
			wantErr: false,
		},
		{
			name:    "invalid_ref",
			ref:     git.Ref{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := yaml.Marshal(tt.ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("MarshalYAML() returned nil for valid ref")
			}
		})
	}
}

func TestRef_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		want    git.Ref
		wantErr bool
	}{
		{
			name: "valid_yaml",
			yaml: `name: refs/heads/main
kind: branch
hash: a1b2c3d4e5f67890abcdef1234567890abcdef12`,
			want: git.Ref{
				Name: "refs/heads/main",
				Kind: git.RefKindBranch,
				Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
			},
			wantErr: false,
		},
		{
			name: "invalid_yaml_zero_ref",
			yaml: `name: ""
kind: unknown
hash: ""`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got git.Ref
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

func TestRef_JSON_RoundTrip(t *testing.T) {
	original := git.Ref{
		Name: "refs/heads/main",
		Kind: git.RefKindBranch,
		Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() failed: %v", err)
	}

	var decoded git.Ref
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() failed: %v", err)
	}

	if !decoded.Equal(original) {
		t.Errorf("Round-trip failed: got %+v, want %+v", decoded, original)
	}
}

func TestRef_YAML_RoundTrip(t *testing.T) {
	original := git.Ref{
		Name: "refs/tags/v1.0.0",
		Kind: git.RefKindTag,
		Hash: "1234567890abcdef1234567890abcdef12345678",
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() failed: %v", err)
	}

	var decoded git.Ref
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() failed: %v", err)
	}

	if !decoded.Equal(original) {
		t.Errorf("Round-trip failed: got %+v, want %+v", decoded, original)
	}
}

func TestNewRef(t *testing.T) {
	tests := []struct {
		name    string
		refName git.RefName
		kind    git.RefKind
		hash    git.Hash
		wantErr bool
	}{
		{
			name:    "valid_ref",
			refName: "refs/heads/main",
			kind:    git.RefKindBranch,
			hash:    "a1b2c3d4e5f67890abcdef1234567890abcdef12",
			wantErr: false,
		},
		{
			name:    "invalid_empty",
			refName: "",
			kind:    git.RefKindUnknown,
			hash:    "",
			wantErr: true,
		},
		{
			name:    "invalid_hash",
			refName: "main",
			kind:    git.RefKindBranch,
			hash:    "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := git.NewRef(tt.refName, tt.kind, tt.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRef() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.IsZero() {
				t.Error("NewRef() returned zero Ref for valid input")
			}
		})
	}
}

func TestRef_AllRefKinds(t *testing.T) {
	// Test that Ref works with all RefKind values
	kinds := []git.RefKind{
		git.RefKindBranch,
		git.RefKindRemoteBranch,
		git.RefKindTag,
		git.RefKindHead,
		git.RefKindHash,
	}

	for _, kind := range kinds {
		t.Run(kind.String(), func(t *testing.T) {
			ref := git.Ref{
				Name: "test",
				Kind: kind,
				Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
			}

			if err := ref.Validate(); err != nil {
				t.Errorf("Validate() failed for kind %v: %v", kind, err)
			}

			// Test round-trip through JSON
			data, err := json.Marshal(ref)
			if err != nil {
				t.Fatalf("Marshal() failed: %v", err)
			}

			var decoded git.Ref
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal() failed: %v", err)
			}

			if !decoded.Equal(ref) {
				t.Errorf("Round-trip failed for kind %v: got %+v, want %+v", kind, decoded, ref)
			}
		})
	}
}
