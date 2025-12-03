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

// ============================================================================
// RefKind Tests
// ============================================================================

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
