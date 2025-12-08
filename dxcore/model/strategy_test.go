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

package model

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestStrategy_String(t *testing.T) {
	tests := []struct {
		name     string
		strategy Strategy
		want     string
	}{
		{"MaxSeverity", MaxSeverity, "max-severity"},
		{"Sequential", Sequential, "sequential"},
		{"Unknown", Strategy(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.strategy.String(); got != tt.want {
				t.Errorf("Strategy.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseStrategy(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Strategy
		wantErr bool
	}{
		// Valid inputs - max-severity
		{"max-severity", "max-severity", MaxSeverity, false},
		{"MaxSeverity", "MaxSeverity", MaxSeverity, false},
		{"max_severity", "max_severity", MaxSeverity, false},
		{"MAX_SEVERITY", "MAX_SEVERITY", MaxSeverity, false},

		// Valid inputs - sequential
		{"sequential", "sequential", Sequential, false},
		{"Sequential", "Sequential", Sequential, false},
		{"SEQUENTIAL", "SEQUENTIAL", Sequential, false},

		// Invalid inputs
		{"empty", "", MaxSeverity, true},
		{"invalid", "invalid", MaxSeverity, true},
		{"number", "1", MaxSeverity, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseStrategy(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseStrategy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseStrategy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStrategy_Valid(t *testing.T) {
	tests := []struct {
		name     string
		strategy Strategy
		want     bool
	}{
		{"MaxSeverity", MaxSeverity, true},
		{"Sequential", Sequential, true},
		{"Invalid negative", Strategy(-1), false},
		{"Invalid positive", Strategy(99), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.strategy.Valid(); got != tt.want {
				t.Errorf("Strategy.Valid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStrategy_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		strategy Strategy
		want     string
		wantErr  bool
	}{
		{"MaxSeverity", MaxSeverity, `"max-severity"`, false},
		{"Sequential", Sequential, `"sequential"`, false},
		{"Invalid", Strategy(99), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.strategy)
			if (err != nil) != tt.wantErr {
				t.Errorf("Strategy.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(got) != tt.want {
				t.Errorf("Strategy.MarshalJSON() = %v, want %v", string(got), tt.want)
			}
		})
	}
}

func TestStrategy_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Strategy
		wantErr bool
	}{
		// String format
		{"max-severity string", `"max-severity"`, MaxSeverity, false},
		{"sequential string", `"sequential"`, Sequential, false},

		// Numeric format
		{"max-severity numeric", `0`, MaxSeverity, false},
		{"sequential numeric", `1`, Sequential, false},

		// Invalid inputs
		{"empty", `""`, MaxSeverity, true},
		{"invalid string", `"invalid"`, MaxSeverity, true},
		{"invalid number", `99`, MaxSeverity, true},
		{"empty data", ``, MaxSeverity, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Strategy
			err := json.Unmarshal([]byte(tt.input), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("Strategy.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Strategy.UnmarshalJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStrategy_YAML(t *testing.T) {
	tests := []struct {
		name     string
		strategy Strategy
		want     string
	}{
		{"MaxSeverity", MaxSeverity, "max-severity\n"},
		{"Sequential", Sequential, "sequential\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			got, err := yaml.Marshal(tt.strategy)
			if err != nil {
				t.Errorf("yaml.Marshal() error = %v", err)
				return
			}
			if string(got) != tt.want {
				t.Errorf("yaml.Marshal() = %v, want %v", string(got), tt.want)
			}

			// Unmarshal
			var strategy Strategy
			if err := yaml.Unmarshal(got, &strategy); err != nil {
				t.Errorf("yaml.Unmarshal() error = %v", err)
				return
			}
			if strategy != tt.strategy {
				t.Errorf("yaml.Unmarshal() = %v, want %v", strategy, tt.strategy)
			}
		})
	}
}

func TestStrategy_RoundTrip(t *testing.T) {
	tests := []Strategy{MaxSeverity, Sequential}

	for _, original := range tests {
		t.Run(original.String(), func(t *testing.T) {
			// JSON round-trip
			jsonData, err := json.Marshal(original)
			if err != nil {
				t.Fatalf("JSON Marshal error: %v", err)
			}
			var jsonResult Strategy
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
			var yamlResult Strategy
			if err := yaml.Unmarshal(yamlData, &yamlResult); err != nil {
				t.Fatalf("YAML Unmarshal error: %v", err)
			}
			if yamlResult != original {
				t.Errorf("YAML round-trip: got %v, want %v", yamlResult, original)
			}
		})
	}
}

func TestStrategy_TypeName(t *testing.T) {
	var s Strategy
	if got := s.TypeName(); got != "Strategy" {
		t.Errorf("TypeName() = %v, want Strategy", got)
	}
}

func TestStrategy_Redacted(t *testing.T) {
	tests := []struct {
		name     string
		strategy Strategy
		want     string
	}{
		{"MaxSeverity", MaxSeverity, "max-severity"},
		{"Sequential", Sequential, "sequential"},
		{"Unknown", Strategy(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.strategy.Redacted(); got != tt.want {
				t.Errorf("Redacted() = %v, want %v", got, tt.want)
			}
			// Redacted should match String for Strategy
			if got := tt.strategy.Redacted(); got != tt.strategy.String() {
				t.Errorf("Redacted() = %v, String() = %v (should match)", got, tt.strategy.String())
			}
		})
	}
}

func TestStrategy_IsZero(t *testing.T) {
	tests := []struct {
		name     string
		strategy Strategy
		want     bool
	}{
		{"MaxSeverity (zero value)", MaxSeverity, true},
		{"Sequential", Sequential, false},
		{"Invalid", Strategy(99), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.strategy.IsZero(); got != tt.want {
				t.Errorf("IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStrategy_Equal(t *testing.T) {
	tests := []struct {
		name  string
		s1    Strategy
		s2    any
		want  bool
	}{
		{"equal MaxSeverity", MaxSeverity, MaxSeverity, true},
		{"equal Sequential", Sequential, Sequential, true},
		{"different values", MaxSeverity, Sequential, false},
		{"pointer equal", MaxSeverity, func() *Strategy { s := MaxSeverity; return &s }(), true},
		{"pointer different", MaxSeverity, func() *Strategy { s := Sequential; return &s }(), false},
		{"nil pointer", MaxSeverity, (*Strategy)(nil), false},
		{"different type", MaxSeverity, "max-severity", false},
		{"different type int", MaxSeverity, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s1.Equal(tt.s2); got != tt.want {
				t.Errorf("Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStrategy_Validate(t *testing.T) {
	tests := []struct {
		name     string
		strategy Strategy
		wantErr  bool
	}{
		{"MaxSeverity valid", MaxSeverity, false},
		{"Sequential valid", Sequential, false},
		{"Invalid negative", Strategy(-1), true},
		{"Invalid positive", Strategy(99), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.strategy.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStrategy_MarshalJSON_Invalid(t *testing.T) {
	// Invalid Strategy should fail to marshal
	invalid := Strategy(99)
	_, err := json.Marshal(invalid)
	if err == nil {
		t.Error("Expected error marshaling invalid Strategy, got nil")
	}
}

func TestStrategy_MarshalText_Invalid(t *testing.T) {
	// Invalid Strategy should fail to marshal as text
	invalid := Strategy(99)
	_, err := invalid.MarshalText()
	if err == nil {
		t.Error("Expected error marshaling invalid Strategy as text, got nil")
	}
}
