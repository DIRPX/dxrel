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
	"fmt"
	"strings"
	"testing"
	"time"

	"dirpx.dev/dxrel/dxcore/model/git"
	"gopkg.in/yaml.v3"
)

func TestSignature_String(t *testing.T) {
	testTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name string
		sig  git.Signature
		want string
	}{
		{
			name: "complete_signature",
			sig: git.Signature{
				Name:  "Jane Doe",
				Email: "jane@example.com",
				When:  testTime,
			},
			want: "Signature{Name:Jane Doe, Email:jane@example.com, When:2025-01-15T10:30:00Z}",
		},
		{
			name: "zero_signature",
			sig:  git.Signature{},
			want: "Signature{Name:, Email:, When:0001-01-01T00:00:00Z}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.sig.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSignature_Redacted(t *testing.T) {
	testTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name string
		sig  git.Signature
		want string
	}{
		{
			name: "redacts_email",
			sig: git.Signature{
				Name:  "Jane Doe",
				Email: "jane@example.com",
				When:  testTime,
			},
			want: "Signature{Name:Jane Doe, Email:j***@example.com, When:2025-01-15T10:30:00Z}",
		},
		{
			name: "short_email",
			sig: git.Signature{
				Name:  "John",
				Email: "a@b.c",
				When:  testTime,
			},
			want: "Signature{Name:John, Email:a***@b.c, When:2025-01-15T10:30:00Z}",
		},
		{
			name: "empty_email",
			sig: git.Signature{
				Name:  "Test",
				Email: "",
				When:  testTime,
			},
			want: "Signature{Name:Test, Email:[empty], When:2025-01-15T10:30:00Z}",
		},
		{
			name: "invalid_email_no_at",
			sig: git.Signature{
				Name:  "Test",
				Email: "notanemail",
				When:  testTime,
			},
			want: "Signature{Name:Test, Email:[invalid], When:2025-01-15T10:30:00Z}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.sig.Redacted()
			if got != tt.want {
				t.Errorf("Redacted() = %q, want %q", got, tt.want)
			}

			// Verify email is actually redacted
			if tt.sig.Email != "" && strings.Contains(got, tt.sig.Email) && tt.sig.Email != "[empty]" && tt.sig.Email != "[invalid]" {
				t.Errorf("Redacted() still contains full email: %q", got)
			}
		})
	}
}

func TestSignature_TypeName(t *testing.T) {
	sig := git.Signature{}
	if got := sig.TypeName(); got != "Signature" {
		t.Errorf("TypeName() = %q, want %q", got, "Signature")
	}
}

