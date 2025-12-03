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
	"testing"

	"dirpx.dev/dxrel/dxcore/model/conventional"
	"gopkg.in/yaml.v3"
)

func TestType_String(t *testing.T) {
	tests := []struct {
		name string
		typ  conventional.Type
		want string
	}{
		{"Feat", conventional.Feat, "feat"},
		{"Fix", conventional.Fix, "fix"},
		{"Docs", conventional.Docs, "docs"},
		{"Style", conventional.Style, "style"},
		{"Refactor", conventional.Refactor, "refactor"},
		{"Perf", conventional.Perf, "perf"},
		{"Test", conventional.Test, "test"},
		{"Build", conventional.Build, "build"},
		{"CI", conventional.CI, "ci"},
		{"Chore", conventional.Chore, "chore"},
		{"Revert", conventional.Revert, "revert"},
		{"Invalid", conventional.Type(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.typ.String(); got != tt.want {
				t.Errorf("Type.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestType_Redacted(t *testing.T) {
	// Redacted should be identical to String for Type
	typ := conventional.Feat
	if typ.Redacted() != typ.String() {
		t.Errorf("Redacted() = %q, want %q", typ.Redacted(), typ.String())
	}
}

func TestType_TypeName(t *testing.T) {
	typ := conventional.Feat
	if got := typ.TypeName(); got != "Type" {
		t.Errorf("TypeName() = %q, want %q", got, "Type")
	}
}

func TestType_IsZero(t *testing.T) {
	// Type never returns true for IsZero because zero value (Feat) is valid
	var typ conventional.Type
	if typ.IsZero() {
		t.Error("IsZero() = true, want false (zero value is Feat, which is valid)")
	}

	typ = conventional.Fix
	if typ.IsZero() {
		t.Error("IsZero() = true, want false")
	}
}

func TestType_Equal(t *testing.T) {
	tests := []struct {
		name  string
		t1    conventional.Type
		t2    conventional.Type
		want  bool
	}{
		{"same feat", conventional.Feat, conventional.Feat, true},
		{"same fix", conventional.Fix, conventional.Fix, true},
		{"same docs", conventional.Docs, conventional.Docs, true},
		{"different feat vs fix", conventional.Feat, conventional.Fix, false},
		{"different fix vs docs", conventional.Fix, conventional.Docs, false},
		{"different feat vs revert", conventional.Feat, conventional.Revert, false},
		{"zero value equals feat", conventional.Type(0), conventional.Feat, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.t1.Equal(tt.t2); got != tt.want {
				t.Errorf("Type.Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestType_Validate(t *testing.T) {
	tests := []struct {
		name    string
		typ     conventional.Type
		wantErr bool
	}{
		{"Feat valid", conventional.Feat, false},
		{"Fix valid", conventional.Fix, false},
		{"Docs valid", conventional.Docs, false},
		{"Style valid", conventional.Style, false},
		{"Refactor valid", conventional.Refactor, false},
		{"Perf valid", conventional.Perf, false},
		{"Test valid", conventional.Test, false},
		{"Build valid", conventional.Build, false},
		{"CI valid", conventional.CI, false},
		{"Chore valid", conventional.Chore, false},
		{"Revert valid", conventional.Revert, false},
		{"Invalid out of range", conventional.Type(99), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.typ.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestType_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		typ     conventional.Type
		want    string
		wantErr bool
	}{
		{"Feat", conventional.Feat, `"feat"`, false},
		{"Fix", conventional.Fix, `"fix"`, false},
		{"Docs", conventional.Docs, `"docs"`, false},
		{"Invalid", conventional.Type(99), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.typ)
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

func TestType_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    conventional.Type
		wantErr bool
	}{
		{"lowercase feat", `"feat"`, conventional.Feat, false},
		{"uppercase FEAT", `"FEAT"`, conventional.Feat, false},
		{"mixed case Fix", `"FiX"`, conventional.Fix, false},
		{"lowercase docs", `"docs"`, conventional.Docs, false},
		{"unknown type", `"unknown"`, conventional.Type(0), true},
		{"invalid JSON", `not json`, conventional.Type(0), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got conventional.Type
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

func TestType_MarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		typ     conventional.Type
		want    string
		wantErr bool
	}{
		{"Feat", conventional.Feat, "feat\n", false},
		{"Fix", conventional.Fix, "fix\n", false},
		{"Perf", conventional.Perf, "perf\n", false},
		{"Invalid", conventional.Type(99), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := yaml.Marshal(tt.typ)
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

func TestType_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    conventional.Type
		wantErr bool
	}{
		{"lowercase test", "test", conventional.Test, false},
		{"uppercase BUILD", "BUILD", conventional.Build, false},
		{"mixed case ChOrE", "ChOrE", conventional.Chore, false},
		{"unknown type", "unknown", conventional.Type(0), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got conventional.Type
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

func TestParseType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    conventional.Type
		wantErr bool
	}{
		{"lowercase feat", "feat", conventional.Feat, false},
		{"uppercase FEAT", "FEAT", conventional.Feat, false},
		{"mixed case FiX", "FiX", conventional.Fix, false},
		{"lowercase ci", "ci", conventional.CI, false},
		{"uppercase CI", "CI", conventional.CI, false},
		{"all types lowercase", "refactor", conventional.Refactor, false},
		{"with leading whitespace", "  fix", conventional.Fix, false},
		{"with trailing whitespace", "test  ", conventional.Test, false},
		{"with surrounding whitespace", "  perf  ", conventional.Perf, false},
		{"with tabs", "\tbuild\t", conventional.Build, false},
		{"unknown type", "unknown", conventional.Type(0), true},
		{"empty string", "", conventional.Type(0), true},
		{"only spaces", "   ", conventional.Type(0), true},
		{"only tabs", "\t\t", conventional.Type(0), true},
		{"only whitespace mixed", " \t \n ", conventional.Type(0), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := conventional.ParseType(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestType_JSON_RoundTrip(t *testing.T) {
	original := conventional.Fix

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded conventional.Type
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded != original {
		t.Errorf("JSON round-trip failed: got %v, want %v", decoded, original)
	}
}

func TestType_YAML_RoundTrip(t *testing.T) {
	original := conventional.Perf

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("yaml.Marshal() error = %v", err)
	}

	var decoded conventional.Type
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	if decoded != original {
		t.Errorf("YAML round-trip failed: got %v, want %v", decoded, original)
	}
}

func TestType_StringConstants(t *testing.T) {
	// Verify string constants match Type.String() output
	tests := []struct {
		typ  conventional.Type
		want string
	}{
		{conventional.Feat, conventional.FeatStr},
		{conventional.Fix, conventional.FixStr},
		{conventional.Docs, conventional.DocsStr},
		{conventional.Style, conventional.StyleStr},
		{conventional.Refactor, conventional.RefactorStr},
		{conventional.Perf, conventional.PerfStr},
		{conventional.Test, conventional.TestStr},
		{conventional.Build, conventional.BuildStr},
		{conventional.CI, conventional.CIStr},
		{conventional.Chore, conventional.ChoreStr},
		{conventional.Revert, conventional.RevertStr},
	}

	for _, tt := range tests {
		t.Run(tt.typ.String(), func(t *testing.T) {
			if got := tt.typ.String(); got != tt.want {
				t.Errorf("Type.String() = %q, constant = %q, mismatch", got, tt.want)
			}
		})
	}
}
