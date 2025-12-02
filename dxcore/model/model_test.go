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

package model_test

import (
	"encoding/json"
	"errors"
	"testing"

	"dirpx.dev/dxrel/dxcore/model"
	"gopkg.in/yaml.v3"
)

// ExampleModel demonstrates a complete Model implementation.
type ExampleModel struct {
	Name     string
	Email    string
	Password string // Sensitive field
}

// Validate implements Validatable
func (e ExampleModel) Validate() error {
	if e.Name == "" {
		return errors.New("name required")
	}
	if e.Email == "" {
		return errors.New("email required")
	}
	return nil
}

// TypeName implements Identifiable
func (e ExampleModel) TypeName() string {
	return "ExampleModel"
}

// IsZero implements ZeroCheckable
func (e ExampleModel) IsZero() bool {
	return e.Name == "" && e.Email == "" && e.Password == ""
}

// Redacted implements Loggable (safe for production logs)
func (e ExampleModel) Redacted() string {
	return "ExampleModel{Name:" + e.Name + ", Email:" + redactEmail(e.Email) + ", Password:[REDACTED]}"
}

// String implements Loggable (UNSAFE - includes sensitive data)
func (e ExampleModel) String() string {
	return "ExampleModel{Name:" + e.Name + ", Email:" + e.Email + "}"
}

// MarshalJSON implements Serializable
func (e ExampleModel) MarshalJSON() ([]byte, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	type alias ExampleModel
	return json.Marshal((alias)(e))
}

// UnmarshalJSON implements Serializable
func (e *ExampleModel) UnmarshalJSON(data []byte) error {
	type alias ExampleModel
	if err := json.Unmarshal(data, (*alias)(e)); err != nil {
		return err
	}
	return e.Validate()
}

// MarshalYAML implements Serializable
func (e ExampleModel) MarshalYAML() (interface{}, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	type alias ExampleModel
	return (alias)(e), nil
}

// UnmarshalYAML implements Serializable
func (e *ExampleModel) UnmarshalYAML(node *yaml.Node) error {
	type alias ExampleModel
	if err := node.Decode((*alias)(e)); err != nil {
		return err
	}
	return e.Validate()
}

// Verify ExampleModel implements Model at compile time
var _ model.Model = (*ExampleModel)(nil)

func redactEmail(email string) string {
	// "user@example.com" -> "u***@example.com"
	if len(email) == 0 {
		return ""
	}
	idx := 0
	for i, c := range email {
		if c == '@' {
			idx = i
			break
		}
	}
	if idx == 0 {
		return "[INVALID]"
	}
	if idx == 1 {
		return "*@" + email[idx+1:]
	}
	return string(email[0]) + "***@" + email[idx+1:]
}

func TestModel_Validate(t *testing.T) {
	tests := []struct {
		name    string
		model   ExampleModel
		wantErr bool
	}{
		{
			name:    "valid model",
			model:   ExampleModel{Name: "John", Email: "john@example.com"},
			wantErr: false,
		},
		{
			name:    "missing name",
			model:   ExampleModel{Email: "john@example.com"},
			wantErr: true,
		},
		{
			name:    "missing email",
			model:   ExampleModel{Name: "John"},
			wantErr: true,
		},
		{
			name:    "empty model",
			model:   ExampleModel{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.model.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestModel_IsZero(t *testing.T) {
	tests := []struct {
		name  string
		model ExampleModel
		want  bool
	}{
		{
			name:  "zero model",
			model: ExampleModel{},
			want:  true,
		},
		{
			name:  "non-zero model",
			model: ExampleModel{Name: "John"},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.model.IsZero(); got != tt.want {
				t.Errorf("IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModel_Redacted(t *testing.T) {
	m := ExampleModel{
		Name:     "John",
		Email:    "john@example.com",
		Password: "secret123",
	}

	redacted := m.Redacted()

	// Should contain name
	if !contains(redacted, "John") {
		t.Errorf("Redacted() should contain name, got %q", redacted)
	}

	// Should NOT contain full email
	if contains(redacted, "john@") {
		t.Errorf("Redacted() should not contain full email, got %q", redacted)
	}

	// Should mask email
	if !contains(redacted, "j***@") {
		t.Errorf("Redacted() should mask email, got %q", redacted)
	}

	// Should NOT contain password
	if contains(redacted, "secret") {
		t.Errorf("Redacted() should not contain password, got %q", redacted)
	}

	// Should indicate password is redacted
	if !contains(redacted, "[REDACTED]") {
		t.Errorf("Redacted() should indicate redacted fields, got %q", redacted)
	}
}

func TestModel_JSON_RoundTrip(t *testing.T) {
	original := ExampleModel{
		Name:  "John",
		Email: "john@example.com",
	}

	// Marshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Unmarshal
	var decoded ExampleModel
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	// Compare
	if decoded.Name != original.Name || decoded.Email != original.Email {
		t.Errorf("JSON round-trip failed: got %+v, want %+v", decoded, original)
	}
}

func TestModel_YAML_RoundTrip(t *testing.T) {
	original := ExampleModel{
		Name:  "John",
		Email: "john@example.com",
	}

	// Marshal
	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("yaml.Marshal() error = %v", err)
	}

	// Unmarshal
	var decoded ExampleModel
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	// Compare
	if decoded.Name != original.Name || decoded.Email != original.Email {
		t.Errorf("YAML round-trip failed: got %+v, want %+v", decoded, original)
	}
}

func TestModel_Marshal_FailsOnInvalid(t *testing.T) {
	invalid := ExampleModel{} // Missing required fields

	// JSON marshal should fail
	_, err := json.Marshal(invalid)
	if err == nil {
		t.Error("json.Marshal() should fail on invalid model")
	}

	// YAML marshal should fail
	_, err = yaml.Marshal(invalid)
	if err == nil {
		t.Error("yaml.Marshal() should fail on invalid model")
	}
}

func TestModel_Unmarshal_FailsOnInvalid(t *testing.T) {
	// JSON with missing required field
	jsonData := []byte(`{"email":"john@example.com"}`)

	var m ExampleModel
	err := json.Unmarshal(jsonData, &m)
	if err == nil {
		t.Error("json.Unmarshal() should fail when validation fails")
	}

	// YAML with missing required field
	yamlData := []byte("email: john@example.com")

	var m2 ExampleModel
	err = yaml.Unmarshal(yamlData, &m2)
	if err == nil {
		t.Error("yaml.Unmarshal() should fail when validation fails")
	}
}

func TestModel_TypeName(t *testing.T) {
	m := ExampleModel{Name: "John", Email: "john@example.com"}

	typeName := m.TypeName()

	if typeName != "ExampleModel" {
		t.Errorf("TypeName() = %q, want %q", typeName, "ExampleModel")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