func TestSignature_IsZero(t *testing.T) {
	testTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name string
		sig  git.Signature
		want bool
	}{
		{
			name: "zero_signature",
			sig:  git.Signature{},
			want: true,
		},
		{
			name: "with_name",
			sig: git.Signature{
				Name: "Jane",
			},
			want: false,
		},
		{
			name: "with_email",
			sig: git.Signature{
				Email: "jane@example.com",
			},
			want: false,
		},
		{
			name: "with_when",
			sig: git.Signature{
				When: testTime,
			},
			want: false,
		},
		{
			name: "complete_signature",
			sig: git.Signature{
				Name:  "Jane",
				Email: "jane@example.com",
				When:  testTime,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.sig.IsZero()
			if got != tt.want {
				t.Errorf("IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSignature_Equal(t *testing.T) {
	t1 := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	t2 := time.Date(2025, 1, 16, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name string
		s1   git.Signature
		s2   git.Signature
		want bool
	}{
		{
			name: "both_zero",
			s1:   git.Signature{},
			s2:   git.Signature{},
			want: true,
		},
		{
			name: "same_complete_signatures",
			s1: git.Signature{
				Name:  "Jane Doe",
				Email: "jane@example.com",
				When:  t1,
			},
			s2: git.Signature{
				Name:  "Jane Doe",
				Email: "jane@example.com",
				When:  t1,
			},
			want: true,
		},
		{
			name: "different_names",
			s1: git.Signature{
				Name:  "Jane Doe",
				Email: "jane@example.com",
				When:  t1,
			},
			s2: git.Signature{
				Name:  "John Doe",
				Email: "jane@example.com",
				When:  t1,
			},
			want: false,
		},
		{
			name: "different_emails",
			s1: git.Signature{
				Name:  "Jane Doe",
				Email: "jane@example.com",
				When:  t1,
			},
			s2: git.Signature{
				Name:  "Jane Doe",
				Email: "jane@different.com",
				When:  t1,
			},
			want: false,
		},
		{
			name: "different_times",
			s1: git.Signature{
				Name:  "Jane Doe",
				Email: "jane@example.com",
				When:  t1,
			},
			s2: git.Signature{
				Name:  "Jane Doe",
				Email: "jane@example.com",
				When:  t2,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.s1.Equal(tt.s2)
			if got != tt.want {
				t.Errorf("Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSignature_Validate(t *testing.T) {
	testTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	longName := strings.Repeat("a", 257)
	longEmail := strings.Repeat("a", 246) + "@test.com" // 255 chars total

	tests := []struct {
		name    string
		sig     git.Signature
		wantErr bool
	}{
		{
			name: "valid_signature",
			sig: git.Signature{
				Name:  "Jane Doe",
				Email: "jane@example.com",
				When:  testTime,
			},
			wantErr: false,
		},
		{
			name: "valid_with_unicode_name",
			sig: git.Signature{
				Name:  "李明",
				Email: "li@example.com",
				When:  testTime,
			},
			wantErr: false,
		},
		{
			name: "valid_complex_email",
			sig: git.Signature{
				Name:  "Developer",
				Email: "developer+git@sub.domain.co.uk",
				When:  testTime,
			},
			wantErr: false,
		},
		{
			name:    "invalid_zero_signature",
			sig:     git.Signature{},
			wantErr: true,
		},
		{
			name: "invalid_empty_name",
			sig: git.Signature{
				Name:  "",
				Email: "jane@example.com",
				When:  testTime,
			},
			wantErr: true,
		},
		{
			name: "invalid_empty_email",
			sig: git.Signature{
				Name:  "Jane Doe",
				Email: "",
				When:  testTime,
			},
			wantErr: true,
		},
		{
			name: "invalid_zero_when",
			sig: git.Signature{
				Name:  "Jane Doe",
				Email: "jane@example.com",
				When:  time.Time{},
			},
			wantErr: true,
		},
		{
			name: "invalid_name_too_long",
			sig: git.Signature{
				Name:  longName,
				Email: "jane@example.com",
				When:  testTime,
			},
			wantErr: true,
		},
		{
			name: "invalid_email_too_long",
			sig: git.Signature{
				Name:  "Jane Doe",
				Email: longEmail,
				When:  testTime,
			},
			wantErr: true,
		},
		{
			name: "invalid_email_no_at",
			sig: git.Signature{
				Name:  "Jane Doe",
				Email: "notanemail",
				When:  testTime,
			},
			wantErr: true,
		},
		{
			name: "invalid_email_no_domain",
			sig: git.Signature{
				Name:  "Jane Doe",
				Email: "jane@",
				When:  testTime,
			},
			wantErr: true,
		},
		{
			name: "invalid_email_with_space",
			sig: git.Signature{
				Name:  "Jane Doe",
				Email: "jane @example.com",
				When:  testTime,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.sig.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSignature_MarshalJSON(t *testing.T) {
	testTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name    string
		sig     git.Signature
		wantErr bool
	}{
		{
			name: "valid_signature",
			sig: git.Signature{
				Name:  "Jane Doe",
				Email: "jane@example.com",
				When:  testTime,
			},
			wantErr: false,
		},
		{
			name:    "invalid_signature",
			sig:     git.Signature{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.sig)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("MarshalJSON() returned nil data for valid signature")
			}
		})
	}
}

func TestSignature_UnmarshalJSON(t *testing.T) {
	testTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name    string
		json    string
		want    git.Signature
		wantErr bool
	}{
		{
			name: "valid_json",
			json: `{"name":"Jane Doe","email":"jane@example.com","when":"2025-01-15T10:30:00Z"}`,
			want: git.Signature{
				Name:  "Jane Doe",
				Email: "jane@example.com",
				When:  testTime,
			},
			wantErr: false,
		},
		{
			name:    "invalid_json_zero_signature",
			json:    `{"name":"","email":"","when":"0001-01-01T00:00:00Z"}`,
			wantErr: true,
		},
		{
			name:    "invalid_json_format",
			json:    `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got git.Signature
			err := json.Unmarshal([]byte(tt.json), &got)
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

func TestSignature_MarshalYAML(t *testing.T) {
	testTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name    string
		sig     git.Signature
		wantErr bool
	}{
		{
			name: "valid_signature",
			sig: git.Signature{
				Name:  "Jane Doe",
				Email: "jane@example.com",
				When:  testTime,
			},
			wantErr: false,
		},
		{
			name:    "invalid_signature",
			sig:     git.Signature{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := yaml.Marshal(tt.sig)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("MarshalYAML() returned nil data for valid signature")
			}
		})
	}
}

func TestSignature_UnmarshalYAML(t *testing.T) {
	testTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name    string
		yaml    string
		want    git.Signature
		wantErr bool
	}{
		{
			name: "valid_yaml",
			yaml: `name: Jane Doe
email: jane@example.com
when: 2025-01-15T10:30:00Z`,
			want: git.Signature{
				Name:  "Jane Doe",
				Email: "jane@example.com",
				When:  testTime,
			},
			wantErr: false,
		},
		{
			name: "invalid_yaml_zero_signature",
			yaml: `name: ""
email: ""
when: 0001-01-01T00:00:00Z`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got git.Signature
			err := yaml.Unmarshal([]byte(tt.yaml), &got)
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

func TestSignature_JSON_RoundTrip(t *testing.T) {
	testTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	sig := git.Signature{
		Name:  "Jane Doe",
		Email: "jane@example.com",
		When:  testTime,
	}

	data, err := json.Marshal(sig)
	if err != nil {
		t.Fatalf("Marshal() failed: %v", err)
	}

	var decoded git.Signature
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() failed: %v", err)
	}

	if !decoded.Equal(sig) {
		t.Errorf("Round-trip failed: got %+v, want %+v", decoded, sig)
	}
}

func TestSignature_YAML_RoundTrip(t *testing.T) {
	testTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	sig := git.Signature{
		Name:  "John Smith",
		Email: "john@example.org",
		When:  testTime,
	}

	data, err := yaml.Marshal(sig)
	if err != nil {
		t.Fatalf("Marshal() failed: %v", err)
	}

	var decoded git.Signature
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() failed: %v", err)
	}

	if !decoded.Equal(sig) {
		t.Errorf("Round-trip failed: got %+v, want %+v", decoded, sig)
	}
}

func TestNewSignature(t *testing.T) {
	testTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name      string
		sigName   string
		email     string
		when      time.Time
		wantErr   bool
		wantEqual git.Signature
	}{
		{
			name:    "valid_signature",
			sigName: "Jane Doe",
			email:   "jane@example.com",
			when:    testTime,
			wantErr: false,
			wantEqual: git.Signature{
				Name:  "Jane Doe",
				Email: "jane@example.com",
				When:  testTime,
			},
		},
		{
			name:    "invalid_empty_name",
			sigName: "",
			email:   "jane@example.com",
			when:    testTime,
			wantErr: true,
		},
		{
			name:    "invalid_empty_email",
			sigName: "Jane Doe",
			email:   "",
			when:    testTime,
			wantErr: true,
		},
		{
			name:    "invalid_zero_when",
			sigName: "Jane Doe",
			email:   "jane@example.com",
			when:    time.Time{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := git.NewSignature(tt.sigName, tt.email, tt.when)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSignature() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.wantEqual) {
				t.Errorf("NewSignature() = %+v, want %+v", got, tt.wantEqual)
			}
		})
	}
}

func TestSignature_CommonScenarios(t *testing.T) {
	testTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	scenarios := []struct {
		name  string
		sig   git.Signature
		valid bool
	}{
		{
			name: "typical_author",
			sig: git.Signature{
				Name:  "Alice Developer",
				Email: "alice@company.com",
				When:  testTime,
			},
			valid: true,
		},
		{
			name: "unicode_name",
			sig: git.Signature{
				Name:  "山田太郎",
				Email: "yamada@example.jp",
				When:  testTime,
			},
			valid: true,
		},
		{
			name: "github_noreply_email",
			sig: git.Signature{
				Name:  "Developer",
				Email: "12345+developer@users.noreply.github.com",
				When:  testTime,
			},
			valid: true,
		},
		{
			name: "long_name",
			sig: git.Signature{
				Name:  "Dr. Professional Middle-Name-Hyphenated Surname-Also-Hyphenated III",
				Email: "doctor@university.edu",
				When:  testTime,
			},
			valid: true,
		},
	}

	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.sig.Validate()
			if tt.valid && err != nil {
				t.Errorf("Expected valid signature, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("Expected invalid signature, but validation passed")
			}

			// Test round-trip
			if tt.valid {
				data, err := json.Marshal(tt.sig)
				if err != nil {
					t.Fatalf("JSON marshal failed: %v", err)
				}

				var decoded git.Signature
				if err := json.Unmarshal(data, &decoded); err != nil {
					t.Fatalf("JSON unmarshal failed: %v", err)
				}

				if !decoded.Equal(tt.sig) {
					t.Errorf("JSON round-trip failed: got %+v, want %+v", decoded, tt.sig)
				}
			}
		})
	}
}

func ExampleSignature() {
	// Create a signature for a commit author
	sig, err := git.NewSignature(
		"Jane Doe",
		"jane@example.com",
		time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println(sig.String())
	fmt.Println(sig.Redacted())

	// Output:
	// Signature{Name:Jane Doe, Email:jane@example.com, When:2025-01-15T10:30:00Z}
	// Signature{Name:Jane Doe, Email:j***@example.com, When:2025-01-15T10:30:00Z}
}
