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

// =============================================================================
// TagName Tests
// =============================================================================

func TestParseTagName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    git.TagName
		wantErr bool
	}{
		{
			name:    "valid_simple_version",
			input:   "v1.2.3",
			want:    "v1.2.3",
			wantErr: false,
		},
		{
			name:    "valid_hierarchical",
			input:   "moduleA/v1.2.3",
			want:    "moduleA/v1.2.3",
			wantErr: false,
		},
		{
			name:    "valid_custom",
			input:   "release-2023-01-15",
			want:    "release-2023-01-15",
			wantErr: false,
		},
		{
			name:    "valid_single_char",
			input:   "v",
			want:    "v",
			wantErr: false,
		},
		{
			name:    "valid_with_whitespace_trimmed",
			input:   "  v1.2.3  ",
			want:    "v1.2.3",
			wantErr: false,
		},
		{
			name:    "empty_string",
			input:   "",
			want:    "",
			wantErr: false, // Zero value is valid
		},
		{
			name:    "whitespace_only",
			input:   "   ",
			want:    "",
			wantErr: false, // Trimmed to empty, zero value is valid
		},
		{
			name:    "too_long",
			input:   strings.Repeat("a", 257),
			want:    "",
			wantErr: true,
		},
		{
			name:    "contains_control_char",
			input:   "v1.2.3\x00",
			want:    "",
			wantErr: true,
		},
		{
			name:    "contains_non_ascii",
			input:   "v1.2.3привет",
			want:    "",
			wantErr: true,
		},
		{
			name:    "contains_space",
			input:   "v1 2 3",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := git.ParseTagName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTagName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseTagName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTagName_String(t *testing.T) {
	tests := []struct {
		name string
		tn   git.TagName
		want string
	}{
		{
			name: "simple_version",
			tn:   "v1.2.3",
			want: "v1.2.3",
		},
		{
			name: "hierarchical",
			tn:   "moduleA/v1.2.3",
			want: "moduleA/v1.2.3",
		},
		{
			name: "zero_value",
			tn:   "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tn.String(); got != tt.want {
				t.Errorf("TagName.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTagName_Redacted(t *testing.T) {
	tests := []struct {
		name string
		tn   git.TagName
		want string
	}{
		{
			name: "simple_version",
			tn:   "v1.2.3",
			want: "v1.2.3",
		},
		{
			name: "zero_value",
			tn:   "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tn.Redacted(); got != tt.want {
				t.Errorf("TagName.Redacted() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTagName_TypeName(t *testing.T) {
	var tn git.TagName
	if got := tn.TypeName(); got != "TagName" {
		t.Errorf("TagName.TypeName() = %v, want TagName", got)
	}
}

func TestTagName_IsZero(t *testing.T) {
	tests := []struct {
		name string
		tn   git.TagName
		want bool
	}{
		{
			name: "zero_value",
			tn:   "",
			want: true,
		},
		{
			name: "non_zero",
			tn:   "v1.2.3",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tn.IsZero(); got != tt.want {
				t.Errorf("TagName.IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTagName_Equal(t *testing.T) {
	tests := []struct {
		name  string
		tn    git.TagName
		other git.TagName
		want  bool
	}{
		{
			name:  "equal_simple",
			tn:    "v1.2.3",
			other: "v1.2.3",
			want:  true,
		},
		{
			name:  "not_equal",
			tn:    "v1.2.3",
			other: "v1.2.4",
			want:  false,
		},
		{
			name:  "case_sensitive",
			tn:    "v1.2.3",
			other: "V1.2.3",
			want:  false,
		},
		{
			name:  "both_zero",
			tn:    "",
			other: "",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tn.Equal(tt.other); got != tt.want {
				t.Errorf("TagName.Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTagName_Validate(t *testing.T) {
	tests := []struct {
		name    string
		tn      git.TagName
		wantErr bool
	}{
		{
			name:    "valid_simple",
			tn:      "v1.2.3",
			wantErr: false,
		},
		{
			name:    "valid_hierarchical",
			tn:      "moduleA/v1.2.3",
			wantErr: false,
		},
		{
			name:    "valid_custom",
			tn:      "release-2023-01-15",
			wantErr: false,
		},
		{
			name:    "valid_with_special_chars",
			tn:      "v1.2.3-rc.1+build.42",
			wantErr: false,
		},
		{
			name:    "valid_zero",
			tn:      "",
			wantErr: false,
		},
		{
			name:    "invalid_too_long",
			tn:      git.TagName(strings.Repeat("a", 257)),
			wantErr: true,
		},
		{
			name:    "invalid_control_char",
			tn:      "v1.2.3\x00",
			wantErr: true,
		},
		{
			name:    "invalid_non_ascii",
			tn:      "v1.2.3привет",
			wantErr: true,
		},
		{
			name:    "invalid_whitespace",
			tn:      "  v1.2.3  ",
			wantErr: true, // Not normalized
		},
		{
			name:    "invalid_space_in_middle",
			tn:      "v1 2 3",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tn.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("TagName.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTagName_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		tn      git.TagName
		want    string
		wantErr bool
	}{
		{
			name:    "valid_simple",
			tn:      "v1.2.3",
			want:    `"v1.2.3"`,
			wantErr: false,
		},
		{
			name:    "zero_value",
			tn:      "",
			want:    `""`,
			wantErr: false,
		},
		{
			name:    "invalid",
			tn:      "v1.2.3\x00",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.tn)
			if (err != nil) != tt.wantErr {
				t.Errorf("TagName.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(got) != tt.want {
				t.Errorf("TagName.MarshalJSON() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestTagName_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    git.TagName
		wantErr bool
	}{
		{
			name:    "valid_simple",
			json:    `"v1.2.3"`,
			want:    "v1.2.3",
			wantErr: false,
		},
		{
			name:    "empty_string",
			json:    `""`,
			want:    "",
			wantErr: false,
		},
		{
			name:    "with_whitespace",
			json:    `"  v1.2.3  "`,
			want:    "v1.2.3",
			wantErr: false,
		},
		{
			name:    "invalid_json",
			json:    `{invalid}`,
			wantErr: true,
		},
		{
			name:    "invalid_tag",
			json:    `"v1.2.3\u0000"`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got git.TagName
			err := json.Unmarshal([]byte(tt.json), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("TagName.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("TagName.UnmarshalJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTagName_MarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		tn      git.TagName
		want    string
		wantErr bool
	}{
		{
			name:    "valid_simple",
			tn:      "v1.2.3",
			want:    "v1.2.3\n",
			wantErr: false,
		},
		{
			name:    "zero_value",
			tn:      "",
			want:    "\"\"\n",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := yaml.Marshal(tt.tn)
			if (err != nil) != tt.wantErr {
				t.Errorf("TagName.MarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(got) != tt.want {
				t.Errorf("TagName.MarshalYAML() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestTagName_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		want    git.TagName
		wantErr bool
	}{
		{
			name:    "valid_simple",
			yaml:    "v1.2.3",
			want:    "v1.2.3",
			wantErr: false,
		},
		{
			name:    "empty_string",
			yaml:    `""`,
			want:    "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got git.TagName
			err := yaml.Unmarshal([]byte(tt.yaml), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("TagName.UnmarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("TagName.UnmarshalYAML() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTagName_JSON_RoundTrip(t *testing.T) {
	tests := []string{
		"v1.2.3",
		"moduleA/v1.2.3",
		"release-2023-01-15",
		"",
	}

	for _, original := range tests {
		t.Run(string(original), func(t *testing.T) {
			tn := git.TagName(original)

			data, err := json.Marshal(tn)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			var decoded git.TagName
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			if decoded != tn {
				t.Errorf("Round trip failed: got %v, want %v", decoded, tn)
			}
		})
	}
}

// =============================================================================
// Tag Tests
// =============================================================================

func TestNewTag(t *testing.T) {
	hash1 := git.Hash("a1b2c3d4e5f67890abcdef1234567890abcdef12")
	hash2 := git.Hash("1234567890abcdef1234567890abcdef12345678")

	tests := []struct {
		name      string
		tagName   git.TagName
		object    git.Hash
		commit    git.Hash
		annotated bool
		message   string
		wantErr   bool
	}{
		{
			name:      "valid_lightweight",
			tagName:   "v1.2.3",
			object:    hash1,
			commit:    hash1,
			annotated: false,
			message:   "",
			wantErr:   false,
		},
		{
			name:      "valid_annotated",
			tagName:   "v2.0.0",
			object:    hash1,
			commit:    hash2,
			annotated: true,
			message:   "Release v2.0.0",
			wantErr:   false,
		},
		{
			name:      "invalid_empty_name",
			tagName:   "",
			object:    hash1,
			commit:    hash1,
			annotated: false,
			message:   "",
			wantErr:   true,
		},
		{
			name:      "invalid_lightweight_with_message",
			tagName:   "v1.0.0",
			object:    hash1,
			commit:    hash1,
			annotated: false,
			message:   "oops",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := git.NewTag(tt.tagName, tt.object, tt.commit, tt.annotated, tt.message)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTag() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Name != tt.tagName {
					t.Errorf("NewTag().Name = %v, want %v", got.Name, tt.tagName)
				}
			}
		})
	}
}

func TestTag_String(t *testing.T) {
	hash := git.Hash("a1b2c3d4e5f67890abcdef1234567890abcdef12")

	tag := git.Tag{
		Name:      "v1.2.3",
		Object:    hash,
		Commit:    hash,
		Annotated: false,
	}

	str := tag.String()
	if !strings.Contains(str, "v1.2.3") {
		t.Errorf("Tag.String() doesn't contain tag name: %s", str)
	}
	if !strings.Contains(str, "Annotated:false") {
		t.Errorf("Tag.String() doesn't contain Annotated flag: %s", str)
	}
}

func TestTag_TypeName(t *testing.T) {
	var tag git.Tag
	if got := tag.TypeName(); got != "Tag" {
		t.Errorf("Tag.TypeName() = %v, want Tag", got)
	}
}

func TestTag_IsZero(t *testing.T) {
	tests := []struct {
		name string
		tag  git.Tag
		want bool
	}{
		{
			name: "zero_value",
			tag:  git.Tag{},
			want: true,
		},
		{
			name: "non_zero",
			tag: git.Tag{
				Name:   "v1.2.3",
				Object: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
				Commit: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tag.IsZero(); got != tt.want {
				t.Errorf("Tag.IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTag_Equal(t *testing.T) {
	hash1 := git.Hash("a1b2c3d4e5f67890abcdef1234567890abcdef12")
	hash2 := git.Hash("1234567890abcdef1234567890abcdef12345678")

	tests := []struct {
		name  string
		tag1  git.Tag
		tag2  git.Tag
		want  bool
	}{
		{
			name: "equal_lightweight",
			tag1: git.Tag{
				Name:   "v1.2.3",
				Object: hash1,
				Commit: hash1,
			},
			tag2: git.Tag{
				Name:   "v1.2.3",
				Object: hash1,
				Commit: hash1,
			},
			want: true,
		},
		{
			name: "different_name",
			tag1: git.Tag{
				Name:   "v1.2.3",
				Object: hash1,
				Commit: hash1,
			},
			tag2: git.Tag{
				Name:   "v1.2.4",
				Object: hash1,
				Commit: hash1,
			},
			want: false,
		},
		{
			name: "different_hashes",
			tag1: git.Tag{
				Name:   "v1.2.3",
				Object: hash1,
				Commit: hash1,
			},
			tag2: git.Tag{
				Name:   "v1.2.3",
				Object: hash2,
				Commit: hash2,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tag1.Equal(tt.tag2); got != tt.want {
				t.Errorf("Tag.Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTag_Validate(t *testing.T) {
	hash1 := git.Hash("a1b2c3d4e5f67890abcdef1234567890abcdef12")
	hash2 := git.Hash("1234567890abcdef1234567890abcdef12345678")

	tests := []struct {
		name    string
		tag     git.Tag
		wantErr bool
	}{
		{
			name: "valid_lightweight",
			tag: git.Tag{
				Name:      "v1.2.3",
				Object:    hash1,
				Commit:    hash1,
				Annotated: false,
				Message:   "",
			},
			wantErr: false,
		},
		{
			name: "valid_annotated",
			tag: git.Tag{
				Name:      "v2.0.0",
				Object:    hash1,
				Commit:    hash2,
				Annotated: true,
				Message:   "Release v2.0.0",
			},
			wantErr: false,
		},
		{
			name: "invalid_empty_name",
			tag: git.Tag{
				Name:   "",
				Object: hash1,
				Commit: hash1,
			},
			wantErr: true,
		},
		{
			name: "invalid_empty_object",
			tag: git.Tag{
				Name:   "v1.2.3",
				Object: "",
				Commit: hash1,
			},
			wantErr: true,
		},
		{
			name: "invalid_empty_commit",
			tag: git.Tag{
				Name:   "v1.2.3",
				Object: hash1,
				Commit: "",
			},
			wantErr: true,
		},
		{
			name: "invalid_lightweight_with_message",
			tag: git.Tag{
				Name:      "v1.0.0",
				Object:    hash1,
				Commit:    hash1,
				Annotated: false,
				Message:   "This should not be here",
			},
			wantErr: true,
		},
		{
			name: "invalid_message_too_long",
			tag: git.Tag{
				Name:      "v1.0.0",
				Object:    hash1,
				Commit:    hash1,
				Annotated: true,
				Message:   strings.Repeat("a", 65537), // > 64KB
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tag.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Tag.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTag_JSON_RoundTrip(t *testing.T) {
	hash1 := git.Hash("a1b2c3d4e5f67890abcdef1234567890abcdef12")
	hash2 := git.Hash("1234567890abcdef1234567890abcdef12345678")

	tests := []struct {
		name string
		tag  git.Tag
	}{
		{
			name: "lightweight",
			tag: git.Tag{
				Name:      "v1.2.3",
				Object:    hash1,
				Commit:    hash1,
				Annotated: false,
			},
		},
		{
			name: "annotated",
			tag: git.Tag{
				Name:      "v2.0.0",
				Object:    hash1,
				Commit:    hash2,
				Annotated: true,
				Message:   "Release v2.0.0\n\nMajor release",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.tag)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			var decoded git.Tag
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			if !decoded.Equal(tt.tag) {
				t.Errorf("Round trip failed:\ngot  %+v\nwant %+v", decoded, tt.tag)
			}
		})
	}
}

func TestTag_YAML_RoundTrip(t *testing.T) {
	hash1 := git.Hash("a1b2c3d4e5f67890abcdef1234567890abcdef12")

	tag := git.Tag{
		Name:      "v1.2.3",
		Object:    hash1,
		Commit:    hash1,
		Annotated: false,
	}

	data, err := yaml.Marshal(tag)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded git.Tag
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if !decoded.Equal(tag) {
		t.Errorf("Round trip failed:\ngot  %+v\nwant %+v", decoded, tag)
	}
}

func TestTag_CommonScenarios(t *testing.T) {
	hash1 := git.Hash("a1b2c3d4e5f67890abcdef1234567890abcdef12")
	hash2 := git.Hash("1234567890abcdef1234567890abcdef12345678")

	scenarios := []struct {
		name  string
		tag   git.Tag
		valid bool
	}{
		{
			name: "semver_lightweight",
			tag: git.Tag{
				Name:   "v1.2.3",
				Object: hash1,
				Commit: hash1,
			},
			valid: true,
		},
		{
			name: "semver_prerelease",
			tag: git.Tag{
				Name:   "v2.0.0-rc.1",
				Object: hash1,
				Commit: hash1,
			},
			valid: true,
		},
		{
			name: "hierarchical_tag",
			tag: git.Tag{
				Name:   "moduleA/v1.2.3",
				Object: hash1,
				Commit: hash1,
			},
			valid: true,
		},
		{
			name: "custom_tag",
			tag: git.Tag{
				Name:   "release-2023-01-15",
				Object: hash1,
				Commit: hash1,
			},
			valid: true,
		},
		{
			name: "annotated_with_message",
			tag: git.Tag{
				Name:      "v1.0.0",
				Object:    hash1,
				Commit:    hash2,
				Annotated: true,
				Message:   "First stable release\n\nIncludes all features from beta.",
			},
			valid: true,
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			err := sc.tag.Validate()
			if sc.valid && err != nil {
				t.Errorf("Expected valid tag, got error: %v", err)
			}
			if !sc.valid && err == nil {
				t.Errorf("Expected invalid tag, got nil error")
			}
		})
	}
}
