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

func TestScope_String(t *testing.T) {
	tests := []struct {
		name  string
		scope conventional.Scope
		want  string
	}{
		{"empty", conventional.Scope(""), ""},
		{"simple", conventional.Scope("api"), "api"},
		{"with dash", conventional.Scope("http-router"), "http-router"},
		{"with slash", conventional.Scope("core/io"), "core/io"},
		{"with dot", conventional.Scope("db.v2"), "db.v2"},
		{"with underscore", conventional.Scope("pkg_utils"), "pkg_utils"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.scope.String(); got != tt.want {
				t.Errorf("Scope.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScope_Redacted(t *testing.T) {
	// Redacted should be identical to String for Scope
	scope := conventional.Scope("api")
	if scope.Redacted() != scope.String() {
		t.Errorf("Redacted() = %q, want %q", scope.Redacted(), scope.String())
	}
}

func TestScope_TypeName(t *testing.T) {
	scope := conventional.Scope("api")
	if got := scope.TypeName(); got != "Scope" {
		t.Errorf("TypeName() = %q, want %q", got, "Scope")
	}
}

func TestScope_IsZero(t *testing.T) {
	tests := []struct {
		name  string
		scope conventional.Scope
		want  bool
	}{
		{"empty is zero", conventional.Scope(""), true},
		{"non-empty is not zero", conventional.Scope("api"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.scope.IsZero(); got != tt.want {
				t.Errorf("IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScope_Validate(t *testing.T) {
	tests := []struct {
		name    string
		scope   conventional.Scope
		wantErr bool
	}{
		// Valid scopes
		{"empty valid", conventional.Scope(""), false},
		{"simple valid", conventional.Scope("api"), false},
		{"with dash", conventional.Scope("http-router"), false},
		{"with slash", conventional.Scope("core/io"), false},
		{"with dot", conventional.Scope("db.v2"), false},
		{"with underscore", conventional.Scope("pkg_utils"), false},
		{"single char", conventional.Scope("a"), false},
		{"starts with digit", conventional.Scope("2fa"), false},
		{"ends with digit", conventional.Scope("api2"), false},
		{"hierarchical", conventional.Scope("platform/services/auth"), false},

		// Invalid scopes
		{"contains uppercase", conventional.Scope("API"), true},
		{"contains space", conventional.Scope("api test"), true},
		{"contains tab", conventional.Scope("api\ttest"), true},
		{"contains newline", conventional.Scope("api\ntest"), true},
		{"starts with dash", conventional.Scope("-api"), true},
		{"ends with dash", conventional.Scope("api-"), true},
		{"starts with dot", conventional.Scope(".api"), true},
		{"ends with dot", conventional.Scope("api."), true},
		{"starts with slash", conventional.Scope("/api"), true},
		{"ends with slash", conventional.Scope("api/"), true},
		{"special char", conventional.Scope("api*"), true},
		{"too long", conventional.Scope(strings.Repeat("a", 33)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.scope.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestScope_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		scope   conventional.Scope
		want    string
		wantErr bool
	}{
		{"empty", conventional.Scope(""), `""`, false},
		{"simple", conventional.Scope("api"), `"api"`, false},
		{"with slash", conventional.Scope("core/io"), `"core/io"`, false},
		{"invalid uppercase", conventional.Scope("API"), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.scope)
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

func TestScope_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    conventional.Scope
		wantErr bool
	}{
		{"empty", `""`, conventional.Scope(""), false},
		{"lowercase api", `"api"`, conventional.Scope("api"), false},
		{"uppercase API", `"API"`, conventional.Scope("api"), false},
		{"mixed case Api", `"Api"`, conventional.Scope("api"), false},
		{"with whitespace", `"  core/io  "`, conventional.Scope("core/io"), false},
		{"hierarchical", `"platform/services/auth"`, conventional.Scope("platform/services/auth"), false},
		{"invalid too long", `"` + strings.Repeat("a", 33) + `"`, conventional.Scope(""), true},
		{"invalid special char", `"api*"`, conventional.Scope(""), true},
		{"invalid JSON", `not json`, conventional.Scope(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got conventional.Scope
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

func TestScope_MarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		scope   conventional.Scope
		want    string
		wantErr bool
	}{
		{"empty", conventional.Scope(""), "\"\"\n", false},
		{"simple", conventional.Scope("api"), "api\n", false},
		{"with dash", conventional.Scope("http-router"), "http-router\n", false},
		{"invalid uppercase", conventional.Scope("API"), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := yaml.Marshal(tt.scope)
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

func TestScope_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    conventional.Scope
		wantErr bool
	}{
		{"empty", `""`, conventional.Scope(""), false},
		{"lowercase test", "api", conventional.Scope("api"), false},
		{"uppercase BUILD", "CORE", conventional.Scope("core"), false},
		{"mixed case", "DbUtils", conventional.Scope("dbutils"), false},
		{"with whitespace", "  auth  ", conventional.Scope("auth"), false},
		{"invalid too long", strings.Repeat("a", 33), conventional.Scope(""), true},
		{"invalid special", "api*", conventional.Scope(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got conventional.Scope
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

func TestParseScope(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    conventional.Scope
		wantErr bool
	}{
		// Valid inputs
		{"empty", "", conventional.Scope(""), false},
		{"lowercase api", "api", conventional.Scope("api"), false},
		{"uppercase API", "API", conventional.Scope("api"), false},
		{"mixed case Core", "Core", conventional.Scope("core"), false},
		{"with leading whitespace", "  api", conventional.Scope("api"), false},
		{"with trailing whitespace", "api  ", conventional.Scope("api"), false},
		{"with surrounding whitespace", "  core/io  ", conventional.Scope("core/io"), false},
		{"with tabs", "\tauth\t", conventional.Scope("auth"), false},
		{"only whitespace", "   ", conventional.Scope(""), false},
		{"hierarchical", "platform/services/auth", conventional.Scope("platform/services/auth"), false},
		{"with dot", "db.v2", conventional.Scope("db.v2"), false},
		{"with underscore", "pkg_utils", conventional.Scope("pkg_utils"), false},
		{"with dash", "http-router", conventional.Scope("http-router"), false},

		// Invalid inputs
		{"too long", strings.Repeat("a", 33), conventional.Scope(""), true},
		{"special char", "api*", conventional.Scope(""), true},
		{"starts with dash", "-api", conventional.Scope(""), true},
		{"ends with dash", "api-", conventional.Scope(""), true},
		{"starts with dot", ".api", conventional.Scope(""), true},
		{"ends with dot", "api.", conventional.Scope(""), true},
		{"contains internal space", "api test", conventional.Scope(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := conventional.ParseScope(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseScope() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseScope() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScope_JSON_RoundTrip(t *testing.T) {
	original := conventional.Scope("core/io")

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded conventional.Scope
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded != original {
		t.Errorf("JSON round-trip failed: got %v, want %v", decoded, original)
	}
}

func TestScope_YAML_RoundTrip(t *testing.T) {
	original := conventional.Scope("db.v2")

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("yaml.Marshal() error = %v", err)
	}

	var decoded conventional.Scope
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	if decoded != original {
		t.Errorf("YAML round-trip failed: got %v, want %v", decoded, original)
	}
}

func TestScope_LengthConstraints(t *testing.T) {
	// Test minimum length (1 character, non-empty)
	minScope := conventional.Scope("a")
	if err := minScope.Validate(); err != nil {
		t.Errorf("Scope with min length should be valid, got error: %v", err)
	}

	// Test maximum length (32 characters)
	maxScope := conventional.Scope(strings.Repeat("a", 32))
	if err := maxScope.Validate(); err != nil {
		t.Errorf("Scope with max length should be valid, got error: %v", err)
	}

	// Test over maximum length (33 characters)
	tooLongScope := conventional.Scope(strings.Repeat("a", 33))
	if err := tooLongScope.Validate(); err == nil {
		t.Error("Scope over max length should be invalid")
	}
}

func TestScope_RegexpValidation(t *testing.T) {
	tests := []struct {
		name    string
		scope   conventional.Scope
		wantErr bool
	}{
		{"starts and ends with letter", conventional.Scope("api"), false},
		{"starts with digit", conventional.Scope("2fa"), false},
		{"ends with digit", conventional.Scope("api2"), false},
		{"contains allowed middle chars", conventional.Scope("a-b_c.d/e"), false},
		{"only lowercase letters", conventional.Scope("abcxyz"), false},
		{"only digits", conventional.Scope("123"), false},

		{"contains uppercase", conventional.Scope("Api"), true},
		{"starts with dash", conventional.Scope("-api"), true},
		{"ends with dash", conventional.Scope("api-"), true},
		{"starts with dot", conventional.Scope(".api"), true},
		{"ends with dot", conventional.Scope("api."), true},
		{"starts with underscore", conventional.Scope("_api"), true},
		{"ends with underscore", conventional.Scope("api_"), true},
		{"starts with slash", conventional.Scope("/api"), true},
		{"ends with slash", conventional.Scope("api/"), true},
		{"contains asterisk", conventional.Scope("api*"), true},
		{"contains space", conventional.Scope("api test"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.scope.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v for scope %q", err, tt.wantErr, tt.scope)
			}
		})
	}
}

func TestScope_Normalization(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  conventional.Scope
	}{
		{"trims leading space", "  api", conventional.Scope("api")},
		{"trims trailing space", "api  ", conventional.Scope("api")},
		{"trims both", "  api  ", conventional.Scope("api")},
		{"converts to lowercase", "API", conventional.Scope("api")},
		{"converts mixed case", "Core/IO", conventional.Scope("core/io")},
		{"trims and lowercases", "  AUTH  ", conventional.Scope("auth")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := conventional.ParseScope(tt.input)
			if err != nil {
				t.Fatalf("ParseScope() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("ParseScope() = %v, want %v", got, tt.want)
			}
		})
	}
}
