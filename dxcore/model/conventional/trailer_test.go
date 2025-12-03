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

func TestTrailer_String(t *testing.T) {
	tests := []struct {
		name    string
		trailer conventional.Trailer
		want    string
	}{
		{"empty", conventional.Trailer{}, ""},
		{"with both key and value", conventional.Trailer{Key: "Fixes", Value: "#123"}, "Fixes: #123"},
		{"with key only", conventional.Trailer{Key: "BREAKING-CHANGE", Value: ""}, "BREAKING-CHANGE:"},
		{"co-authored-by", conventional.Trailer{Key: "Co-authored-by", Value: "Jane Doe <jane@example.com>"}, "Co-authored-by: Jane Doe <jane@example.com>"},
		{"signed-off-by", conventional.Trailer{Key: "Signed-off-by", Value: "John Smith <john@example.com>"}, "Signed-off-by: John Smith <john@example.com>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.trailer.String(); got != tt.want {
				t.Errorf("Trailer.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTrailer_Redacted(t *testing.T) {
	trailer := conventional.Trailer{Key: "Co-authored-by", Value: "Jane Doe <jane@example.com>"}
	if got := trailer.Redacted(); got != "Co-authored-by" {
		t.Errorf("Redacted() = %q, want %q", got, "Co-authored-by")
	}

	empty := conventional.Trailer{}
	if got := empty.Redacted(); got != "" {
		t.Errorf("Redacted() for empty trailer = %q, want empty string", got)
	}
}

func TestTrailer_TypeName(t *testing.T) {
	trailer := conventional.Trailer{Key: "Fixes", Value: "#123"}
	if got := trailer.TypeName(); got != "Trailer" {
		t.Errorf("TypeName() = %q, want %q", got, "Trailer")
	}
}

func TestTrailer_IsZero(t *testing.T) {
	tests := []struct {
		name    string
		trailer conventional.Trailer
		want    bool
	}{
		{"empty is zero", conventional.Trailer{}, true},
		{"key only not zero", conventional.Trailer{Key: "Fixes", Value: ""}, false},
		{"value only not zero", conventional.Trailer{Key: "", Value: "#123"}, false},
		{"both fields not zero", conventional.Trailer{Key: "Fixes", Value: "#123"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.trailer.IsZero(); got != tt.want {
				t.Errorf("IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTrailer_Equal(t *testing.T) {
	tests := []struct {
		name string
		tr1  conventional.Trailer
		tr2  conventional.Trailer
		want bool
	}{
		{"both empty", conventional.Trailer{}, conventional.Trailer{}, true},
		{"same key and value", conventional.Trailer{Key: "Fixes", Value: "#123"}, conventional.Trailer{Key: "Fixes", Value: "#123"}, true},
		{"different key", conventional.Trailer{Key: "Fixes", Value: "#123"}, conventional.Trailer{Key: "Closes", Value: "#123"}, false},
		{"different value", conventional.Trailer{Key: "Fixes", Value: "#123"}, conventional.Trailer{Key: "Fixes", Value: "#456"}, false},
		{"different case in key", conventional.Trailer{Key: "Fixes", Value: "#123"}, conventional.Trailer{Key: "fixes", Value: "#123"}, false},
		{"key only vs empty", conventional.Trailer{Key: "Fixes", Value: ""}, conventional.Trailer{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tr1.Equal(tt.tr2); got != tt.want {
				t.Errorf("Trailer.Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTrailer_Validate(t *testing.T) {
	tests := []struct {
		name    string
		trailer conventional.Trailer
		wantErr bool
	}{
		// Valid trailers
		{"empty valid", conventional.Trailer{}, false},
		{"fixes valid", conventional.Trailer{Key: "Fixes", Value: "#123"}, false},
		{"co-authored-by valid", conventional.Trailer{Key: "Co-authored-by", Value: "Jane Doe <jane@example.com>"}, false},
		{"signed-off-by valid", conventional.Trailer{Key: "Signed-off-by", Value: "John Smith <john@example.com>"}, false},
		{"breaking-change valid", conventional.Trailer{Key: "BREAKING-CHANGE", Value: "removed legacy API"}, false},
		{"key only valid", conventional.Trailer{Key: "Acked-by", Value: ""}, false},
		{"single letter key", conventional.Trailer{Key: "A", Value: "test"}, false},
		{"key with digits", conventional.Trailer{Key: "Foo123", Value: "bar"}, false},
		{"key with hyphens", conventional.Trailer{Key: "Reviewed-by-Team", Value: "Alice"}, false},

		// Invalid trailers
		{"empty key", conventional.Trailer{Key: "", Value: "#123"}, true},
		{"key with colon", conventional.Trailer{Key: "Fixes:", Value: "#123"}, true},
		{"key starting with digit", conventional.Trailer{Key: "123-Fixes", Value: "#123"}, true},
		{"key starting with hyphen", conventional.Trailer{Key: "-Fixes", Value: "#123"}, true},
		{"key with space", conventional.Trailer{Key: "Co authored by", Value: "Jane"}, true},
		{"key with special char", conventional.Trailer{Key: "Fixes@", Value: "#123"}, true},
		{"key too long", conventional.Trailer{Key: strings.Repeat("a", 65), Value: "test"}, true},
		{"value with newline", conventional.Trailer{Key: "Fixes", Value: "#123\n#456"}, true},
		{"value with CRLF", conventional.Trailer{Key: "Fixes", Value: "#123\r\n#456"}, true},
		{"value too long", conventional.Trailer{Key: "Fixes", Value: strings.Repeat("a", 257)}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.trailer.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTrailer_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		trailer conventional.Trailer
		wantErr bool
	}{
		{"empty", conventional.Trailer{}, false},
		{"fixes", conventional.Trailer{Key: "Fixes", Value: "#123"}, false},
		{"co-authored-by", conventional.Trailer{Key: "Co-authored-by", Value: "Jane Doe <jane@example.com>"}, false},
		{"key only", conventional.Trailer{Key: "BREAKING-CHANGE", Value: ""}, false},
		{"invalid key with colon", conventional.Trailer{Key: "Fixes:", Value: "#123"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.trailer)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Verify it's valid JSON and can be unmarshaled
				var decoded conventional.Trailer
				if err := json.Unmarshal(got, &decoded); err != nil {
					t.Errorf("MarshalJSON() produced invalid JSON: %v", err)
				}
				if !decoded.Equal(tt.trailer) {
					t.Errorf("MarshalJSON() round-trip failed: got %+v, want %+v", decoded, tt.trailer)
				}
			}
		})
	}
}

func TestTrailer_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    conventional.Trailer
		wantErr bool
	}{
		{"empty", `{"key":"","value":""}`, conventional.Trailer{}, false},
		{"fixes", `{"key":"Fixes","value":"#123"}`, conventional.Trailer{Key: "Fixes", Value: "#123"}, false},
		{"co-authored-by", `{"key":"Co-authored-by","value":"Jane Doe <jane@example.com>"}`, conventional.Trailer{Key: "Co-authored-by", Value: "Jane Doe <jane@example.com>"}, false},
		{"with whitespace trimmed", `{"key":"  Fixes  ","value":"  #123  "}`, conventional.Trailer{Key: "Fixes", Value: "#123"}, false},
		{"key only", `{"key":"BREAKING-CHANGE","value":""}`, conventional.Trailer{Key: "BREAKING-CHANGE", Value: ""}, false},
		{"invalid key with colon", `{"key":"Fixes:","value":"#123"}`, conventional.Trailer{}, true},
		{"invalid key starting with digit", `{"key":"123-Fixes","value":"#123"}`, conventional.Trailer{}, true},
		{"invalid JSON", `not json`, conventional.Trailer{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got conventional.Trailer
			err := json.Unmarshal([]byte(tt.data), &got)
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

func TestTrailer_MarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		trailer conventional.Trailer
		wantErr bool
	}{
		{"empty", conventional.Trailer{}, false},
		{"fixes", conventional.Trailer{Key: "Fixes", Value: "#123"}, false},
		{"co-authored-by", conventional.Trailer{Key: "Co-authored-by", Value: "Jane Doe <jane@example.com>"}, false},
		{"invalid key with colon", conventional.Trailer{Key: "Fixes:", Value: "#123"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := yaml.Marshal(tt.trailer)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTrailer_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    conventional.Trailer
		wantErr bool
	}{
		{"empty", "key: \"\"\nvalue: \"\"", conventional.Trailer{}, false},
		{"fixes", "key: Fixes\nvalue: '#123'", conventional.Trailer{Key: "Fixes", Value: "#123"}, false},
		{"co-authored-by", "key: Co-authored-by\nvalue: Jane Doe <jane@example.com>", conventional.Trailer{Key: "Co-authored-by", Value: "Jane Doe <jane@example.com>"}, false},
		{"with whitespace trimmed", "key: '  Fixes  '\nvalue: '  #123  '", conventional.Trailer{Key: "Fixes", Value: "#123"}, false},
		{"invalid key with colon", "key: 'Fixes:'\nvalue: '#123'", conventional.Trailer{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got conventional.Trailer
			err := yaml.Unmarshal([]byte(tt.data), &got)
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

func TestParseTrailer(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    conventional.Trailer
		wantErr bool
	}{
		// Valid inputs
		{"fixes", "Fixes: #123", conventional.Trailer{Key: "Fixes", Value: "#123"}, false},
		{"fixes without space", "Fixes:#123", conventional.Trailer{Key: "Fixes", Value: "#123"}, false},
		{"co-authored-by", "Co-authored-by: Jane Doe <jane@example.com>", conventional.Trailer{Key: "Co-authored-by", Value: "Jane Doe <jane@example.com>"}, false},
		{"signed-off-by", "Signed-off-by: John Smith <john@example.com>", conventional.Trailer{Key: "Signed-off-by", Value: "John Smith <john@example.com>"}, false},
		{"breaking-change", "BREAKING-CHANGE: removed legacy API", conventional.Trailer{Key: "BREAKING-CHANGE", Value: "removed legacy API"}, false},
		{"with leading whitespace", "  Fixes: #123  ", conventional.Trailer{Key: "Fixes", Value: "#123"}, false},
		{"with trailing whitespace", "Fixes: #123  ", conventional.Trailer{Key: "Fixes", Value: "#123"}, false},
		{"key only", "BREAKING-CHANGE:", conventional.Trailer{Key: "BREAKING-CHANGE", Value: ""}, false},
		{"multiple colons in value", "Refs: https://example.com/issue:123", conventional.Trailer{Key: "Refs", Value: "https://example.com/issue:123"}, false},

		// Invalid inputs
		{"empty string", "", conventional.Trailer{}, true},
		{"whitespace only", "   ", conventional.Trailer{}, true},
		{"no colon", "Fixes #123", conventional.Trailer{}, true},
		{"key starting with digit", "123-Fixes: #123", conventional.Trailer{}, true},
		{"key starting with hyphen", "-Fixes: #123", conventional.Trailer{}, true},
		{"key with space", "Co authored by: Jane", conventional.Trailer{}, true},
		{"key with special char", "Fixes@: #123", conventional.Trailer{}, true},
		{"value with newline", "Fixes: #123\n#456", conventional.Trailer{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := conventional.ParseTrailer(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTrailer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.want) {
				t.Errorf("ParseTrailer() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestTrailer_JSON_RoundTrip(t *testing.T) {
	original := conventional.Trailer{Key: "Co-authored-by", Value: "Jane Doe <jane@example.com>"}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded conventional.Trailer
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if !decoded.Equal(original) {
		t.Errorf("JSON round-trip failed: got %+v, want %+v", decoded, original)
	}
}

func TestTrailer_YAML_RoundTrip(t *testing.T) {
	original := conventional.Trailer{Key: "Signed-off-by", Value: "John Smith <john@example.com>"}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("yaml.Marshal() error = %v", err)
	}

	var decoded conventional.Trailer
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	if !decoded.Equal(original) {
		t.Errorf("YAML round-trip failed: got %+v, want %+v", decoded, original)
	}
}

func TestTrailer_KeyValidation(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"single letter", "A", false},
		{"lowercase", "fixes", false},
		{"uppercase", "FIXES", false},
		{"mixed case", "Co-authored-by", false},
		{"with hyphens", "Reviewed-by-Team", false},
		{"with digits", "Foo123", false},
		{"starts with letter", "F123", false},
		{"max length", strings.Repeat("a", 64), false},

		{"empty", "", true},
		{"starts with digit", "1Fixes", true},
		{"starts with hyphen", "-Fixes", true},
		{"contains space", "Foo Bar", true},
		{"contains colon", "Foo:", true},
		{"contains special char", "Foo@Bar", true},
		{"contains underscore", "Foo_Bar", true},
		{"contains dot", "Foo.Bar", true},
		{"too long", strings.Repeat("a", 65), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trailer := conventional.Trailer{Key: tt.key, Value: "test"}
			err := trailer.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() for key %q error = %v, wantErr %v", tt.key, err, tt.wantErr)
			}
		})
	}
}

func TestTrailer_ValueValidation(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty", "", false},
		{"short", "#123", false},
		{"with email", "Jane Doe <jane@example.com>", false},
		{"with url", "https://example.com/issue/123", false},
		{"max length", strings.Repeat("a", 256), false},

		{"with LF newline", "#123\n#456", true},
		{"with CR newline", "#123\r#456", true},
		{"with CRLF newline", "#123\r\n#456", true},
		{"too long", strings.Repeat("a", 257), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trailer := conventional.Trailer{Key: "Fixes", Value: tt.value}
			err := trailer.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() for value %q error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestTrailer_CommonGitTrailers(t *testing.T) {
	// Test common git trailer types
	tests := []struct {
		name    string
		trailer conventional.Trailer
	}{
		{"Co-authored-by", conventional.Trailer{Key: "Co-authored-by", Value: "Jane Doe <jane@example.com>"}},
		{"Signed-off-by", conventional.Trailer{Key: "Signed-off-by", Value: "John Smith <john@example.com>"}},
		{"Reviewed-by", conventional.Trailer{Key: "Reviewed-by", Value: "Alice Example <alice@example.com>"}},
		{"Acked-by", conventional.Trailer{Key: "Acked-by", Value: "Bob Builder <bob@example.com>"}},
		{"Reported-by", conventional.Trailer{Key: "Reported-by", Value: "Charlie Tester <charlie@example.com>"}},
		{"Fixes", conventional.Trailer{Key: "Fixes", Value: "#123"}},
		{"Closes", conventional.Trailer{Key: "Closes", Value: "#456"}},
		{"Refs", conventional.Trailer{Key: "Refs", Value: "#789"}},
		{"BREAKING-CHANGE", conventional.Trailer{Key: "BREAKING-CHANGE", Value: "removed legacy API"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.trailer.Validate(); err != nil {
				t.Errorf("Validate() for common trailer %q error = %v", tt.name, err)
			}

			// Test round-trip
			str := tt.trailer.String()
			parsed, err := conventional.ParseTrailer(str)
			if err != nil {
				t.Errorf("ParseTrailer() for %q error = %v", str, err)
			}
			if !parsed.Equal(tt.trailer) {
				t.Errorf("Round-trip failed for %q: got %+v, want %+v", tt.name, parsed, tt.trailer)
			}
		})
	}
}
