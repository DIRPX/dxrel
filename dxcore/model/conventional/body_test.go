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

func TestBody_String(t *testing.T) {
	tests := []struct {
		name string
		body conventional.Body
		want string
	}{
		{"empty", conventional.Body(""), ""},
		{"single line", conventional.Body("Fix authentication bug"), "Fix authentication bug"},
		{"multi-line", conventional.Body("Line 1\nLine 2"), "Line 1\nLine 2"},
		{"with blank lines", conventional.Body("Para 1\n\nPara 2"), "Para 1\n\nPara 2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.body.String(); got != tt.want {
				t.Errorf("Body.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBody_Redacted(t *testing.T) {
	// Redacted should be identical to String for Body
	body := conventional.Body("Sensitive content\nMultiple lines")
	if body.Redacted() != body.String() {
		t.Errorf("Redacted() = %q, want %q", body.Redacted(), body.String())
	}
}

func TestBody_TypeName(t *testing.T) {
	body := conventional.Body("test")
	if got := body.TypeName(); got != "Body" {
		t.Errorf("TypeName() = %q, want %q", got, "Body")
	}
}

func TestBody_IsZero(t *testing.T) {
	tests := []struct {
		name string
		body conventional.Body
		want bool
	}{
		{"empty is zero", conventional.Body(""), true},
		{"non-empty is not zero", conventional.Body("content"), false},
		{"single line is not zero", conventional.Body("x"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.body.IsZero(); got != tt.want {
				t.Errorf("IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBody_Validate(t *testing.T) {
	tests := []struct {
		name    string
		body    conventional.Body
		wantErr bool
	}{
		// Valid bodies
		{"empty valid", conventional.Body(""), false},
		{"single line", conventional.Body("Fix bug"), false},
		{"multi-line", conventional.Body("Line 1\nLine 2\nLine 3"), false},
		{"with blank lines", conventional.Body("Para 1\n\nPara 2"), false},
		{"max bytes", conventional.Body(strings.Repeat("a", 8*1024)), false},
		{"max lines", conventional.Body(strings.Repeat("line\n", 99) + "line"), false},

		// Invalid bodies
		{"contains CR", conventional.Body("line\rtest"), true},
		{"contains CRLF", conventional.Body("line\r\ntest"), true},
		{"exceeds byte limit", conventional.Body(strings.Repeat("a", 8*1024+1)), true},
		{"exceeds line limit", conventional.Body(strings.Repeat("line\n", 101)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.body.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBody_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		body    conventional.Body
		want    string
		wantErr bool
	}{
		{"empty", conventional.Body(""), `""`, false},
		{"single line", conventional.Body("Fix bug"), `"Fix bug"`, false},
		{"multi-line", conventional.Body("Line 1\nLine 2"), `"Line 1\nLine 2"`, false},
		{"invalid CR", conventional.Body("bad\rline"), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.body)
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

func TestBody_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    conventional.Body
		wantErr bool
	}{
		{"empty", `""`, conventional.Body(""), false},
		{"single line", `"Fix bug"`, conventional.Body("Fix bug"), false},
		{"multi-line", `"Line 1\nLine 2"`, conventional.Body("Line 1\nLine 2"), false},
		{"with CRLF normalized", `"Line 1\r\nLine 2"`, conventional.Body("Line 1\nLine 2"), false},
		{"with leading blanks", `"\n\nContent"`, conventional.Body("Content"), false},
		{"with trailing blanks", `"Content\n\n"`, conventional.Body("Content"), false},
		{"whitespace only", `"   \n   "`, conventional.Body(""), false},
		{"too large", `"` + strings.Repeat("a", 8*1024+1) + `"`, conventional.Body(""), true},
		{"invalid JSON", `not json`, conventional.Body(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got conventional.Body
			err := json.Unmarshal([]byte(tt.data), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("UnmarshalJSON() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBody_MarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		body    conventional.Body
		wantErr bool
	}{
		{"empty", conventional.Body(""), false},
		{"single line", conventional.Body("Fix bug"), false},
		{"multi-line", conventional.Body("Line 1\nLine 2"), false},
		{"invalid CR", conventional.Body("bad\rline"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := yaml.Marshal(tt.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBody_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    conventional.Body
		wantErr bool
	}{
		{"empty", `""`, conventional.Body(""), false},
		{"single line", "Fix bug", conventional.Body("Fix bug"), false},
		{"quoted multi-line", `"Line 1\nLine 2"`, conventional.Body("Line 1\nLine 2"), false},
		{"whitespace trimmed", "  Content  ", conventional.Body("Content"), false},
		{"too large", strings.Repeat("a", 8*1024+1), conventional.Body(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got conventional.Body
			err := yaml.Unmarshal([]byte(tt.data), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("UnmarshalYAML() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseBody(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    conventional.Body
		wantErr bool
	}{
		// Valid inputs
		{"empty", "", conventional.Body(""), false},
		{"single line", "Fix bug", conventional.Body("Fix bug"), false},
		{"multi-line", "Line 1\nLine 2", conventional.Body("Line 1\nLine 2"), false},
		{"CRLF normalized", "Line 1\r\nLine 2", conventional.Body("Line 1\nLine 2"), false},
		{"lone CR removed", "Line 1\rLine 2", conventional.Body("Line 1Line 2"), false},
		{"leading blanks trimmed", "\n\nContent", conventional.Body("Content"), false},
		{"trailing blanks trimmed", "Content\n\n", conventional.Body("Content"), false},
		{"surrounding blanks trimmed", "\n\nContent\n\n", conventional.Body("Content"), false},
		{"internal blanks preserved", "Para 1\n\nPara 2", conventional.Body("Para 1\n\nPara 2"), false},
		{"whitespace only", "   \n   ", conventional.Body(""), false},
		{"tabs and spaces", "\t\n  \n\t", conventional.Body(""), false},

		// Invalid inputs
		{"too large", strings.Repeat("a", 8*1024+1), conventional.Body(""), true},
		{"too many lines", strings.Repeat("line\n", 101), conventional.Body(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := conventional.ParseBody(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseBody() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseBody() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBody_JSON_RoundTrip(t *testing.T) {
	original := conventional.Body("Fix authentication bug\n\nDetailed explanation here.")

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded conventional.Body
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded != original {
		t.Errorf("JSON round-trip failed: got %q, want %q", decoded, original)
	}
}

func TestBody_YAML_RoundTrip(t *testing.T) {
	original := conventional.Body("Improve performance\n\nBenchmarks show 2x speedup.")

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("yaml.Marshal() error = %v", err)
	}

	var decoded conventional.Body
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	if decoded != original {
		t.Errorf("YAML round-trip failed: got %q, want %q", decoded, original)
	}
}

func TestBody_ByteLimit(t *testing.T) {
	// Test exactly at limit
	atLimit := conventional.Body(strings.Repeat("a", 8*1024))
	if err := atLimit.Validate(); err != nil {
		t.Errorf("Body at byte limit should be valid, got error: %v", err)
	}

	// Test over limit
	overLimit := conventional.Body(strings.Repeat("a", 8*1024+1))
	if err := overLimit.Validate(); err == nil {
		t.Error("Body over byte limit should be invalid")
	}
}

func TestBody_LineLimit(t *testing.T) {
	// Test exactly at limit (100 lines)
	atLimit := conventional.Body(strings.Repeat("line\n", 99) + "line")
	if err := atLimit.Validate(); err != nil {
		t.Errorf("Body at line limit should be valid, got error: %v", err)
	}

	// Test over limit
	overLimit := conventional.Body(strings.Repeat("line\n", 101))
	if err := overLimit.Validate(); err == nil {
		t.Error("Body over line limit should be invalid")
	}
}

func TestBody_BlankLineTrimming(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  conventional.Body
	}{
		{"no blanks", "Content", conventional.Body("Content")},
		{"leading blank", "\nContent", conventional.Body("Content")},
		{"trailing blank", "Content\n", conventional.Body("Content")},
		{"both blanks", "\nContent\n", conventional.Body("Content")},
		{"multiple leading", "\n\n\nContent", conventional.Body("Content")},
		{"multiple trailing", "Content\n\n\n", conventional.Body("Content")},
		{"internal preserved", "Para 1\n\nPara 2", conventional.Body("Para 1\n\nPara 2")},
		{"complex", "\n\nPara 1\n\nPara 2\n\n", conventional.Body("Para 1\n\nPara 2")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := conventional.ParseBody(tt.input)
			if err != nil {
				t.Fatalf("ParseBody() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("ParseBody() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBody_LineEndingNormalization(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  conventional.Body
	}{
		{"LF preserved", "Line 1\nLine 2", conventional.Body("Line 1\nLine 2")},
		{"CRLF to LF", "Line 1\r\nLine 2", conventional.Body("Line 1\nLine 2")},
		{"lone CR removed", "Line 1\rLine 2", conventional.Body("Line 1Line 2")},
		{"mixed endings", "A\nB\r\nC\rD", conventional.Body("A\nB\nCD")},
		{"trailing CRLF", "Content\r\n", conventional.Body("Content")},
		{"leading CRLF", "\r\nContent", conventional.Body("Content")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := conventional.ParseBody(tt.input)
			if err != nil {
				t.Fatalf("ParseBody() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("ParseBody() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBody_MultiLinePreservation(t *testing.T) {
	multiLine := `First paragraph with details.

Second paragraph explaining more.

Third paragraph with conclusion.`

	body, err := conventional.ParseBody(multiLine)
	if err != nil {
		t.Fatalf("ParseBody() error = %v", err)
	}

	// Verify structure is preserved
	lines := strings.Split(body.String(), "\n")
	if len(lines) != 5 {
		t.Errorf("Expected 5 lines (3 content + 2 blank), got %d", len(lines))
	}

	// Verify blank lines are in correct positions
	if lines[1] != "" || lines[3] != "" {
		t.Error("Internal blank lines not preserved correctly")
	}

	// Verify content lines
	if !strings.Contains(lines[0], "First") {
		t.Error("First paragraph not preserved")
	}
	if !strings.Contains(lines[2], "Second") {
		t.Error("Second paragraph not preserved")
	}
	if !strings.Contains(lines[4], "Third") {
		t.Error("Third paragraph not preserved")
	}
}
