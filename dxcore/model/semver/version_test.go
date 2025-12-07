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

package semver_test

import (
	"encoding/json"
	"testing"

	"dirpx.dev/dxrel/dxcore/model/semver"
	"gopkg.in/yaml.v3"
)

func TestVersion_String(t *testing.T) {
	tests := []struct {
		name    string
		version semver.Version
		want    string
	}{
		{
			name:    "simple_version",
			version: semver.Version{Major: 1, Minor: 2, Patch: 3},
			want:    "1.2.3",
		},
		{
			name:    "with_prerelease",
			version: semver.Version{Major: 1, Minor: 0, Patch: 0, Prerelease: "alpha.1"},
			want:    "1.0.0-alpha.1",
		},
		{
			name:    "with_metadata",
			version: semver.Version{Major: 2, Minor: 0, Patch: 0, Metadata: "build.123"},
			want:    "2.0.0+build.123",
		},
		{
			name:    "with_prerelease_and_metadata",
			version: semver.Version{Major: 1, Minor: 0, Patch: 0, Prerelease: "rc.1", Metadata: "exp.sha.5114f85"},
			want:    "1.0.0-rc.1+exp.sha.5114f85",
		},
		{
			name:    "zero_version",
			version: semver.Version{Major: 0, Minor: 0, Patch: 0},
			want:    "0.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.version.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    semver.Version
		wantErr bool
	}{
		{
			name:  "simple_version",
			input: "1.2.3",
			want:  semver.Version{Major: 1, Minor: 2, Patch: 3},
		},
		{
			name:  "with_v_prefix",
			input: "v2.0.0",
			want:  semver.Version{Major: 2, Minor: 0, Patch: 0},
		},
		{
			name:  "with_prerelease",
			input: "1.0.0-alpha.1",
			want:  semver.Version{Major: 1, Minor: 0, Patch: 0, Prerelease: "alpha.1"},
		},
		{
			name:  "with_metadata",
			input: "1.0.0+20130313144700",
			want:  semver.Version{Major: 1, Minor: 0, Patch: 0, Metadata: "20130313144700"},
		},
		{
			name:  "with_prerelease_and_metadata",
			input: "v2.0.0-rc.1+build.123",
			want:  semver.Version{Major: 2, Minor: 0, Patch: 0, Prerelease: "rc.1", Metadata: "build.123"},
		},
		{
			name:  "complex_prerelease",
			input: "1.0.0-alpha.beta.1",
			want:  semver.Version{Major: 1, Minor: 0, Patch: 0, Prerelease: "alpha.beta.1"},
		},
		{
			name:  "zero_version",
			input: "0.0.0",
			want:  semver.Version{Major: 0, Minor: 0, Patch: 0},
		},
		{
			name:    "invalid_missing_patch",
			input:   "1.2",
			wantErr: true,
		},
		{
			name:    "invalid_non_numeric",
			input:   "1.2.x",
			wantErr: true,
		},
		{
			name:    "invalid_negative",
			input:   "1.-2.3",
			wantErr: true,
		},
		{
			name:    "invalid_empty",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid_leading_zeros",
			input:   "1.02.3",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := semver.ParseVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Major != tt.want.Major || got.Minor != tt.want.Minor || got.Patch != tt.want.Patch {
					t.Errorf("ParseVersion() = %+v, want %+v", got, tt.want)
				}
				if got.Prerelease != tt.want.Prerelease {
					t.Errorf("ParseVersion() Prerelease = %q, want %q", got.Prerelease, tt.want.Prerelease)
				}
				if got.Metadata != tt.want.Metadata {
					t.Errorf("ParseVersion() Metadata = %q, want %q", got.Metadata, tt.want.Metadata)
				}
			}
		})
	}
}

