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

func TestRefName_String(t *testing.T) {
	tests := []struct {
		name    string
		refName git.RefName
		want    string
	}{
		{
			name:    "empty",
			refName: git.RefName(""),
			want:    "",
		},
		{
			name:    "branch",
			refName: git.RefName("refs/heads/main"),
			want:    "refs/heads/main",
		},
		{
			name:    "tag",
			refName: git.RefName("refs/tags/v1.0.0"),
			want:    "refs/tags/v1.0.0",
		},
		{
			name:    "HEAD",
			refName: git.RefName("HEAD"),
			want:    "HEAD",
		},
		{
			name:    "revision_expression",
			refName: git.RefName("HEAD~3"),
			want:    "HEAD~3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.refName.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRefName_Redacted(t *testing.T) {
	tests := []struct {
		name    string
		refName git.RefName
		want    string
	}{
		{
			name:    "empty",
			refName: git.RefName(""),
			want:    "",
		},
		{
			name:    "branch",
			refName: git.RefName("refs/heads/main"),
			want:    "refs/heads/main",
		},
		{
			name:    "sensitive_looking_ref",
			refName: git.RefName("refs/heads/secret/feature"),
			want:    "refs/heads/secret/feature",
		},
		{
			name:    "revision",
			refName: git.RefName("main@{upstream}"),
			want:    "main@{upstream}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.refName.Redacted()
			if got != tt.want {
				t.Errorf("Redacted() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRefName_TypeName(t *testing.T) {
	var rn git.RefName
	got := rn.TypeName()
	want := "RefName"
	if got != want {
		t.Errorf("TypeName() = %q, want %q", got, want)
	}
}

func TestRefName_IsZero(t *testing.T) {
	tests := []struct {
		name    string
		refName git.RefName
		want    bool
	}{
		{
			name:    "empty_is_zero",
			refName: git.RefName(""),
			want:    true,
		},
		{
			name:    "branch_not_zero",
			refName: git.RefName("main"),
			want:    false,
		},
		{
			name:    "HEAD_not_zero",
			refName: git.RefName("HEAD"),
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.refName.IsZero()
			if got != tt.want {
				t.Errorf("IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRefName_Equal(t *testing.T) {
	tests := []struct {
		name  string
		rn1   git.RefName
		rn2   git.RefName
		want  bool
	}{
		{
			name: "both_empty",
			rn1:  git.RefName(""),
			rn2:  git.RefName(""),
			want: true,
		},
		{
			name: "same_branch",
			rn1:  git.RefName("refs/heads/main"),
			rn2:  git.RefName("refs/heads/main"),
			want: true,
		},
		{
			name: "different_branches",
			rn1:  git.RefName("refs/heads/main"),
			rn2:  git.RefName("refs/heads/develop"),
			want: false,
		},
		{
			name: "case_difference",
			rn1:  git.RefName("refs/heads/Main"),
			rn2:  git.RefName("refs/heads/main"),
			want: false,
		},
		{
			name: "empty_vs_non_empty",
			rn1:  git.RefName(""),
			rn2:  git.RefName("HEAD"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.rn1.Equal(tt.rn2)
			if got != tt.want {
				t.Errorf("Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRefName_Validate(t *testing.T) {
	tests := []struct {
		name    string
		refName git.RefName
		wantErr bool
	}{
		{
			name:    "empty_valid",
			refName: git.RefName(""),
			wantErr: false,
		},
		{
			name:    "branch_valid",
			refName: git.RefName("refs/heads/main"),
			wantErr: false,
		},
		{
			name:    "tag_valid",
			refName: git.RefName("refs/tags/v1.0.0"),
			wantErr: false,
		},
		{
			name:    "remote_branch_valid",
			refName: git.RefName("refs/remotes/origin/main"),
			wantErr: false,
		},
		{
			name:    "HEAD_valid",
			refName: git.RefName("HEAD"),
			wantErr: false,
		},
		{
			name:    "FETCH_HEAD_valid",
			refName: git.RefName("FETCH_HEAD"),
			wantErr: false,
		},
		{
			name:    "revision_tilde_valid",
			refName: git.RefName("HEAD~3"),
			wantErr: false,
		},
		{
			name:    "revision_caret_valid",
			refName: git.RefName("main^2"),
			wantErr: false,
		},
		{
			name:    "revision_at_valid",
			refName: git.RefName("main@{upstream}"),
			wantErr: false,
		},
		{
			name:    "revision_date_valid",
			refName: git.RefName("main@{2023-01-01}"),
			wantErr: false,
		},
		{
			name:    "short_branch_name",
			refName: git.RefName("a"),
			wantErr: false,
		},
		{
			name:    "feature_branch_valid",
			refName: git.RefName("feature/user/alice/new-thing"),
			wantErr: false,
		},
		{
			name:    "hash_sha1_valid",
			refName: git.RefName("a1b2c3d4e5f67890abcdef1234567890abcdef12"),
			wantErr: false,
		},
		{
			name:    "abbreviated_hash_valid",
			refName: git.RefName("a1b2c3d"),
			wantErr: false,
		},
		{
			name:    "with_leading_whitespace",
			refName: git.RefName("  main"),
			wantErr: true,
		},
		{
			name:    "with_trailing_whitespace",
			refName: git.RefName("main  "),
			wantErr: true,
		},
		{
			name:    "with_control_char",
			refName: git.RefName("main\x00"),
			wantErr: true,
		},
		{
			name:    "with_newline",
			refName: git.RefName("main\n"),
			wantErr: true,
		},
		{
			name:    "too_long",
			refName: git.RefName(strings.Repeat("a", 257)),
			wantErr: true,
		},
		{
			name:    "with_invalid_char_space",
			refName: git.RefName("refs/heads/my branch"),
			wantErr: true,
		},
		{
			name:    "with_invalid_char_backslash",
			refName: git.RefName("refs\\heads\\main"),
			wantErr: true,
		},
		{
			name:    "with_non_ascii",
			refName: git.RefName("refs/heads/фича"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.refName.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRefName_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		refName git.RefName
		wantErr bool
	}{
		{
			name:    "empty",
			refName: git.RefName(""),
			wantErr: false,
		},
		{
			name:    "branch",
			refName: git.RefName("refs/heads/main"),
			wantErr: false,
		},
		{
			name:    "tag",
			refName: git.RefName("refs/tags/v1.0.0"),
			wantErr: false,
		},
		{
			name:    "HEAD",
			refName: git.RefName("HEAD"),
			wantErr: false,
		},
		{
			name:    "revision",
			refName: git.RefName("HEAD~1"),
			wantErr: false,
		},
		{
			name:    "invalid_with_control_char",
			refName: git.RefName("main\x00"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.refName.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Verify round-trip
				var decoded git.RefName
				if err := json.Unmarshal(got, &decoded); err != nil {
					t.Errorf("MarshalJSON() produced invalid JSON: %v", err)
				}
				if !decoded.Equal(tt.refName) {
					t.Errorf("MarshalJSON() round-trip failed: got %q, want %q", decoded, tt.refName)
				}
			}
		})
	}
}

func TestRefName_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    git.RefName
		wantErr bool
	}{
		{
			name:    "empty",
			json:    `""`,
			want:    git.RefName(""),
			wantErr: false,
		},
		{
			name:    "branch",
			json:    `"refs/heads/main"`,
			want:    git.RefName("refs/heads/main"),
			wantErr: false,
		},
		{
			name:    "tag",
			json:    `"refs/tags/v1.0.0"`,
			want:    git.RefName("refs/tags/v1.0.0"),
			wantErr: false,
		},
		{
			name:    "HEAD",
			json:    `"HEAD"`,
			want:    git.RefName("HEAD"),
			wantErr: false,
		},
		{
			name:    "with_whitespace_normalized",
			json:    `"  feature/new-thing  "`,
			want:    git.RefName("feature/new-thing"),
			wantErr: false,
		},
		{
			name:    "revision_expression",
			json:    `"main@{upstream}"`,
			want:    git.RefName("main@{upstream}"),
			wantErr: false,
		},
		{
			name:    "invalid_control_char",
			json:    `"main\u0000"`,
			want:    git.RefName(""),
			wantErr: true,
		},
		{
			name:    "invalid_json",
			json:    `not-json`,
			want:    git.RefName(""),
			wantErr: true,
		},
		{
			name:    "too_long",
			json:    `"` + strings.Repeat("a", 257) + `"`,
			want:    git.RefName(""),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got git.RefName
			err := json.Unmarshal([]byte(tt.json), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.want) {
				t.Errorf("UnmarshalJSON() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRefName_MarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		refName git.RefName
		wantErr bool
	}{
		{
			name:    "empty",
			refName: git.RefName(""),
			wantErr: false,
		},
		{
			name:    "branch",
			refName: git.RefName("refs/heads/main"),
			wantErr: false,
		},
		{
			name:    "tag",
			refName: git.RefName("refs/tags/v1.0.0"),
			wantErr: false,
		},
		{
			name:    "revision",
			refName: git.RefName("HEAD~1"),
			wantErr: false,
		},
		{
			name:    "invalid_control_char",
			refName: git.RefName("main\x00"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := yaml.Marshal(tt.refName)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Verify round-trip
				var decoded git.RefName
				if err := yaml.Unmarshal(got, &decoded); err != nil {
					t.Errorf("MarshalYAML() produced invalid YAML: %v", err)
				}
				if !decoded.Equal(tt.refName) {
					t.Errorf("MarshalYAML() round-trip failed: got %q, want %q", decoded, tt.refName)
				}
			}
		})
	}
}

func TestRefName_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		want    git.RefName
		wantErr bool
	}{
		{
			name:    "empty",
			yaml:    `""`,
			want:    git.RefName(""),
			wantErr: false,
		},
		{
			name:    "branch",
			yaml:    `refs/heads/main`,
			want:    git.RefName("refs/heads/main"),
			wantErr: false,
		},
		{
			name:    "tag",
			yaml:    `refs/tags/v1.0.0`,
			want:    git.RefName("refs/tags/v1.0.0"),
			wantErr: false,
		},
		{
			name:    "HEAD",
			yaml:    `HEAD`,
			want:    git.RefName("HEAD"),
			wantErr: false,
		},
		{
			name:    "with_whitespace_normalized",
			yaml:    `"  feature/new  "`,
			want:    git.RefName("feature/new"),
			wantErr: false,
		},
		{
			name:    "revision",
			yaml:    `main@{upstream}`,
			want:    git.RefName("main@{upstream}"),
			wantErr: false,
		},
		{
			name:    "invalid_too_long",
			yaml:    strings.Repeat("a", 257),
			want:    git.RefName(""),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got git.RefName
			err := yaml.Unmarshal([]byte(tt.yaml), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.want) {
				t.Errorf("UnmarshalYAML() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseRefName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    git.RefName
		wantErr bool
	}{
		{
			name:    "empty_string",
			input:   "",
			want:    git.RefName(""),
			wantErr: false,
		},
		{
			name:    "whitespace_only",
			input:   "   ",
			want:    git.RefName(""),
			wantErr: false,
		},
		{
			name:    "branch",
			input:   "refs/heads/main",
			want:    git.RefName("refs/heads/main"),
			wantErr: false,
		},
		{
			name:    "tag",
			input:   "refs/tags/v1.0.0",
			want:    git.RefName("refs/tags/v1.0.0"),
			wantErr: false,
		},
		{
			name:    "remote_branch",
			input:   "refs/remotes/origin/develop",
			want:    git.RefName("refs/remotes/origin/develop"),
			wantErr: false,
		},
		{
			name:    "HEAD",
			input:   "HEAD",
			want:    git.RefName("HEAD"),
			wantErr: false,
		},
		{
			name:    "revision_tilde",
			input:   "HEAD~3",
			want:    git.RefName("HEAD~3"),
			wantErr: false,
		},
		{
			name:    "revision_caret",
			input:   "main^2",
			want:    git.RefName("main^2"),
			wantErr: false,
		},
		{
			name:    "revision_at_upstream",
			input:   "main@{upstream}",
			want:    git.RefName("main@{upstream}"),
			wantErr: false,
		},
		{
			name:    "revision_at_date",
			input:   "develop@{2023-01-01}",
			want:    git.RefName("develop@{2023-01-01}"),
			wantErr: false,
		},
		{
			name:    "with_leading_whitespace",
			input:   "  feature/new-thing",
			want:    git.RefName("feature/new-thing"),
			wantErr: false,
		},
		{
			name:    "with_trailing_whitespace",
			input:   "feature/new-thing  ",
			want:    git.RefName("feature/new-thing"),
			wantErr: false,
		},
		{
			name:    "with_surrounding_whitespace",
			input:   "  main  ",
			want:    git.RefName("main"),
			wantErr: false,
		},
		{
			name:    "short_name",
			input:   "m",
			want:    git.RefName("m"),
			wantErr: false,
		},
		{
			name:    "hash_sha1",
			input:   "a1b2c3d4e5f67890abcdef1234567890abcdef12",
			want:    git.RefName("a1b2c3d4e5f67890abcdef1234567890abcdef12"),
			wantErr: false,
		},
		{
			name:    "abbreviated_hash",
			input:   "a1b2c3d",
			want:    git.RefName("a1b2c3d"),
			wantErr: false,
		},
		{
			name:    "feature_branch_with_slashes",
			input:   "feature/team/alice/new-ui",
			want:    git.RefName("feature/team/alice/new-ui"),
			wantErr: false,
		},
		{
			name:    "with_dots",
			input:   "v1.0.0",
			want:    git.RefName("v1.0.0"),
			wantErr: false,
		},
		{
			name:    "with_hyphens",
			input:   "fix-bug-123",
			want:    git.RefName("fix-bug-123"),
			wantErr: false,
		},
		{
			name:    "with_underscores",
			input:   "feature_new_thing",
			want:    git.RefName("feature_new_thing"),
			wantErr: false,
		},
		{
			name:    "invalid_control_char",
			input:   "main\x00",
			want:    git.RefName(""),
			wantErr: true,
		},
		{
			name:    "invalid_newline",
			input:   "main\n",
			want:    git.RefName(""),
			wantErr: true,
		},
		{
			name:    "invalid_tab",
			input:   "main\tdev",
			want:    git.RefName(""),
			wantErr: true,
		},
		{
			name:    "invalid_space",
			input:   "my branch",
			want:    git.RefName(""),
			wantErr: true,
		},
		{
			name:    "invalid_backslash",
			input:   "refs\\heads\\main",
			want:    git.RefName(""),
			wantErr: true,
		},
		{
			name:    "invalid_non_ascii",
			input:   "фича",
			want:    git.RefName(""),
			wantErr: true,
		},
		{
			name:    "invalid_too_long",
			input:   strings.Repeat("a", 257),
			want:    git.RefName(""),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := git.ParseRefName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRefName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !got.Equal(tt.want) {
				t.Errorf("ParseRefName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRefName_JSON_RoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		refName git.RefName
	}{
		{
			name:    "empty",
			refName: git.RefName(""),
		},
		{
			name:    "branch",
			refName: git.RefName("refs/heads/main"),
		},
		{
			name:    "tag",
			refName: git.RefName("refs/tags/v1.0.0"),
		},
		{
			name:    "HEAD",
			refName: git.RefName("HEAD"),
		},
		{
			name:    "revision",
			refName: git.RefName("HEAD~3"),
		},
		{
			name:    "feature_branch",
			refName: git.RefName("feature/new-thing"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.refName)
			if err != nil {
				t.Fatalf("Marshal() failed: %v", err)
			}

			var decoded git.RefName
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal() failed: %v", err)
			}

			if !decoded.Equal(tt.refName) {
				t.Errorf("Round-trip failed: got %q, want %q", decoded, tt.refName)
			}
		})
	}
}

func TestRefName_YAML_RoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		refName git.RefName
	}{
		{
			name:    "empty",
			refName: git.RefName(""),
		},
		{
			name:    "branch",
			refName: git.RefName("refs/heads/main"),
		},
		{
			name:    "tag",
			refName: git.RefName("refs/tags/v1.0.0"),
		},
		{
			name:    "HEAD",
			refName: git.RefName("HEAD"),
		},
		{
			name:    "revision",
			refName: git.RefName("HEAD~3"),
		},
		{
			name:    "feature_branch",
			refName: git.RefName("feature/new-thing"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := yaml.Marshal(tt.refName)
			if err != nil {
				t.Fatalf("Marshal() failed: %v", err)
			}

			var decoded git.RefName
			if err := yaml.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal() failed: %v", err)
			}

			if !decoded.Equal(tt.refName) {
				t.Errorf("Round-trip failed: got %q, want %q", decoded, tt.refName)
			}
		})
	}
}

func TestRefName_CommonRefs(t *testing.T) {
	// Test common Git reference names to ensure they all validate correctly
	commonRefs := []string{
		"refs/heads/main",
		"refs/heads/master",
		"refs/heads/develop",
		"refs/heads/feature/new-thing",
		"refs/heads/bugfix/issue-123",
		"refs/heads/release/v1.0",
		"refs/tags/v1.0.0",
		"refs/tags/v2.1.3-beta.1",
		"refs/remotes/origin/main",
		"refs/remotes/upstream/develop",
		"HEAD",
		"FETCH_HEAD",
		"ORIG_HEAD",
		"MERGE_HEAD",
		"HEAD~1",
		"HEAD~10",
		"HEAD^",
		"HEAD^2",
		"main@{upstream}",
		"main@{u}",
		"develop@{yesterday}",
		"main@{2023-01-01}",
		"v1.0.0^{}",
		"main^{tree}",
		"HEAD:README.md",
	}

	for _, refStr := range commonRefs {
		t.Run(refStr, func(t *testing.T) {
			ref, err := git.ParseRefName(refStr)
			if err != nil {
				t.Errorf("ParseRefName(%q) failed: %v", refStr, err)
				return
			}

			if err := ref.Validate(); err != nil {
				t.Errorf("Validate() failed for %q: %v", refStr, err)
			}

			if ref.String() != refStr {
				t.Errorf("String() = %q, want %q", ref.String(), refStr)
			}
		})
	}
}

func TestRefName_LengthValidation(t *testing.T) {
	tests := []struct {
		name    string
		length  int
		wantErr bool
	}{
		{
			name:    "empty",
			length:  0,
			wantErr: false, // Zero value is valid
		},
		{
			name:    "min_length_1",
			length:  1,
			wantErr: false,
		},
		{
			name:    "normal_length_50",
			length:  50,
			wantErr: false,
		},
		{
			name:    "max_length_256",
			length:  256,
			wantErr: false,
		},
		{
			name:    "too_long_257",
			length:  257,
			wantErr: true,
		},
		{
			name:    "way_too_long_1000",
			length:  1000,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use 'a' as a simple valid character
			refStr := strings.Repeat("a", tt.length)
			ref := git.RefName(refStr)

			err := ref.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() with length %d: error = %v, wantErr %v", tt.length, err, tt.wantErr)
			}
		})
	}
}
