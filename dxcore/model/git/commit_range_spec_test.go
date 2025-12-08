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

func TestNewCommitRangeSpec(t *testing.T) {
	tests := []struct {
		name    string
		from    git.RefName
		to      git.RefName
		wantErr bool
	}{
		{
			name:    "valid_two_tags",
			from:    "v1.0.0",
			to:      "v2.0.0",
			wantErr: false,
		},
		{
			name:    "valid_from_beginning",
			from:    "", // Empty = from beginning
			to:      "HEAD",
			wantErr: false,
		},
		{
			name:    "valid_branch_to_branch",
			from:    "main",
			to:      "develop",
			wantErr: false,
		},
		{
			name:    "invalid_empty_to",
			from:    "v1.0.0",
			to:      "", // Empty To is invalid
			wantErr: true,
		},
		{
			name:    "invalid_both_empty",
			from:    "",
			to:      "",
			wantErr: true,
		},
		{
			name:    "invalid_from_with_newline",
			from:    "v1.0.0\n",
			to:      "v2.0.0",
			wantErr: true,
		},
		{
			name:    "invalid_to_with_newline",
			from:    "v1.0.0",
			to:      "v2.0.0\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := git.NewCommitRangeSpec(tt.from, tt.to)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewCommitRangeSpec() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if !got.From.Equal(tt.from) {
					t.Errorf("NewCommitRangeSpec().From = %v, want %v", got.From, tt.from)
				}
				if !got.To.Equal(tt.to) {
					t.Errorf("NewCommitRangeSpec().To = %v, want %v", got.To, tt.to)
				}
			}
		})
	}
}

func TestCommitRangeSpec_String(t *testing.T) {
	tests := []struct {
		name     string
		spec     git.CommitRangeSpec
		contains []string
	}{
		{
			name: "named_range",
			spec: git.CommitRangeSpec{
				From: "v1.0.0",
				To:   "v2.0.0",
			},
			contains: []string{"v1.0.0", "v2.0.0", ".."},
		},
		{
			name: "from_beginning",
			spec: git.CommitRangeSpec{
				From: "", // Empty
				To:   "HEAD",
			},
			contains: []string{"(empty)", "HEAD", ".."},
		},
		{
			name: "branch_range",
			spec: git.CommitRangeSpec{
				From: "main",
				To:   "develop",
			},
			contains: []string{"main", "develop", ".."},
		},
		{
			name: "zero_value",
			spec: git.CommitRangeSpec{
				From: "",
				To:   "",
			},
			contains: []string{"(empty)", "(empty)", ".."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.spec.String()
			for _, substr := range tt.contains {
				if !strings.Contains(got, substr) {
					t.Errorf("CommitRangeSpec.String() = %q, should contain %q", got, substr)
				}
			}
		})
	}
}

func TestCommitRangeSpec_TypeName(t *testing.T) {
	var spec git.CommitRangeSpec
	if got := spec.TypeName(); got != "CommitRangeSpec" {
		t.Errorf("CommitRangeSpec.TypeName() = %v, want CommitRangeSpec", got)
	}
}

