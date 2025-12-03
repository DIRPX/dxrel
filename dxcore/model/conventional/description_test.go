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

package conventional_test

import (
	"encoding/json"
	"strings"
	"testing"

	"dirpx.dev/dxrel/dxcore/model/conventional"
	"gopkg.in/yaml.v3"
)

func TestDescription_String(t *testing.T) {
	tests := []struct {
		name string
		desc conventional.Description
		want string
	}{
		{"empty", conventional.Description(""), ""},
		{"simple", conventional.Description("add user endpoint"), "add user endpoint"},
		{"with capitals", conventional.Description("Add User Endpoint"), "Add User Endpoint"},
		{"with punctuation", conventional.Description("fix: handle nil pointer!"), "fix: handle nil pointer!"},
		{"with emoji", conventional.Description("add feature 🚀"), "add feature 🚀"},
		{"with unicode", conventional.Description("добавить функцию"), "добавить функцию"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.desc.String(); got != tt.want {
				t.Errorf("Description.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDescription_Redacted(t *testing.T) {
	// Redacted should be identical to String for Description
	desc := conventional.Description("add user endpoint")
	if desc.Redacted() != desc.String() {
		t.Errorf("Redacted() = %q, want %q", desc.Redacted(), desc.String())
	}
}

func TestDescription_TypeName(t *testing.T) {
	desc := conventional.Description("add user endpoint")
	if got := desc.TypeName(); got != "Description" {
		t.Errorf("TypeName() = %q, want %q", got, "Description")
	}
}

func TestDescription_IsZero(t *testing.T) {
	tests := []struct {
		name string
		desc conventional.Description
		want bool
	}{
		{"empty is zero", conventional.Description(""), true},
		{"non-empty is not zero", conventional.Description("add feature"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.desc.IsZero(); got != tt.want {
				t.Errorf("IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDescription_Equal(t *testing.T) {
	tests := []struct {
		name string
		d1   conventional.Description
		d2   conventional.Description
		want bool
	}{
		{"both empty", conventional.Description(""), conventional.Description(""), true},
		{"same lowercase", conventional.Description("add feature"), conventional.Description("add feature"), true},
		{"same with capitals", conventional.Description("Add Feature"), conventional.Description("Add Feature"), true},
		{"same with emoji", conventional.Description("add 🚀"), conventional.Description("add 🚀"), true},
		{"different content", conventional.Description("add feature"), conventional.Description("fix bug"), false},
		{"different case", conventional.Description("add feature"), conventional.Description("Add Feature"), false},
		{"empty vs non-empty", conventional.Description(""), conventional.Description("add feature"), false},
		{"different emoji", conventional.Description("add 🚀"), conventional.Description("add ✨"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d1.Equal(tt.d2); got != tt.want {
				t.Errorf("Description.Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDescription_Validate(t *testing.T) {
	tests := []struct {
		name    string
		desc    conventional.Description
		wantErr bool
	}{
		// Valid descriptions
		{"empty valid", conventional.Description(""), false},
		{"simple valid", conventional.Description("add user endpoint"), false},
		{"with capitals", conventional.Description("Add Feature"), false},
		{"single char", conventional.Description("x"), false},
		{"max length 72", conventional.Description(strings.Repeat("a", 72)), false},
		{"with emoji (counts as 1 rune)", conventional.Description("add feature 🚀"), false},
		{"with unicode", conventional.Description("добавить функцию"), false},
		{"with punctuation", conventional.Description("fix: handle error!"), false},
		{"with internal spaces", conventional.Description("add user registration endpoint"), false},

		// Invalid descriptions
		{"contains LF newline", conventional.Description("add\nfeature"), true},
		{"contains CR newline", conventional.Description("add\rfeature"), true},
		{"contains CRLF newline", conventional.Description("add\r\nfeature"), true},
		{"too long 73 chars", conventional.Description(strings.Repeat("a", 73)), true},
		{"way too long", conventional.Description(strings.Repeat("a", 100)), true},
		{"only spaces", conventional.Description("   "), true},
		{"only tabs", conventional.Description("\t\t"), true},
		{"mixed whitespace only", conventional.Description(" \t \n "), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.desc.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDescription_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		desc    conventional.Description
		want    string
		wantErr bool
	}{
		{"empty", conventional.Description(""), `""`, false},
		{"simple", conventional.Description("add user endpoint"), `"add user endpoint"`, false},
		{"with capitals", conventional.Description("Add Feature"), `"Add Feature"`, false},
		{"with emoji", conventional.Description("add 🚀"), `"add 🚀"`, false},
		{"invalid newline", conventional.Description("add\nfeature"), "", true},
		{"invalid too long", conventional.Description(strings.Repeat("a", 73)), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.desc)
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

func TestDescription_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    conventional.Description
		wantErr bool
	}{
		{"empty", `""`, conventional.Description(""), false},
		{"simple lowercase", `"add user endpoint"`, conventional.Description("add user endpoint"), false},
		{"with capitals preserved", `"Add User Endpoint"`, conventional.Description("Add User Endpoint"), false},
		{"with whitespace trimmed", `"  fix bug  "`, conventional.Description("fix bug"), false},
		{"with emoji", `"add feature 🚀"`, conventional.Description("add feature 🚀"), false},
		{"with unicode", `"добавить функцию"`, conventional.Description("добавить функцию"), false},
		{"invalid newline", `"add\nfeature"`, conventional.Description(""), true},
		{"invalid too long", `"` + strings.Repeat("a", 73) + `"`, conventional.Description(""), true},
		{"invalid JSON", `not json`, conventional.Description(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got conventional.Description
			err := json.Unmarshal([]byte(tt.data), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("UnmarshalJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDescription_MarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		desc    conventional.Description
		want    string
		wantErr bool
	}{
		{"empty", conventional.Description(""), "\"\"\n", false},
		{"simple", conventional.Description("add user endpoint"), "add user endpoint\n", false},
		{"with capitals", conventional.Description("Fix Bug"), "Fix Bug\n", false},
		{"invalid newline", conventional.Description("add\nfeature"), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := yaml.Marshal(tt.desc)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(got) != tt.want {
				t.Errorf("MarshalYAML() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDescription_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    conventional.Description
		wantErr bool
	}{
		{"empty", `""`, conventional.Description(""), false},
		{"simple", "add user endpoint", conventional.Description("add user endpoint"), false},
		{"with capitals preserved", "Add User Endpoint", conventional.Description("Add User Endpoint"), false},
		{"with whitespace", "  fix bug  ", conventional.Description("fix bug"), false},
		{"with emoji", "add 🚀", conventional.Description("add 🚀"), false},
		{"invalid too long", strings.Repeat("a", 73), conventional.Description(""), true},
		// Note: YAML multiline string handling is complex, tested separately in newline validation
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got conventional.Description
			err := yaml.Unmarshal([]byte(tt.data), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("UnmarshalYAML() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseDescription(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    conventional.Description
		wantErr bool
	}{
		// Valid inputs
		{"empty", "", conventional.Description(""), false},
		{"simple", "add user endpoint", conventional.Description("add user endpoint"), false},
		{"with capitals preserved", "Add User Endpoint", conventional.Description("Add User Endpoint"), false},
		{"with leading whitespace", "  add feature", conventional.Description("add feature"), false},
		{"with trailing whitespace", "fix bug  ", conventional.Description("fix bug"), false},
		{"with surrounding whitespace", "  update docs  ", conventional.Description("update docs"), false},
		{"with tabs", "\tremove code\t", conventional.Description("remove code"), false},
		{"only whitespace", "   ", conventional.Description(""), false},
		{"with emoji", "add feature 🚀", conventional.Description("add feature 🚀"), false},
		{"with unicode", "добавить функцию", conventional.Description("добавить функцию"), false},
		{"max length", strings.Repeat("a", 72), conventional.Description(strings.Repeat("a", 72)), false},

		// Invalid inputs
		{"too long", strings.Repeat("a", 73), conventional.Description(""), true},
		{"contains LF newline", "add\nfeature", conventional.Description(""), true},
		{"contains CR newline", "add\rfeature", conventional.Description(""), true},
		{"contains CRLF", "add\r\nfeature", conventional.Description(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := conventional.ParseDescription(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDescription() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseDescription() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDescription_JSON_RoundTrip(t *testing.T) {
	original := conventional.Description("add user registration endpoint")

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded conventional.Description
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded != original {
		t.Errorf("JSON round-trip failed: got %v, want %v", decoded, original)
	}
}

func TestDescription_YAML_RoundTrip(t *testing.T) {
	original := conventional.Description("fix authentication bug")

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("yaml.Marshal() error = %v", err)
	}

	var decoded conventional.Description
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	if decoded != original {
		t.Errorf("YAML round-trip failed: got %v, want %v", decoded, original)
	}
}

func TestDescription_LengthConstraints(t *testing.T) {
	// Test minimum length (1 rune, non-empty)
	minDesc := conventional.Description("x")
	if err := minDesc.Validate(); err != nil {
		t.Errorf("Description with min length should be valid, got error: %v", err)
	}

	// Test maximum length (72 runes)
	maxDesc := conventional.Description(strings.Repeat("a", 72))
	if err := maxDesc.Validate(); err != nil {
		t.Errorf("Description with max length should be valid, got error: %v", err)
	}

	// Test over maximum length (73 runes)
	tooLongDesc := conventional.Description(strings.Repeat("a", 73))
	if err := tooLongDesc.Validate(); err == nil {
		t.Error("Description over max length should be invalid")
	}
}

func TestDescription_RuneCounting(t *testing.T) {
	tests := []struct {
		name      string
		desc      conventional.Description
		runeCount int
		wantErr   bool
	}{
		{"ASCII only", conventional.Description("add feature"), 11, false},
		{"emoji counts as 1 rune", conventional.Description("add 🚀"), 5, false},
		{"unicode Cyrillic", conventional.Description("добавить"), 8, false},
		{"mixed ASCII and emoji", conventional.Description(strings.Repeat("a", 70) + "🚀"), 71, false},
		{"emoji at max length", conventional.Description(strings.Repeat("🚀", 72)), 72, false},
		{"emoji exceeds max", conventional.Description(strings.Repeat("🚀", 73)), 73, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualRuneCount := len([]rune(string(tt.desc)))
			if actualRuneCount != tt.runeCount {
				t.Errorf("Rune count = %d, want %d", actualRuneCount, tt.runeCount)
			}

			err := tt.desc.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v for %d runes", err, tt.wantErr, tt.runeCount)
			}
		})
	}
}

func TestDescription_NewlineValidation(t *testing.T) {
	tests := []struct {
		name    string
		desc    conventional.Description
		wantErr bool
	}{
		{"no newline", conventional.Description("add feature"), false},
		{"LF newline", conventional.Description("add\nfeature"), true},
		{"CR newline", conventional.Description("add\rfeature"), true},
		{"CRLF newline", conventional.Description("add\r\nfeature"), true},
		{"multiple newlines", conventional.Description("add\n\nfeature"), true},
		{"trailing newline", conventional.Description("add feature\n"), true},
		{"leading newline", conventional.Description("\nadd feature"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.desc.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v for description %q", err, tt.wantErr, tt.desc)
			}
		})
	}
}

func TestDescription_WhitespaceValidation(t *testing.T) {
	tests := []struct {
		name    string
		desc    conventional.Description
		wantErr bool
	}{
		{"normal text", conventional.Description("add feature"), false},
		{"with internal spaces", conventional.Description("add user endpoint"), false},
		{"only spaces", conventional.Description("   "), true},
		{"only tabs", conventional.Description("\t\t"), true},
		{"only newlines", conventional.Description("\n\n"), true},
		{"mixed whitespace", conventional.Description(" \t \n "), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.desc.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v for description %q", err, tt.wantErr, tt.desc)
			}
		})
	}
}

func TestDescription_CasePreservation(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  conventional.Description
	}{
		{"lowercase", "add feature", conventional.Description("add feature")},
		{"uppercase", "ADD FEATURE", conventional.Description("ADD FEATURE")},
		{"mixed case", "Add User Endpoint", conventional.Description("Add User Endpoint")},
		{"sentence case", "Fix authentication bug", conventional.Description("Fix authentication bug")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := conventional.ParseDescription(tt.input)
			if err != nil {
				t.Fatalf("ParseDescription() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("ParseDescription() = %v, want %v (case should be preserved)", got, tt.want)
			}
		})
	}
}

func TestDescription_Normalization(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  conventional.Description
	}{
		{"trims leading space", "  add feature", conventional.Description("add feature")},
		{"trims trailing space", "add feature  ", conventional.Description("add feature")},
		{"trims both", "  add feature  ", conventional.Description("add feature")},
		{"trims tabs", "\tadd feature\t", conventional.Description("add feature")},
		{"preserves internal spaces", "add user endpoint", conventional.Description("add user endpoint")},
		{"does not lowercase", "  Add Feature  ", conventional.Description("Add Feature")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := conventional.ParseDescription(tt.input)
			if err != nil {
				t.Fatalf("ParseDescription() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("ParseDescription() = %v, want %v", got, tt.want)
			}
		})
	}
}
