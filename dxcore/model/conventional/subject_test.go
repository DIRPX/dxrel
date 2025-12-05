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

func TestSubject_String(t *testing.T) {
	tests := []struct {
		name string
		desc conventional.Subject
		want string
	}{
		{"empty", conventional.Subject(""), ""},
		{"simple", conventional.Subject("add user endpoint"), "add user endpoint"},
		{"with capitals", conventional.Subject("Add User Endpoint"), "Add User Endpoint"},
		{"with punctuation", conventional.Subject("fix: handle nil pointer!"), "fix: handle nil pointer!"},
		{"with emoji", conventional.Subject("add feature 🚀"), "add feature 🚀"},
		{"with unicode", conventional.Subject("добавить функцию"), "добавить функцию"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.desc.String(); got != tt.want {
				t.Errorf("Subject.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSubject_Redacted(t *testing.T) {
	// Redacted should be identical to String for Subject
	desc := conventional.Subject("add user endpoint")
	if desc.Redacted() != desc.String() {
		t.Errorf("Redacted() = %q, want %q", desc.Redacted(), desc.String())
	}
}

func TestSubject_TypeName(t *testing.T) {
	desc := conventional.Subject("add user endpoint")
	if got := desc.TypeName(); got != "Subject" {
		t.Errorf("TypeName() = %q, want %q", got, "Subject")
	}
}

func TestSubject_IsZero(t *testing.T) {
	tests := []struct {
		name string
		desc conventional.Subject
		want bool
	}{
		{"empty is zero", conventional.Subject(""), true},
		{"non-empty is not zero", conventional.Subject("add feature"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.desc.IsZero(); got != tt.want {
				t.Errorf("IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSubject_Equal(t *testing.T) {
	tests := []struct {
		name string
		d1   conventional.Subject
		d2   conventional.Subject
		want bool
	}{
		{"both empty", conventional.Subject(""), conventional.Subject(""), true},
		{"same lowercase", conventional.Subject("add feature"), conventional.Subject("add feature"), true},
		{"same with capitals", conventional.Subject("Add Feature"), conventional.Subject("Add Feature"), true},
		{"same with emoji", conventional.Subject("add 🚀"), conventional.Subject("add 🚀"), true},
		{"different content", conventional.Subject("add feature"), conventional.Subject("fix bug"), false},
		{"different case", conventional.Subject("add feature"), conventional.Subject("Add Feature"), false},
		{"empty vs non-empty", conventional.Subject(""), conventional.Subject("add feature"), false},
		{"different emoji", conventional.Subject("add 🚀"), conventional.Subject("add ✨"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d1.Equal(tt.d2); got != tt.want {
				t.Errorf("Subject.Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSubject_Validate(t *testing.T) {
	tests := []struct {
		name    string
		desc    conventional.Subject
		wantErr bool
	}{
		// Valid descriptions
		{"empty valid", conventional.Subject(""), false},
		{"simple valid", conventional.Subject("add user endpoint"), false},
		{"with capitals", conventional.Subject("Add Feature"), false},
		{"single char", conventional.Subject("x"), false},
		{"max length 72", conventional.Subject(strings.Repeat("a", 72)), false},
		{"with emoji (counts as 1 rune)", conventional.Subject("add feature 🚀"), false},
		{"with unicode", conventional.Subject("добавить функцию"), false},
		{"with punctuation", conventional.Subject("fix: handle error!"), false},
		{"with internal spaces", conventional.Subject("add user registration endpoint"), false},

		// Invalid descriptions
		{"contains LF newline", conventional.Subject("add\nfeature"), true},
		{"contains CR newline", conventional.Subject("add\rfeature"), true},
		{"contains CRLF newline", conventional.Subject("add\r\nfeature"), true},
		{"too long 73 chars", conventional.Subject(strings.Repeat("a", 73)), true},
		{"way too long", conventional.Subject(strings.Repeat("a", 100)), true},
		{"only spaces", conventional.Subject("   "), true},
		{"only tabs", conventional.Subject("\t\t"), true},
		{"mixed whitespace only", conventional.Subject(" \t \n "), true},
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

func TestSubject_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		desc    conventional.Subject
		want    string
		wantErr bool
	}{
		{"empty", conventional.Subject(""), `""`, false},
		{"simple", conventional.Subject("add user endpoint"), `"add user endpoint"`, false},
		{"with capitals", conventional.Subject("Add Feature"), `"Add Feature"`, false},
		{"with emoji", conventional.Subject("add 🚀"), `"add 🚀"`, false},
		{"invalid newline", conventional.Subject("add\nfeature"), "", true},
		{"invalid too long", conventional.Subject(strings.Repeat("a", 73)), "", true},
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

func TestSubject_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    conventional.Subject
		wantErr bool
	}{
		{"empty", `""`, conventional.Subject(""), false},
		{"simple lowercase", `"add user endpoint"`, conventional.Subject("add user endpoint"), false},
		{"with capitals preserved", `"Add User Endpoint"`, conventional.Subject("Add User Endpoint"), false},
		{"with whitespace trimmed", `"  fix bug  "`, conventional.Subject("fix bug"), false},
		{"with emoji", `"add feature 🚀"`, conventional.Subject("add feature 🚀"), false},
		{"with unicode", `"добавить функцию"`, conventional.Subject("добавить функцию"), false},
		{"invalid newline", `"add\nfeature"`, conventional.Subject(""), true},
		{"invalid too long", `"` + strings.Repeat("a", 73) + `"`, conventional.Subject(""), true},
		{"invalid JSON", `not json`, conventional.Subject(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got conventional.Subject
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

func TestSubject_MarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		desc    conventional.Subject
		want    string
		wantErr bool
	}{
		{"empty", conventional.Subject(""), "\"\"\n", false},
		{"simple", conventional.Subject("add user endpoint"), "add user endpoint\n", false},
		{"with capitals", conventional.Subject("Fix Bug"), "Fix Bug\n", false},
		{"invalid newline", conventional.Subject("add\nfeature"), "", true},
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

func TestSubject_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    conventional.Subject
		wantErr bool
	}{
		{"empty", `""`, conventional.Subject(""), false},
		{"simple", "add user endpoint", conventional.Subject("add user endpoint"), false},
		{"with capitals preserved", "Add User Endpoint", conventional.Subject("Add User Endpoint"), false},
		{"with whitespace", "  fix bug  ", conventional.Subject("fix bug"), false},
		{"with emoji", "add 🚀", conventional.Subject("add 🚀"), false},
		{"invalid too long", strings.Repeat("a", 73), conventional.Subject(""), true},
		// Note: YAML multiline string handling is complex, tested separately in newline validation
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got conventional.Subject
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

func TestParseSubject(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    conventional.Subject
		wantErr bool
	}{
		// Valid inputs
		{"empty", "", conventional.Subject(""), false},
		{"simple", "add user endpoint", conventional.Subject("add user endpoint"), false},
		{"with capitals preserved", "Add User Endpoint", conventional.Subject("Add User Endpoint"), false},
		{"with leading whitespace", "  add feature", conventional.Subject("add feature"), false},
		{"with trailing whitespace", "fix bug  ", conventional.Subject("fix bug"), false},
		{"with surrounding whitespace", "  update docs  ", conventional.Subject("update docs"), false},
		{"with tabs", "\tremove code\t", conventional.Subject("remove code"), false},
		{"only whitespace", "   ", conventional.Subject(""), false},
		{"with emoji", "add feature 🚀", conventional.Subject("add feature 🚀"), false},
		{"with unicode", "добавить функцию", conventional.Subject("добавить функцию"), false},
		{"max length", strings.Repeat("a", 72), conventional.Subject(strings.Repeat("a", 72)), false},

		// Invalid inputs
		{"too long", strings.Repeat("a", 73), conventional.Subject(""), true},
		{"contains LF newline", "add\nfeature", conventional.Subject(""), true},
		{"contains CR newline", "add\rfeature", conventional.Subject(""), true},
		{"contains CRLF", "add\r\nfeature", conventional.Subject(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := conventional.ParseSubject(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSubject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseSubject() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSubject_JSON_RoundTrip(t *testing.T) {
	original := conventional.Subject("add user registration endpoint")

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded conventional.Subject
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded != original {
		t.Errorf("JSON round-trip failed: got %v, want %v", decoded, original)
	}
}

func TestSubject_YAML_RoundTrip(t *testing.T) {
	original := conventional.Subject("fix authentication bug")

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("yaml.Marshal() error = %v", err)
	}

	var decoded conventional.Subject
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	if decoded != original {
		t.Errorf("YAML round-trip failed: got %v, want %v", decoded, original)
	}
}

func TestSubject_LengthConstraints(t *testing.T) {
	// Test minimum length (1 rune, non-empty)
	minDesc := conventional.Subject("x")
	if err := minDesc.Validate(); err != nil {
		t.Errorf("Subject with min length should be valid, got error: %v", err)
	}

	// Test maximum length (72 runes)
	maxDesc := conventional.Subject(strings.Repeat("a", 72))
	if err := maxDesc.Validate(); err != nil {
		t.Errorf("Subject with max length should be valid, got error: %v", err)
	}

	// Test over maximum length (73 runes)
	tooLongDesc := conventional.Subject(strings.Repeat("a", 73))
	if err := tooLongDesc.Validate(); err == nil {
		t.Error("Subject over max length should be invalid")
	}
}

func TestSubject_RuneCounting(t *testing.T) {
	tests := []struct {
		name      string
		desc      conventional.Subject
		runeCount int
		wantErr   bool
	}{
		{"ASCII only", conventional.Subject("add feature"), 11, false},
		{"emoji counts as 1 rune", conventional.Subject("add 🚀"), 5, false},
		{"unicode Cyrillic", conventional.Subject("добавить"), 8, false},
		{"mixed ASCII and emoji", conventional.Subject(strings.Repeat("a", 70) + "🚀"), 71, false},
		{"emoji at max length", conventional.Subject(strings.Repeat("🚀", 72)), 72, false},
		{"emoji exceeds max", conventional.Subject(strings.Repeat("🚀", 73)), 73, true},
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

func TestSubject_NewlineValidation(t *testing.T) {
	tests := []struct {
		name    string
		desc    conventional.Subject
		wantErr bool
	}{
		{"no newline", conventional.Subject("add feature"), false},
		{"LF newline", conventional.Subject("add\nfeature"), true},
		{"CR newline", conventional.Subject("add\rfeature"), true},
		{"CRLF newline", conventional.Subject("add\r\nfeature"), true},
		{"multiple newlines", conventional.Subject("add\n\nfeature"), true},
		{"trailing newline", conventional.Subject("add feature\n"), true},
		{"leading newline", conventional.Subject("\nadd feature"), true},
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

func TestSubject_WhitespaceValidation(t *testing.T) {
	tests := []struct {
		name    string
		desc    conventional.Subject
		wantErr bool
	}{
		{"normal text", conventional.Subject("add feature"), false},
		{"with internal spaces", conventional.Subject("add user endpoint"), false},
		{"only spaces", conventional.Subject("   "), true},
		{"only tabs", conventional.Subject("\t\t"), true},
		{"only newlines", conventional.Subject("\n\n"), true},
		{"mixed whitespace", conventional.Subject(" \t \n "), true},
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

func TestSubject_CasePreservation(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  conventional.Subject
	}{
		{"lowercase", "add feature", conventional.Subject("add feature")},
		{"uppercase", "ADD FEATURE", conventional.Subject("ADD FEATURE")},
		{"mixed case", "Add User Endpoint", conventional.Subject("Add User Endpoint")},
		{"sentence case", "Fix authentication bug", conventional.Subject("Fix authentication bug")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := conventional.ParseSubject(tt.input)
			if err != nil {
				t.Fatalf("ParseSubject() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("ParseSubject() = %v, want %v (case should be preserved)", got, tt.want)
			}
		})
	}
}

func TestSubject_Normalization(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  conventional.Subject
	}{
		{"trims leading space", "  add feature", conventional.Subject("add feature")},
		{"trims trailing space", "add feature  ", conventional.Subject("add feature")},
		{"trims both", "  add feature  ", conventional.Subject("add feature")},
		{"trims tabs", "\tadd feature\t", conventional.Subject("add feature")},
		{"preserves internal spaces", "add user endpoint", conventional.Subject("add user endpoint")},
		{"does not lowercase", "  Add Feature  ", conventional.Subject("Add Feature")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := conventional.ParseSubject(tt.input)
			if err != nil {
				t.Fatalf("ParseSubject() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("ParseSubject() = %v, want %v", got, tt.want)
			}
		})
	}
}
