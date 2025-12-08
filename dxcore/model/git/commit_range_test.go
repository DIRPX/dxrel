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

func TestNewCommitRange(t *testing.T) {
	validFrom := git.Ref{
		Name: "v1.0.0",
		Kind: git.RefKindTag,
		Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
	}
	validTo := git.Ref{
		Name: "v2.0.0",
		Kind: git.RefKindTag,
		Hash: "1234567890abcdef1234567890abcdef12345678",
	}

	tests := []struct {
		name    string
		from    git.Ref
		to      git.Ref
		wantErr bool
	}{
		{
			name:    "valid_range_two_tags",
			from:    validFrom,
			to:      validTo,
			wantErr: false,
		},
		{
			name:    "valid_from_beginning",
			from:    git.Ref{}, // Zero = from beginning
			to:      validTo,
			wantErr: false,
		},
		{
			name:    "invalid_zero_to",
			from:    validFrom,
			to:      git.Ref{}, // Zero To is invalid
			wantErr: true,
		},
		{
			name:    "invalid_both_zero",
			from:    git.Ref{},
			to:      git.Ref{},
			wantErr: true,
		},
		{
			name: "invalid_from_bad_hash",
			from: git.Ref{
				Name: "v1.0.0",
				Kind: git.RefKindTag,
				Hash: "INVALID",
			},
			to:      validTo,
			wantErr: true,
		},
		{
			name: "invalid_to_bad_hash",
			from: validFrom,
			to: git.Ref{
				Name: "v2.0.0",
				Kind: git.RefKindTag,
				Hash: "INVALID",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := git.NewCommitRange(tt.from, tt.to)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewCommitRange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if !got.From.Equal(tt.from) {
					t.Errorf("NewCommitRange().From = %v, want %v", got.From, tt.from)
				}
				if !got.To.Equal(tt.to) {
					t.Errorf("NewCommitRange().To = %v, want %v", got.To, tt.to)
				}
			}
		})
	}
}

func TestCommitRange_String(t *testing.T) {
	tests := []struct {
		name     string
		cr       git.CommitRange
		contains []string
	}{
		{
			name: "named_range",
			cr: git.CommitRange{
				From: git.Ref{
					Name: "v1.0.0",
					Kind: git.RefKindTag,
					Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
				},
				To: git.Ref{
					Name: "v2.0.0",
					Kind: git.RefKindTag,
					Hash: "1234567890abcdef1234567890abcdef12345678",
				},
			},
			contains: []string{"v1.0.0", "v2.0.0", "a1b2c3d", "1234567", ".."},
		},
		{
			name: "from_beginning",
			cr: git.CommitRange{
				From: git.Ref{}, // Zero
				To: git.Ref{
					Name: "HEAD",
					Kind: git.RefKindBranch,
					Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
				},
			},
			contains: []string{"(zero)", "HEAD", "a1b2c3d", ".."},
		},
		{
			name: "hash_only_range",
			cr: git.CommitRange{
				From: git.Ref{
					Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
					Kind: git.RefKindHash,
				},
				To: git.Ref{
					Hash: "1234567890abcdef1234567890abcdef12345678",
					Kind: git.RefKindHash,
				},
			},
			contains: []string{"a1b2c3d", "1234567", ".."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cr.String()
			for _, substr := range tt.contains {
				if !strings.Contains(got, substr) {
					t.Errorf("CommitRange.String() = %q, should contain %q", got, substr)
				}
			}
		})
	}
}

func TestCommitRange_TypeName(t *testing.T) {
	var cr git.CommitRange
	if got := cr.TypeName(); got != "CommitRange" {
		t.Errorf("CommitRange.TypeName() = %v, want CommitRange", got)
	}
}