func TestVersion_Validate(t *testing.T) {
	tests := []struct {
		name    string
		version semver.Version
		wantErr bool
	}{
		{
			name:    "valid_simple",
			version: semver.Version{Major: 1, Minor: 2, Patch: 3},
			wantErr: false,
		},
		{
			name:    "valid_with_prerelease",
			version: semver.Version{Major: 1, Minor: 0, Patch: 0, Prerelease: "alpha.1"},
			wantErr: false,
		},
		{
			name:    "valid_with_metadata",
			version: semver.Version{Major: 1, Minor: 0, Patch: 0, Metadata: "build.123"},
			wantErr: false,
		},
		{
			name:    "valid_zero",
			version: semver.Version{Major: 0, Minor: 0, Patch: 0},
			wantErr: false,
		},
		{
			name:    "invalid_negative_major",
			version: semver.Version{Major: -1, Minor: 0, Patch: 0},
			wantErr: true,
		},
		{
			name:    "invalid_negative_minor",
			version: semver.Version{Major: 1, Minor: -1, Patch: 0},
			wantErr: true,
		},
		{
			name:    "invalid_negative_patch",
			version: semver.Version{Major: 1, Minor: 0, Patch: -1},
			wantErr: true,
		},
		{
			name:    "invalid_prerelease_empty_identifier",
			version: semver.Version{Major: 1, Minor: 0, Patch: 0, Prerelease: "alpha..1"},
			wantErr: true,
		},
		{
			name:    "invalid_metadata_empty_identifier",
			version: semver.Version{Major: 1, Minor: 0, Patch: 0, Metadata: "build..123"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.version.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVersion_IsZero(t *testing.T) {
	tests := []struct {
		name    string
		version semver.Version
		want    bool
	}{
		{
			name:    "zero_version",
			version: semver.Version{Major: 0, Minor: 0, Patch: 0},
			want:    true,
		},
		{
			name:    "non_zero_major",
			version: semver.Version{Major: 1, Minor: 0, Patch: 0},
			want:    false,
		},
		{
			name:    "non_zero_minor",
			version: semver.Version{Major: 0, Minor: 1, Patch: 0},
			want:    false,
		},
		{
			name:    "non_zero_patch",
			version: semver.Version{Major: 0, Minor: 0, Patch: 1},
			want:    false,
		},
		{
			name:    "with_prerelease",
			version: semver.Version{Major: 0, Minor: 0, Patch: 0, Prerelease: "alpha"},
			want:    false,
		},
		{
			name:    "with_metadata",
			version: semver.Version{Major: 0, Minor: 0, Patch: 0, Metadata: "build"},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.version.IsZero()
			if got != tt.want {
				t.Errorf("IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVersion_Compare(t *testing.T) {
	tests := []struct {
		name  string
		v1    semver.Version
		v2    semver.Version
		want  int
		desc  string
	}{
		{
			name: "equal_versions",
			v1:   semver.Version{Major: 1, Minor: 2, Patch: 3},
			v2:   semver.Version{Major: 1, Minor: 2, Patch: 3},
			want: 0,
			desc: "1.2.3 == 1.2.3",
		},
		{
			name: "major_differs",
			v1:   semver.Version{Major: 1, Minor: 0, Patch: 0},
			v2:   semver.Version{Major: 2, Minor: 0, Patch: 0},
			want: -1,
			desc: "1.0.0 < 2.0.0",
		},
		{
			name: "minor_differs",
			v1:   semver.Version{Major: 1, Minor: 1, Patch: 0},
			v2:   semver.Version{Major: 1, Minor: 2, Patch: 0},
			want: -1,
			desc: "1.1.0 < 1.2.0",
		},
		{
			name: "patch_differs",
			v1:   semver.Version{Major: 1, Minor: 0, Patch: 1},
			v2:   semver.Version{Major: 1, Minor: 0, Patch: 2},
			want: -1,
			desc: "1.0.1 < 1.0.2",
		},
		{
			name: "prerelease_vs_release",
			v1:   semver.Version{Major: 1, Minor: 0, Patch: 0, Prerelease: "alpha"},
			v2:   semver.Version{Major: 1, Minor: 0, Patch: 0},
			want: -1,
			desc: "1.0.0-alpha < 1.0.0",
		},
		{
			name: "prerelease_ordering",
			v1:   semver.Version{Major: 1, Minor: 0, Patch: 0, Prerelease: "alpha"},
			v2:   semver.Version{Major: 1, Minor: 0, Patch: 0, Prerelease: "beta"},
			want: -1,
			desc: "1.0.0-alpha < 1.0.0-beta",
		},
		{
			name: "prerelease_numeric_vs_alpha",
			v1:   semver.Version{Major: 1, Minor: 0, Patch: 0, Prerelease: "1"},
			v2:   semver.Version{Major: 1, Minor: 0, Patch: 0, Prerelease: "alpha"},
			want: -1,
			desc: "1.0.0-1 < 1.0.0-alpha (numeric < alphanumeric)",
		},
		{
			name: "prerelease_length",
			v1:   semver.Version{Major: 1, Minor: 0, Patch: 0, Prerelease: "alpha"},
			v2:   semver.Version{Major: 1, Minor: 0, Patch: 0, Prerelease: "alpha.1"},
			want: -1,
			desc: "1.0.0-alpha < 1.0.0-alpha.1",
		},
		{
			name: "metadata_ignored",
			v1:   semver.Version{Major: 1, Minor: 0, Patch: 0, Metadata: "build1"},
			v2:   semver.Version{Major: 1, Minor: 0, Patch: 0, Metadata: "build2"},
			want: 0,
			desc: "1.0.0+build1 == 1.0.0+build2 (metadata ignored)",
		},
		{
			name: "greater_version",
			v1:   semver.Version{Major: 2, Minor: 0, Patch: 0},
			v2:   semver.Version{Major: 1, Minor: 9, Patch: 9},
			want: 1,
			desc: "2.0.0 > 1.9.9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.v1.Compare(tt.v2)
			if got != tt.want {
				t.Errorf("Compare() = %d, want %d (%s)", got, tt.want, tt.desc)
			}

			// Test symmetry: if v1 < v2, then v2 > v1
			if tt.want != 0 {
				reversed := tt.v2.Compare(tt.v1)
				if reversed != -tt.want {
					t.Errorf("Compare() symmetry failed: v1.Compare(v2)=%d, v2.Compare(v1)=%d", got, reversed)
				}
			}
		})
	}
}

func TestVersion_Less_Equal_Greater(t *testing.T) {
	v1 := semver.Version{Major: 1, Minor: 0, Patch: 0}
	v2 := semver.Version{Major: 1, Minor: 0, Patch: 0}
	v3 := semver.Version{Major: 2, Minor: 0, Patch: 0}

	// Test Equal
	if !v1.Equal(v2) {
		t.Errorf("Equal() failed: %v should equal %v", v1, v2)
	}
	if v1.Equal(v3) {
		t.Errorf("Equal() failed: %v should not equal %v", v1, v3)
	}

	// Test Less
	if !v1.Less(v3) {
		t.Errorf("Less() failed: %v should be less than %v", v1, v3)
	}
	if v3.Less(v1) {
		t.Errorf("Less() failed: %v should not be less than %v", v3, v1)
	}

	// Test Greater
	if !v3.Greater(v1) {
		t.Errorf("Greater() failed: %v should be greater than %v", v3, v1)
	}
	if v1.Greater(v3) {
		t.Errorf("Greater() failed: %v should not be greater than %v", v1, v3)
	}
}

func TestVersion_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		version semver.Version
		want    string
		wantErr bool
	}{
		{
			name:    "simple_version",
			version: semver.Version{Major: 1, Minor: 2, Patch: 3},
			want:    `"1.2.3"`,
		},
		{
			name:    "with_prerelease",
			version: semver.Version{Major: 1, Minor: 0, Patch: 0, Prerelease: "alpha.1"},
			want:    `"1.0.0-alpha.1"`,
		},
		{
			name:    "with_metadata",
			version: semver.Version{Major: 1, Minor: 0, Patch: 0, Metadata: "build.123"},
			want:    `"1.0.0+build.123"`,
		},
		{
			name:    "invalid_negative",
			version: semver.Version{Major: -1, Minor: 0, Patch: 0},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.version)
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

func TestVersion_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    semver.Version
		wantErr bool
	}{
		{
			name: "simple_version",
			json: `"1.2.3"`,
			want: semver.Version{Major: 1, Minor: 2, Patch: 3},
		},
		{
			name: "with_v_prefix",
			json: `"v2.0.0"`,
			want: semver.Version{Major: 2, Minor: 0, Patch: 0},
		},
		{
			name: "with_prerelease",
			json: `"1.0.0-alpha.1"`,
			want: semver.Version{Major: 1, Minor: 0, Patch: 0, Prerelease: "alpha.1"},
		},
		{
			name: "with_metadata",
			json: `"1.0.0+build.123"`,
			want: semver.Version{Major: 1, Minor: 0, Patch: 0, Metadata: "build.123"},
		},
		{
			name:    "invalid_format",
			json:    `"1.2"`,
			wantErr: true,
		},
		{
			name:    "invalid_json",
			json:    `not-json`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got semver.Version
			err := json.Unmarshal([]byte(tt.json), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Major != tt.want.Major || got.Minor != tt.want.Minor || got.Patch != tt.want.Patch {
					t.Errorf("UnmarshalJSON() = %+v, want %+v", got, tt.want)
				}
				if got.Prerelease != tt.want.Prerelease {
					t.Errorf("UnmarshalJSON() Prerelease = %q, want %q", got.Prerelease, tt.want.Prerelease)
				}
				if got.Metadata != tt.want.Metadata {
					t.Errorf("UnmarshalJSON() Metadata = %q, want %q", got.Metadata, tt.want.Metadata)
				}
			}
		})
	}
}

func TestVersion_MarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		version semver.Version
		want    string
		wantErr bool
	}{
		{
			name:    "simple_version",
			version: semver.Version{Major: 1, Minor: 2, Patch: 3},
			want:    "1.2.3\n",
		},
		{
			name:    "with_prerelease",
			version: semver.Version{Major: 1, Minor: 0, Patch: 0, Prerelease: "alpha.1"},
			want:    "1.0.0-alpha.1\n",
		},
		{
			name:    "with_metadata",
			version: semver.Version{Major: 1, Minor: 0, Patch: 0, Metadata: "build.123"},
			want:    "1.0.0+build.123\n",
		},
		{
			name:    "invalid_negative",
			version: semver.Version{Major: -1, Minor: 0, Patch: 0},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := yaml.Marshal(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(got) != tt.want {
				t.Errorf("MarshalYAML() = %q, want %q", string(got), tt.want)
			}
		})
	}
}

func TestVersion_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		want    semver.Version
		wantErr bool
	}{
		{
			name: "simple_version",
			yaml: "1.2.3",
			want: semver.Version{Major: 1, Minor: 2, Patch: 3},
		},
		{
			name: "with_v_prefix",
			yaml: "v2.0.0",
			want: semver.Version{Major: 2, Minor: 0, Patch: 0},
		},
		{
			name: "with_prerelease",
			yaml: "1.0.0-alpha.1",
			want: semver.Version{Major: 1, Minor: 0, Patch: 0, Prerelease: "alpha.1"},
		},
		{
			name: "with_metadata",
			yaml: "1.0.0+build.123",
			want: semver.Version{Major: 1, Minor: 0, Patch: 0, Metadata: "build.123"},
		},
		{
			name:    "invalid_format",
			yaml:    "1.2",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got semver.Version
			err := yaml.Unmarshal([]byte(tt.yaml), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Major != tt.want.Major || got.Minor != tt.want.Minor || got.Patch != tt.want.Patch {
					t.Errorf("UnmarshalYAML() = %+v, want %+v", got, tt.want)
				}
				if got.Prerelease != tt.want.Prerelease {
					t.Errorf("UnmarshalYAML() Prerelease = %q, want %q", got.Prerelease, tt.want.Prerelease)
				}
				if got.Metadata != tt.want.Metadata {
					t.Errorf("UnmarshalYAML() Metadata = %q, want %q", got.Metadata, tt.want.Metadata)
				}
			}
		})
	}
}

