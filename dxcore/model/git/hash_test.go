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
	"strings"
	"testing"

	"dirpx.dev/dxrel/dxcore/model/git"
	"gopkg.in/yaml.v3"
)

func TestHash_String(t *testing.T) {
	tests := []struct {
		name string
		hash git.Hash
		want string
	}{
		{"empty", git.Hash(""), ""},
		{"sha1", git.Hash("a1b2c3d4e5f6789012345678901234567890abcd"), "a1b2c3d4e5f6789012345678901234567890abcd"},
		{"sha256", git.Hash("a1b2c3d4e5f6789012345678901234567890abcda1b2c3d4e5f6789012345678"), "a1b2c3d4e5f6789012345678901234567890abcda1b2c3d4e5f6789012345678"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.hash.String(); got != tt.want {
				t.Errorf("Hash.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHash_Redacted(t *testing.T) {
	tests := []struct {
		name string
		hash git.Hash
		want string
	}{
		{"empty", git.Hash(""), ""},
		{"sha1", git.Hash("a1b2c3d4e5f6789012345678901234567890abcd"), "a1b2c3d"},
		{"sha256", git.Hash("a1b2c3d4e5f6789012345678901234567890abcda1b2c3d4e5f6789012345678"), "a1b2c3d"},
		{"short hash", git.Hash("a1b2c"), "a1b2c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.hash.Redacted(); got != tt.want {
				t.Errorf("Hash.Redacted() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHash_TypeName(t *testing.T) {
	hash := git.Hash("a1b2c3d4e5f6789012345678901234567890abcd")
	if got := hash.TypeName(); got != "Hash" {
		t.Errorf("TypeName() = %q, want %q", got, "Hash")
	}
}

func TestHash_IsZero(t *testing.T) {
	tests := []struct {
		name string
		hash git.Hash
		want bool
	}{
		{"empty is zero", git.Hash(""), true},
		{"sha1 not zero", git.Hash("a1b2c3d4e5f6789012345678901234567890abcd"), false},
		{"sha256 not zero", git.Hash("a1b2c3d4e5f6789012345678901234567890abcda1b2c3d4e5f6789012345678"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.hash.IsZero(); got != tt.want {
				t.Errorf("IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHash_Equal(t *testing.T) {
	tests := []struct {
		name string
		h1   git.Hash
		h2   git.Hash
		want bool
	}{
		{"both empty", git.Hash(""), git.Hash(""), true},
		{"same sha1", git.Hash("a1b2c3d4e5f6789012345678901234567890abcd"), git.Hash("a1b2c3d4e5f6789012345678901234567890abcd"), true},
		{"different sha1", git.Hash("a1b2c3d4e5f6789012345678901234567890abcd"), git.Hash("1234567890abcdef1234567890abcdef12345678"), false},
		{"empty vs non-empty", git.Hash(""), git.Hash("a1b2c3d4e5f6789012345678901234567890abcd"), false},
		{"case difference", git.Hash("a1b2c3d4e5f6789012345678901234567890abcd"), git.Hash("A1B2C3D4E5F6789012345678901234567890ABCD"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.h1.Equal(tt.h2); got != tt.want {
				t.Errorf("Hash.Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHash_Short(t *testing.T) {
	tests := []struct {
		name string
		hash git.Hash
		want string
	}{
		{"empty", git.Hash(""), ""},
		{"sha1", git.Hash("a1b2c3d4e5f6789012345678901234567890abcd"), "a1b2c3d"},
		{"sha256", git.Hash("a1b2c3d4e5f6789012345678901234567890abcda1b2c3d4e5f6789012345678"), "a1b2c3d"},
		{"short hash", git.Hash("a1b2c"), "a1b2c"},
		{"exactly 7 chars", git.Hash("a1b2c3d"), "a1b2c3d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.hash.Short(); got != tt.want {
				t.Errorf("Hash.Short() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHash_IsSHA1(t *testing.T) {
	tests := []struct {
		name string
		hash git.Hash
		want bool
	}{
		{"empty", git.Hash(""), false},
		{"sha1", git.Hash("a1b2c3d4e5f6789012345678901234567890abcd"), true},
		{"sha256", git.Hash("a1b2c3d4e5f6789012345678901234567890abcda1b2c3d4e5f6789012345678"), false},
		{"abbreviated", git.Hash("a1b2c3d"), false},
		{"wrong length", git.Hash("a1b2c3d4e5f6789012345678901234567890"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.hash.IsSHA1(); got != tt.want {
				t.Errorf("Hash.IsSHA1() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHash_IsSHA256(t *testing.T) {
	tests := []struct {
		name string
		hash git.Hash
		want bool
	}{
		{"empty", git.Hash(""), false},
		{"sha1", git.Hash("a1b2c3d4e5f6789012345678901234567890abcd"), false},
		{"sha256", git.Hash("a1b2c3d4e5f6789012345678901234567890abcda1b2c3d4e5f6789012345678"), true},
		{"abbreviated", git.Hash("a1b2c3d"), false},
		{"wrong length", git.Hash("a1b2c3d4e5f6789012345678901234567890abcda1b2c3d4"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.hash.IsSHA256(); got != tt.want {
				t.Errorf("Hash.IsSHA256() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHash_Validate(t *testing.T) {
	tests := []struct {
		name    string
		hash    git.Hash
		wantErr bool
	}{
		// Valid hashes
		{"empty valid", git.Hash(""), false},
		{"sha1 valid", git.Hash("a1b2c3d4e5f6789012345678901234567890abcd"), false},
		{"sha256 valid", git.Hash("a1b2c3d4e5f6789012345678901234567890abcda1b2c3d4e5f6789012345678"), false},
		{"sha1 all zeros", git.Hash("0000000000000000000000000000000000000000"), false},
		{"sha1 all f", git.Hash("ffffffffffffffffffffffffffffffffffffffff"), false},

		// Invalid hashes
		{"abbreviated", git.Hash("a1b2c3d"), true},
		{"sha1 with uppercase", git.Hash("A1B2C3D4E5F6789012345678901234567890ABCD"), true},
		{"sha1 with mixed case", git.Hash("a1B2c3D4e5F6789012345678901234567890abCD"), true},
		{"sha1 with invalid char", git.Hash("g1b2c3d4e5f6789012345678901234567890abcd"), true},
		{"sha1 with space", git.Hash("a1b2c3d4e5f6789012345678901234567890abc "), true},
		{"sha1 too short", git.Hash("a1b2c3d4e5f6789012345678901234567890abc"), true},
		{"sha1 too long", git.Hash("a1b2c3d4e5f6789012345678901234567890abcde"), true},
		{"sha256 with uppercase", git.Hash("A1B2C3D4E5F6789012345678901234567890ABCDA1B2C3D4E5F6789012345678"), true},
		{"sha256 too short", git.Hash("a1b2c3d4e5f6789012345678901234567890abcda1b2c3d4e5f678901234567"), true},
		{"sha256 too long", git.Hash("a1b2c3d4e5f6789012345678901234567890abcda1b2c3d4e5f67890123456789"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.hash.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHash_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		hash    git.Hash
		wantErr bool
	}{
		{"empty", git.Hash(""), false},
		{"sha1", git.Hash("a1b2c3d4e5f6789012345678901234567890abcd"), false},
		{"sha256", git.Hash("a1b2c3d4e5f6789012345678901234567890abcda1b2c3d4e5f6789012345678"), false},
		{"invalid uppercase", git.Hash("A1B2C3D4E5F6789012345678901234567890ABCD"), true},
		{"invalid abbreviated", git.Hash("a1b2c3d"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Verify it's valid JSON and can be unmarshaled
				var decoded git.Hash
				if err := json.Unmarshal(got, &decoded); err != nil {
					t.Errorf("MarshalJSON() produced invalid JSON: %v", err)
				}
				if !decoded.Equal(tt.hash) {
					t.Errorf("MarshalJSON() round-trip failed: got %+v, want %+v", decoded, tt.hash)
				}
			}
		})
	}
}

func TestHash_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    git.Hash
		wantErr bool
	}{
		{"empty", `""`, git.Hash(""), false},
		{"sha1 lowercase", `"a1b2c3d4e5f6789012345678901234567890abcd"`, git.Hash("a1b2c3d4e5f6789012345678901234567890abcd"), false},
		{"sha1 uppercase normalized", `"A1B2C3D4E5F6789012345678901234567890ABCD"`, git.Hash("a1b2c3d4e5f6789012345678901234567890abcd"), false},
		{"sha1 mixed case normalized", `"a1B2c3D4e5F6789012345678901234567890abCD"`, git.Hash("a1b2c3d4e5f6789012345678901234567890abcd"), false},
		{"sha256 lowercase", `"a1b2c3d4e5f6789012345678901234567890abcda1b2c3d4e5f6789012345678"`, git.Hash("a1b2c3d4e5f6789012345678901234567890abcda1b2c3d4e5f6789012345678"), false},
		{"sha256 uppercase normalized", `"A1B2C3D4E5F6789012345678901234567890ABCDA1B2C3D4E5F6789012345678"`, git.Hash("a1b2c3d4e5f6789012345678901234567890abcda1b2c3d4e5f6789012345678"), false},
		{"with whitespace", `"  a1b2c3d4e5f6789012345678901234567890abcd  "`, git.Hash("a1b2c3d4e5f6789012345678901234567890abcd"), false},
		{"invalid abbreviated", `"a1b2c3d"`, git.Hash(""), true},
		{"invalid char", `"g1b2c3d4e5f6789012345678901234567890abcd"`, git.Hash(""), true},
		{"invalid JSON", `not json`, git.Hash(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got git.Hash
			err := json.Unmarshal([]byte(tt.data), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.want) {
				t.Errorf("UnmarshalJSON() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHash_MarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		hash    git.Hash
		wantErr bool
	}{
		{"empty", git.Hash(""), false},
		{"sha1", git.Hash("a1b2c3d4e5f6789012345678901234567890abcd"), false},
		{"sha256", git.Hash("a1b2c3d4e5f6789012345678901234567890abcda1b2c3d4e5f6789012345678"), false},
		{"invalid uppercase", git.Hash("A1B2C3D4E5F6789012345678901234567890ABCD"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := yaml.Marshal(tt.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHash_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    git.Hash
		wantErr bool
	}{
		{"empty", `""`, git.Hash(""), false},
		{"sha1 lowercase", "a1b2c3d4e5f6789012345678901234567890abcd", git.Hash("a1b2c3d4e5f6789012345678901234567890abcd"), false},
		{"sha1 uppercase normalized", "A1B2C3D4E5F6789012345678901234567890ABCD", git.Hash("a1b2c3d4e5f6789012345678901234567890abcd"), false},
		{"sha256 lowercase", "a1b2c3d4e5f6789012345678901234567890abcda1b2c3d4e5f6789012345678", git.Hash("a1b2c3d4e5f6789012345678901234567890abcda1b2c3d4e5f6789012345678"), false},
		{"with whitespace", "  a1b2c3d4e5f6789012345678901234567890abcd  ", git.Hash("a1b2c3d4e5f6789012345678901234567890abcd"), false},
		{"invalid abbreviated", "a1b2c3d", git.Hash(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got git.Hash
			err := yaml.Unmarshal([]byte(tt.data), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.want) {
				t.Errorf("UnmarshalYAML() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseHash(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    git.Hash
		wantErr bool
	}{
		// Valid inputs
		{"empty", "", git.Hash(""), false},
		{"whitespace only", "   ", git.Hash(""), false},
		{"sha1 lowercase", "a1b2c3d4e5f6789012345678901234567890abcd", git.Hash("a1b2c3d4e5f6789012345678901234567890abcd"), false},
		{"sha1 uppercase", "A1B2C3D4E5F6789012345678901234567890ABCD", git.Hash("a1b2c3d4e5f6789012345678901234567890abcd"), false},
		{"sha1 mixed case", "a1B2c3D4e5F6789012345678901234567890abCD", git.Hash("a1b2c3d4e5f6789012345678901234567890abcd"), false},
		{"sha1 with leading whitespace", "  a1b2c3d4e5f6789012345678901234567890abcd", git.Hash("a1b2c3d4e5f6789012345678901234567890abcd"), false},
		{"sha1 with trailing whitespace", "a1b2c3d4e5f6789012345678901234567890abcd  ", git.Hash("a1b2c3d4e5f6789012345678901234567890abcd"), false},
		{"sha1 with surrounding whitespace", "  a1b2c3d4e5f6789012345678901234567890abcd  ", git.Hash("a1b2c3d4e5f6789012345678901234567890abcd"), false},
		{"sha256 lowercase", "a1b2c3d4e5f6789012345678901234567890abcda1b2c3d4e5f6789012345678", git.Hash("a1b2c3d4e5f6789012345678901234567890abcda1b2c3d4e5f6789012345678"), false},
		{"sha256 uppercase", "A1B2C3D4E5F6789012345678901234567890ABCDA1B2C3D4E5F6789012345678", git.Hash("a1b2c3d4e5f6789012345678901234567890abcda1b2c3d4e5f6789012345678"), false},

		// Invalid inputs
		{"abbreviated", "a1b2c3d", git.Hash(""), true},
		{"sha1 with invalid char", "g1b2c3d4e5f6789012345678901234567890abcd", git.Hash(""), true},
		{"sha1 with space in middle", "a1b2c3d4 e5f6789012345678901234567890abcd", git.Hash(""), true},
		{"sha1 too short", "a1b2c3d4e5f6789012345678901234567890abc", git.Hash(""), true},
		{"sha1 too long", "a1b2c3d4e5f6789012345678901234567890abcde", git.Hash(""), true},
		{"non-hex characters", "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz", git.Hash(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := git.ParseHash(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseHash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.want) {
				t.Errorf("ParseHash() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHash_JSON_RoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		original git.Hash
	}{
		{"empty", git.Hash("")},
		{"sha1", git.Hash("a1b2c3d4e5f6789012345678901234567890abcd")},
		{"sha256", git.Hash("a1b2c3d4e5f6789012345678901234567890abcda1b2c3d4e5f6789012345678")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.original)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			var decoded git.Hash
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			if !decoded.Equal(tt.original) {
				t.Errorf("JSON round-trip failed: got %q, want %q", decoded, tt.original)
			}
		})
	}
}

func TestHash_YAML_RoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		original git.Hash
	}{
		{"empty", git.Hash("")},
		{"sha1", git.Hash("a1b2c3d4e5f6789012345678901234567890abcd")},
		{"sha256", git.Hash("a1b2c3d4e5f6789012345678901234567890abcda1b2c3d4e5f6789012345678")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := yaml.Marshal(tt.original)
			if err != nil {
				t.Fatalf("yaml.Marshal() error = %v", err)
			}

			var decoded git.Hash
			if err := yaml.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("yaml.Unmarshal() error = %v", err)
			}

			if !decoded.Equal(tt.original) {
				t.Errorf("YAML round-trip failed: got %q, want %q", decoded, tt.original)
			}
		})
	}
}

func TestHash_CaseNormalization(t *testing.T) {
	// Test that uppercase input is normalized to lowercase
	input := "A1B2C3D4E5F6789012345678901234567890ABCD"
	expected := git.Hash("a1b2c3d4e5f6789012345678901234567890abcd")

	hash, err := git.ParseHash(input)
	if err != nil {
		t.Fatalf("ParseHash() error = %v", err)
	}

	if !hash.Equal(expected) {
		t.Errorf("Case normalization failed: got %q, want %q", hash, expected)
	}

	// Test that validation rejects uppercase
	uppercaseHash := git.Hash("A1B2C3D4E5F6789012345678901234567890ABCD")
	if err := uppercaseHash.Validate(); err == nil {
		t.Error("Validate() should reject uppercase hash")
	}
}

func TestHash_LengthValidation(t *testing.T) {
	tests := []struct {
		name    string
		length  int
		wantErr bool
	}{
		{"empty", 0, false},
		{"sha1", 40, false},
		{"sha256", 64, false},
		{"abbreviated 7", 7, true},
		{"abbreviated 8", 8, true},
		{"wrong 39", 39, true},
		{"wrong 41", 41, true},
		{"wrong 63", 63, true},
		{"wrong 65", 65, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hashStr string
			if tt.length > 0 {
				hashStr = strings.Repeat("a", tt.length)
			}
			hash := git.Hash(hashStr)
			err := hash.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() for length %d error = %v, wantErr %v", tt.length, err, tt.wantErr)
			}
		})
	}
}

func TestHash_HexValidation(t *testing.T) {
	tests := []struct {
		name    string
		char    rune
		wantErr bool
	}{
		{"digit 0", '0', false},
		{"digit 9", '9', false},
		{"lowercase a", 'a', false},
		{"lowercase f", 'f', false},
		{"uppercase A", 'A', true},
		{"uppercase F", 'F', true},
		{"letter g", 'g', true},
		{"letter z", 'z', true},
		{"special @", '@', true},
		{"space", ' ', true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a 40-character hash with all same character
			hashStr := strings.Repeat(string(tt.char), 40)
			hash := git.Hash(hashStr)
			err := hash.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() for char %q error = %v, wantErr %v", tt.char, err, tt.wantErr)
			}
		})
	}
}
