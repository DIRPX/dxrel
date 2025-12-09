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

package change

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestBump_String(t *testing.T) {
	tests := []struct {
		name string
		bump Bump
		want string
	}{
		{"BumpNone", BumpNone, "none"},
		{"BumpPatch", BumpPatch, "patch"},
		{"BumpMinor", BumpMinor, "minor"},
		{"BumpMajor", BumpMajor, "major"},
		{"Unknown", Bump(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.bump.String(); got != tt.want {
				t.Errorf("Bump.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseBump(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Bump
		wantErr bool
	}{
		// Valid inputs
		{"none lowercase", "none", BumpNone, false},
		{"none title", "None", BumpNone, false},
		{"none uppercase", "NONE", BumpNone, false},
		{"patch lowercase", "patch", BumpPatch, false},
		{"patch title", "Patch", BumpPatch, false},
		{"patch uppercase", "PATCH", BumpPatch, false},
		{"minor lowercase", "minor", BumpMinor, false},
		{"minor title", "Minor", BumpMinor, false},
		{"minor uppercase", "MINOR", BumpMinor, false},
		{"major lowercase", "major", BumpMajor, false},
		{"major title", "Major", BumpMajor, false},
		{"major uppercase", "MAJOR", BumpMajor, false},

		// Invalid inputs
		{"empty", "", BumpNone, true},
		{"invalid", "invalid", BumpNone, true},
		{"number", "1", BumpNone, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseBump(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseBump() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseBump() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBump_Valid(t *testing.T) {
	tests := []struct {
		name string
		bump Bump
		want bool
	}{
		{"BumpNone", BumpNone, true},
		{"BumpPatch", BumpPatch, true},
		{"BumpMinor", BumpMinor, true},
		{"BumpMajor", BumpMajor, true},
		{"Invalid negative", Bump(-1), false},
		{"Invalid positive", Bump(99), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.bump.Valid(); got != tt.want {
				t.Errorf("Bump.Valid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBump_TypeName(t *testing.T) {
	var b Bump
	if got := b.TypeName(); got != "Bump" {
		t.Errorf("TypeName() = %v, want Bump", got)
	}
}

func TestBump_Redacted(t *testing.T) {
	tests := []struct {
		name string
		bump Bump
		want string
	}{
		{"BumpNone", BumpNone, "none"},
		{"BumpPatch", BumpPatch, "patch"},
		{"BumpMinor", BumpMinor, "minor"},
		{"BumpMajor", BumpMajor, "major"},
		{"Unknown", Bump(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.bump.Redacted(); got != tt.want {
				t.Errorf("Redacted() = %v, want %v", got, tt.want)
			}
			// Redacted should match String for Bump
			if got := tt.bump.Redacted(); got != tt.bump.String() {
				t.Errorf("Redacted() = %v, String() = %v (should match)", got, tt.bump.String())
			}
		})
	}
}

func TestBump_IsZero(t *testing.T) {
	tests := []struct {
		name string
		bump Bump
		want bool
	}{
		{"BumpNone (zero value)", BumpNone, true},
		{"BumpPatch", BumpPatch, false},
		{"BumpMinor", BumpMinor, false},
		{"BumpMajor", BumpMajor, false},
		{"Invalid", Bump(99), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.bump.IsZero(); got != tt.want {
				t.Errorf("IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBump_Equal(t *testing.T) {
	tests := []struct {
		name string
		b1   Bump
		b2   any
		want bool
	}{
		{"equal BumpNone", BumpNone, BumpNone, true},
		{"equal BumpPatch", BumpPatch, BumpPatch, true},
		{"equal BumpMinor", BumpMinor, BumpMinor, true},
		{"equal BumpMajor", BumpMajor, BumpMajor, true},
		{"different values", BumpNone, BumpPatch, false},
		{"pointer equal", BumpNone, func() *Bump { b := BumpNone; return &b }(), true},
		{"pointer different", BumpNone, func() *Bump { b := BumpPatch; return &b }(), false},
		{"nil pointer", BumpNone, (*Bump)(nil), false},
		{"different type", BumpNone, "none", false},
		{"different type int", BumpNone, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.b1.Equal(tt.b2); got != tt.want {
				t.Errorf("Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBump_Validate(t *testing.T) {
	tests := []struct {
		name    string
		bump    Bump
		wantErr bool
	}{
		{"BumpNone valid", BumpNone, false},
		{"BumpPatch valid", BumpPatch, false},
		{"BumpMinor valid", BumpMinor, false},
		{"BumpMajor valid", BumpMajor, false},
		{"Invalid negative", Bump(-1), true},
		{"Invalid positive", Bump(99), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.bump.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBump_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		bump    Bump
		want    string
		wantErr bool
	}{
		{"BumpNone", BumpNone, `"none"`, false},
		{"BumpPatch", BumpPatch, `"patch"`, false},
		{"BumpMinor", BumpMinor, `"minor"`, false},
		{"BumpMajor", BumpMajor, `"major"`, false},
		{"Invalid", Bump(99), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.bump)
			if (err != nil) != tt.wantErr {
				t.Errorf("Bump.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(got) != tt.want {
				t.Errorf("Bump.MarshalJSON() = %v, want %v", string(got), tt.want)
			}
		})
	}
}

func TestBump_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Bump
		wantErr bool
	}{
		// String format
		{"none string", `"none"`, BumpNone, false},
		{"patch string", `"patch"`, BumpPatch, false},
		{"minor string", `"minor"`, BumpMinor, false},
		{"major string", `"major"`, BumpMajor, false},

		// Numeric format
		{"none numeric", `0`, BumpNone, false},
		{"patch numeric", `1`, BumpPatch, false},
		{"minor numeric", `2`, BumpMinor, false},
		{"major numeric", `3`, BumpMajor, false},

		// Invalid inputs
		{"empty", `""`, BumpNone, true},
		{"invalid string", `"invalid"`, BumpNone, true},
		{"invalid number", `99`, BumpNone, true},
		{"empty data", ``, BumpNone, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Bump
			err := json.Unmarshal([]byte(tt.input), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("Bump.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Bump.UnmarshalJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBump_MarshalText(t *testing.T) {
	tests := []struct {
		name    string
		bump    Bump
		want    string
		wantErr bool
	}{
		{"BumpNone", BumpNone, "none", false},
		{"BumpPatch", BumpPatch, "patch", false},
		{"BumpMinor", BumpMinor, "minor", false},
		{"BumpMajor", BumpMajor, "major", false},
		{"Invalid", Bump(99), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.bump.MarshalText()
			if (err != nil) != tt.wantErr {
				t.Errorf("Bump.MarshalText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(got) != tt.want {
				t.Errorf("Bump.MarshalText() = %v, want %v", string(got), tt.want)
			}
		})
	}
}

func TestBump_UnmarshalText(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Bump
		wantErr bool
	}{
		{"none", "none", BumpNone, false},
		{"patch", "patch", BumpPatch, false},
		{"minor", "minor", BumpMinor, false},
		{"major", "major", BumpMajor, false},
		{"invalid", "invalid", BumpNone, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Bump
			err := got.UnmarshalText([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("Bump.UnmarshalText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Bump.UnmarshalText() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBump_YAML(t *testing.T) {
	tests := []struct {
		name string
		bump Bump
		want string
	}{
		{"BumpNone", BumpNone, "none\n"},
		{"BumpPatch", BumpPatch, "patch\n"},
		{"BumpMinor", BumpMinor, "minor\n"},
		{"BumpMajor", BumpMajor, "major\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			got, err := yaml.Marshal(tt.bump)
			if err != nil {
				t.Errorf("yaml.Marshal() error = %v", err)
				return
			}
			if string(got) != tt.want {
				t.Errorf("yaml.Marshal() = %v, want %v", string(got), tt.want)
			}

			// Unmarshal
			var bump Bump
			if err := yaml.Unmarshal(got, &bump); err != nil {
				t.Errorf("yaml.Unmarshal() error = %v", err)
				return
			}
			if bump != tt.bump {
				t.Errorf("yaml.Unmarshal() = %v, want %v", bump, tt.bump)
			}
		})
	}
}

func TestBump_RoundTrip(t *testing.T) {
	tests := []Bump{BumpNone, BumpPatch, BumpMinor, BumpMajor}

	for _, original := range tests {
		t.Run(original.String(), func(t *testing.T) {
			// JSON round-trip
			jsonData, err := json.Marshal(original)
			if err != nil {
				t.Fatalf("JSON Marshal error: %v", err)
			}
			var jsonResult Bump
			if err := json.Unmarshal(jsonData, &jsonResult); err != nil {
				t.Fatalf("JSON Unmarshal error: %v", err)
			}
			if jsonResult != original {
				t.Errorf("JSON round-trip: got %v, want %v", jsonResult, original)
			}

			// YAML round-trip
			yamlData, err := yaml.Marshal(original)
			if err != nil {
				t.Fatalf("YAML Marshal error: %v", err)
			}
			var yamlResult Bump
			if err := yaml.Unmarshal(yamlData, &yamlResult); err != nil {
				t.Fatalf("YAML Unmarshal error: %v", err)
			}
			if yamlResult != original {
				t.Errorf("YAML round-trip: got %v, want %v", yamlResult, original)
			}
		})
	}
}

func TestBump_MarshalJSON_Invalid(t *testing.T) {
	// Invalid Bump should fail to marshal
	invalid := Bump(99)
	_, err := json.Marshal(invalid)
	if err == nil {
		t.Error("Expected error marshaling invalid Bump, got nil")
	}
}

func TestBump_MarshalYAML_Invalid(t *testing.T) {
	// Invalid Bump should fail to marshal as YAML
	invalid := Bump(99)
	_, err := yaml.Marshal(invalid)
	if err == nil {
		t.Error("Expected error marshaling invalid Bump as YAML, got nil")
	}
}

func TestBump_MarshalText_Invalid(t *testing.T) {
	// Invalid Bump should fail to marshal as text
	invalid := Bump(99)
	_, err := invalid.MarshalText()
	if err == nil {
		t.Error("Expected error marshaling invalid Bump as text, got nil")
	}
}