func TestVersion_RoundTrip_JSON(t *testing.T) {
	versions := []semver.Version{
		{Major: 1, Minor: 2, Patch: 3},
		{Major: 1, Minor: 0, Patch: 0, Prerelease: "alpha.1"},
		{Major: 1, Minor: 0, Patch: 0, Metadata: "build.123"},
		{Major: 1, Minor: 0, Patch: 0, Prerelease: "rc.1", Metadata: "exp.sha.5114f85"},
	}

	for _, v := range versions {
		t.Run(v.String(), func(t *testing.T) {
			data, err := json.Marshal(v)
			if err != nil {
				t.Fatalf("Marshal() failed: %v", err)
			}

			var decoded semver.Version
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal() failed: %v", err)
			}

			if decoded.Major != v.Major || decoded.Minor != v.Minor || decoded.Patch != v.Patch {
				t.Errorf("Round-trip failed: got %+v, want %+v", decoded, v)
			}
			if decoded.Prerelease != v.Prerelease {
				t.Errorf("Round-trip Prerelease failed: got %q, want %q", decoded.Prerelease, v.Prerelease)
			}
			if decoded.Metadata != v.Metadata {
				t.Errorf("Round-trip Metadata failed: got %q, want %q", decoded.Metadata, v.Metadata)
			}
		})
	}
}