func TestCommitRangeSpec_IsZero(t *testing.T) {
	tests := []struct {
		name string
		spec git.CommitRangeSpec
		want bool
	}{
		{
			name: "zero_value",
			spec: git.CommitRangeSpec{},
			want: true,
		},
		{
			name: "non_zero_both",
			spec: git.CommitRangeSpec{
				From: "v1.0.0",
				To:   "v2.0.0",
			},
			want: false,
		},
		{
			name: "zero_from_only",
			spec: git.CommitRangeSpec{
				From: "",
				To:   "HEAD",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.spec.IsZero(); got != tt.want {
				t.Errorf("CommitRangeSpec.IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommitRangeSpec_Equal(t *testing.T) {
	tests := []struct {
		name  string
		spec1 git.CommitRangeSpec
		spec2 any
		want  bool
	}{
		{
			name:  "equal_specs",
			spec1: git.CommitRangeSpec{From: "v1.0.0", To: "v2.0.0"},
			spec2: git.CommitRangeSpec{From: "v1.0.0", To: "v2.0.0"},
			want:  true,
		},
		{
			name:  "different_to",
			spec1: git.CommitRangeSpec{From: "v1.0.0", To: "v2.0.0"},
			spec2: git.CommitRangeSpec{From: "v1.0.0", To: "HEAD"},
			want:  false,
		},
		{
			name:  "different_from",
			spec1: git.CommitRangeSpec{From: "v1.0.0", To: "v2.0.0"},
			spec2: git.CommitRangeSpec{From: "v0.9.0", To: "v2.0.0"},
			want:  false,
		},
		{
			name:  "pointer_equal",
			spec1: git.CommitRangeSpec{From: "v1.0.0", To: "v2.0.0"},
			spec2: &git.CommitRangeSpec{From: "v1.0.0", To: "v2.0.0"},
			want:  true,
		},
		{
			name:  "nil_pointer",
			spec1: git.CommitRangeSpec{From: "v1.0.0", To: "v2.0.0"},
			spec2: (*git.CommitRangeSpec)(nil),
			want:  false,
		},
		{
			name:  "different_type",
			spec1: git.CommitRangeSpec{From: "v1.0.0", To: "v2.0.0"},
			spec2: "not a spec",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.spec1.Equal(tt.spec2); got != tt.want {
				t.Errorf("CommitRangeSpec.Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommitRangeSpec_Validate(t *testing.T) {
	tests := []struct {
		name    string
		spec    git.CommitRangeSpec
		wantErr bool
	}{
		{
			name: "valid_full_range",
			spec: git.CommitRangeSpec{
				From: "v1.0.0",
				To:   "v2.0.0",
			},
			wantErr: false,
		},
		{
			name: "valid_from_beginning",
			spec: git.CommitRangeSpec{
				From: "", // Empty = from beginning
				To:   "v2.0.0",
			},
			wantErr: false,
		},
		{
			name: "valid_branch_to_head",
			spec: git.CommitRangeSpec{
				From: "main",
				To:   "HEAD",
			},
			wantErr: false,
		},
		{
			name: "invalid_zero_value",
			spec: git.CommitRangeSpec{
				From: "",
				To:   "",
			},
			wantErr: true,
		},
		{
			name: "invalid_empty_to",
			spec: git.CommitRangeSpec{
				From: "v1.0.0",
				To:   "",
			},
			wantErr: true,
		},
		{
			name: "invalid_from_too_long",
			spec: git.CommitRangeSpec{
				From: git.RefName(strings.Repeat("a", 257)),
				To:   "v2.0.0",
			},
			wantErr: true,
		},
		{
			name: "invalid_to_too_long",
			spec: git.CommitRangeSpec{
				From: "v1.0.0",
				To:   git.RefName(strings.Repeat("a", 257)),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("CommitRangeSpec.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCommitRangeSpec_JSON_RoundTrip(t *testing.T) {
	spec := git.CommitRangeSpec{
		From: "v1.0.0",
		To:   "v2.0.0",
	}

	data, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded git.CommitRangeSpec
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if !decoded.Equal(spec) {
		t.Errorf("Round trip failed:\ngot  %+v\nwant %+v", decoded, spec)
	}
}

func TestCommitRangeSpec_JSON_MarshalInvalid(t *testing.T) {
	// Zero CommitRangeSpec is invalid
	spec := git.CommitRangeSpec{}

	_, err := json.Marshal(spec)
	if err == nil {
		t.Error("Expected error marshaling invalid CommitRangeSpec, got nil")
	}
}

func TestCommitRangeSpec_JSON_UnmarshalInvalid(t *testing.T) {
	// JSON with empty To (invalid)
	jsonData := `{"from":"v1.0.0","to":""}`

	var spec git.CommitRangeSpec
	err := json.Unmarshal([]byte(jsonData), &spec)
	if err == nil {
		t.Error("Expected error unmarshaling invalid CommitRangeSpec, got nil")
	}
}

func TestCommitRangeSpec_YAML_RoundTrip(t *testing.T) {
	spec := git.CommitRangeSpec{
		From: "v1.0.0",
		To:   "v2.0.0",
	}

	data, err := yaml.Marshal(spec)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded git.CommitRangeSpec
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if !decoded.Equal(spec) {
		t.Errorf("Round trip failed:\ngot  %+v\nwant %+v", decoded, spec)
	}
}

func TestCommitRangeSpec_YAML_MarshalInvalid(t *testing.T) {
	// Zero CommitRangeSpec is invalid
	spec := git.CommitRangeSpec{}

	_, err := yaml.Marshal(spec)
	if err == nil {
		t.Error("Expected error marshaling invalid CommitRangeSpec, got nil")
	}
}

func TestCommitRangeSpec_YAML_UnmarshalInvalid(t *testing.T) {
	// YAML with empty To (invalid)
	yamlData := `
from: v1.0.0
to: ""
`

	var spec git.CommitRangeSpec
	err := yaml.Unmarshal([]byte(yamlData), &spec)
	if err == nil {
		t.Error("Expected error unmarshaling invalid CommitRangeSpec, got nil")
	}
}

func TestCommitRangeSpec_CommonScenarios(t *testing.T) {
	scenarios := []struct {
		name  string
		spec  git.CommitRangeSpec
		valid bool
	}{
		{
			name: "release_to_release",
			spec: git.CommitRangeSpec{
				From: "v1.0.0",
				To:   "v2.0.0",
			},
			valid: true,
		},
		{
			name: "initial_release",
			spec: git.CommitRangeSpec{
				From: "", // From beginning
				To:   "v1.0.0",
			},
			valid: true,
		},
		{
			name: "branch_to_head",
			spec: git.CommitRangeSpec{
				From: "main",
				To:   "HEAD",
			},
			valid: true,
		},
		{
			name: "tag_to_branch",
			spec: git.CommitRangeSpec{
				From: "v1.0.0",
				To:   "develop",
			},
			valid: true,
		},
		{
			name: "with_revision_syntax",
			spec: git.CommitRangeSpec{
				From: "HEAD~10",
				To:   "HEAD",
			},
			valid: true,
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			err := sc.spec.Validate()
			if sc.valid && err != nil {
				t.Errorf("Expected valid CommitRangeSpec, got error: %v", err)
			}
			if !sc.valid && err == nil {
				t.Errorf("Expected invalid CommitRangeSpec, got nil error")
			}
		})
	}
}