func TestCommitRange_IsZero(t *testing.T) {
	tests := []struct {
		name string
		cr   git.CommitRange
		want bool
	}{
		{
			name: "zero_value",
			cr:   git.CommitRange{},
			want: true,
		},
		{
			name: "non_zero_both",
			cr: git.CommitRange{
				From: git.Ref{
					Name: "v1.0.0",
					Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
				},
				To: git.Ref{
					Name: "v2.0.0",
					Hash: "1234567890abcdef1234567890abcdef12345678",
				},
			},
			want: false,
		},
		{
			name: "zero_from_only",
			cr: git.CommitRange{
				From: git.Ref{},
				To: git.Ref{
					Name: "HEAD",
					Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cr.IsZero(); got != tt.want {
				t.Errorf("CommitRange.IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommitRange_Equal(t *testing.T) {
	ref1 := git.Ref{
		Name: "v1.0.0",
		Kind: git.RefKindTag,
		Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
	}
	ref2 := git.Ref{
		Name: "v2.0.0",
		Kind: git.RefKindTag,
		Hash: "1234567890abcdef1234567890abcdef12345678",
	}
	ref3 := git.Ref{
		Name: "v3.0.0",
		Kind: git.RefKindTag,
		Hash: "fedcba0987654321fedcba0987654321fedcba09",
	}

	tests := []struct {
		name  string
		cr1   git.CommitRange
		cr2   any
		want  bool
	}{
		{
			name: "equal_ranges",
			cr1: git.CommitRange{From: ref1, To: ref2},
			cr2:  git.CommitRange{From: ref1, To: ref2},
			want: true,
		},
		{
			name: "different_to",
			cr1:  git.CommitRange{From: ref1, To: ref2},
			cr2:  git.CommitRange{From: ref1, To: ref3},
			want: false,
		},
		{
			name: "different_from",
			cr1:  git.CommitRange{From: ref1, To: ref2},
			cr2:  git.CommitRange{From: ref3, To: ref2},
			want: false,
		},
		{
			name: "pointer_equal",
			cr1:  git.CommitRange{From: ref1, To: ref2},
			cr2:  &git.CommitRange{From: ref1, To: ref2},
			want: true,
		},
		{
			name: "nil_pointer",
			cr1:  git.CommitRange{From: ref1, To: ref2},
			cr2:  (*git.CommitRange)(nil),
			want: false,
		},
		{
			name: "different_type",
			cr1:  git.CommitRange{From: ref1, To: ref2},
			cr2:  "not a commit range",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cr1.Equal(tt.cr2); got != tt.want {
				t.Errorf("CommitRange.Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommitRange_Validate(t *testing.T) {
	validFrom := git.Ref{
		Name: "v1.0.0",
		Kind: git.RefKindTag,
		Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
	}
	validTo := git.Ref{
		Name: "v2.0.0",
		Kind: git.RefKindTag,
		Hash: "1234567890abcdef1234567890abcdef12345678",
	}

	tests := []struct {
		name    string
		cr      git.CommitRange
		wantErr bool
	}{
		{
			name: "valid_full_range",
			cr: git.CommitRange{
				From: validFrom,
				To:   validTo,
			},
			wantErr: false,
		},
		{
			name: "valid_from_beginning",
			cr: git.CommitRange{
				From: git.Ref{}, // Zero = from beginning
				To:   validTo,
			},
			wantErr: false,
		},
		{
			name: "invalid_zero_value",
			cr: git.CommitRange{
				From: git.Ref{},
				To:   git.Ref{},
			},
			wantErr: true,
		},
		{
			name: "invalid_zero_to",
			cr: git.CommitRange{
				From: validFrom,
				To:   git.Ref{},
			},
			wantErr: true,
		},
		{
			name: "invalid_from_bad_hash",
			cr: git.CommitRange{
				From: git.Ref{
					Name: "v1.0.0",
					Kind: git.RefKindTag,
					Hash: "INVALID",
				},
				To: validTo,
			},
			wantErr: true,
		},
		{
			name: "invalid_to_bad_hash",
			cr: git.CommitRange{
				From: validFrom,
				To: git.Ref{
					Name: "v2.0.0",
					Kind: git.RefKindTag,
					Hash: "INVALID",
				},
			},
			wantErr: true,
		},
		{
			name: "valid_from_hash_only",
			cr: git.CommitRange{
				From: git.Ref{
					Name: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
					Kind: git.RefKindHash,
					Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
				},
				To: validTo,
			},
			wantErr: false,
		},
		{
			name: "valid_to_hash_only",
			cr: git.CommitRange{
				From: validFrom,
				To: git.Ref{
					Name: "1234567890abcdef1234567890abcdef12345678",
					Kind: git.RefKindHash,
					Hash: "1234567890abcdef1234567890abcdef12345678",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cr.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("CommitRange.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCommitRange_JSON_RoundTrip(t *testing.T) {
	cr := git.CommitRange{
		From: git.Ref{
			Name: "v1.0.0",
			Kind: git.RefKindTag,
			Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
		},
		To: git.Ref{
			Name: "v2.0.0",
			Kind: git.RefKindTag,
			Hash: "1234567890abcdef1234567890abcdef12345678",
		},
	}

	data, err := json.Marshal(cr)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded git.CommitRange
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if !decoded.Equal(cr) {
		t.Errorf("Round trip failed:\ngot  %+v\nwant %+v", decoded, cr)
	}
}

func TestCommitRange_JSON_MarshalInvalid(t *testing.T) {
	// Zero CommitRange is invalid
	cr := git.CommitRange{}

	_, err := json.Marshal(cr)
	if err == nil {
		t.Error("Expected error marshaling invalid CommitRange, got nil")
	}
}

func TestCommitRange_JSON_UnmarshalInvalid(t *testing.T) {
	// JSON with zero To (invalid)
	jsonData := `{"from":{"name":"v1.0.0","kind":"tag","hash":"a1b2c3d4e5f67890abcdef1234567890abcdef12"},"to":{"name":"","kind":"unknown","hash":""}}`

	var cr git.CommitRange
	err := json.Unmarshal([]byte(jsonData), &cr)
	if err == nil {
		t.Error("Expected error unmarshaling invalid CommitRange, got nil")
	}
}

func TestCommitRange_YAML_RoundTrip(t *testing.T) {
	cr := git.CommitRange{
		From: git.Ref{
			Name: "v1.0.0",
			Kind: git.RefKindTag,
			Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
		},
		To: git.Ref{
			Name: "v2.0.0",
			Kind: git.RefKindTag,
			Hash: "1234567890abcdef1234567890abcdef12345678",
		},
	}

	data, err := yaml.Marshal(cr)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded git.CommitRange
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if !decoded.Equal(cr) {
		t.Errorf("Round trip failed:\ngot  %+v\nwant %+v", decoded, cr)
	}
}

func TestCommitRange_YAML_MarshalInvalid(t *testing.T) {
	// Zero CommitRange is invalid
	cr := git.CommitRange{}

	_, err := yaml.Marshal(cr)
	if err == nil {
		t.Error("Expected error marshaling invalid CommitRange, got nil")
	}
}

func TestCommitRange_YAML_UnmarshalInvalid(t *testing.T) {
	// YAML with zero To (invalid)
	yamlData := `
from:
  name: v1.0.0
  kind: tag
  hash: a1b2c3d4e5f67890abcdef1234567890abcdef12
to:
  name: ""
  kind: unknown
  hash: ""
`

	var cr git.CommitRange
	err := yaml.Unmarshal([]byte(yamlData), &cr)
	if err == nil {
		t.Error("Expected error unmarshaling invalid CommitRange, got nil")
	}
}

func TestCommitRange_CommonScenarios(t *testing.T) {
	scenarios := []struct {
		name    string
		cr      git.CommitRange
		valid   bool
	}{
		{
			name: "release_to_release",
			cr: git.CommitRange{
				From: git.Ref{
					Name: "v1.0.0",
					Kind: git.RefKindTag,
					Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
				},
				To: git.Ref{
					Name: "v2.0.0",
					Kind: git.RefKindTag,
					Hash: "1234567890abcdef1234567890abcdef12345678",
				},
			},
			valid: true,
		},
		{
			name: "initial_release",
			cr: git.CommitRange{
				From: git.Ref{}, // From beginning
				To: git.Ref{
					Name: "v1.0.0",
					Kind: git.RefKindTag,
					Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
				},
			},
			valid: true,
		},
		{
			name: "branch_to_head",
			cr: git.CommitRange{
				From: git.Ref{
					Name: "main",
					Kind: git.RefKindBranch,
					Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
				},
				To: git.Ref{
					Name: "HEAD",
					Kind: git.RefKindHead,
					Hash: "1234567890abcdef1234567890abcdef12345678",
				},
			},
			valid: true,
		},
		{
			name: "hash_to_hash",
			cr: git.CommitRange{
				From: git.Ref{
					Name: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
					Kind: git.RefKindHash,
					Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
				},
				To: git.Ref{
					Name: "1234567890abcdef1234567890abcdef12345678",
					Kind: git.RefKindHash,
					Hash: "1234567890abcdef1234567890abcdef12345678",
				},
			},
			valid: true,
		},
		{
			name: "tag_to_branch",
			cr: git.CommitRange{
				From: git.Ref{
					Name: "v1.0.0",
					Kind: git.RefKindTag,
					Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
				},
				To: git.Ref{
					Name: "develop",
					Kind: git.RefKindBranch,
					Hash: "1234567890abcdef1234567890abcdef12345678",
				},
			},
			valid: true,
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			err := sc.cr.Validate()
			if sc.valid && err != nil {
				t.Errorf("Expected valid CommitRange, got error: %v", err)
			}
			if !sc.valid && err == nil {
				t.Errorf("Expected invalid CommitRange, got nil error")
			}
		})
	}
}