func TestVersion_RoundTrip_YAML(t *testing.T) {
	versions := []semver.Version{
		{Major: 1, Minor: 2, Patch: 3},
		{Major: 1, Minor: 0, Patch: 0, Prerelease: "alpha.1"},
		{Major: 1, Minor: 0, Patch: 0, Metadata: "build.123"},
		{Major: 1, Minor: 0, Patch: 0, Prerelease: "rc.1", Metadata: "exp.sha.5114f85"},
	}

	for _, v := range versions {
		t.Run(v.String(), func(t *testing.T) {
			data, err := yaml.Marshal(v)
			if err != nil {
				t.Fatalf("Marshal() failed: %v", err)
			}

			var decoded semver.Version
			if err := yaml.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal() failed: %v", err)
			}

			if decoded.Major != v.Major || decoded.Minor != v.Minor || decoded.Patch != v.Patch {
				t.Errorf("Round-trip failed: got %+v, want %+v", decoded, v)
			}
			if decoded.Prerelease != v.Prerelease {
				t.Errorf("Round-trip Prerelease failed: got %q, want %q", decoded.Prerelease, v.Prerelease)
			}
			if decoded.Metadata != v.Metadata {
				t.Errorf("Round-trip Metadata failed: got %q, want %q", decoded.Metadata, v.Metadata)
			}
		})
	}
}
