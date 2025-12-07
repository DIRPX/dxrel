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

package semver

import (
	"encoding/json"
	"fmt"
	"strings"

	dxerrors "dirpx.dev/dxrel/dxcore/errors"
	bsemver "github.com/blang/semver/v4"

	"gopkg.in/yaml.v3"
)

// Version represents a semantic version according to Semantic Versioning 2.0.0
// specification (https://semver.org), as used by dxrel when computing and
// comparing module releases.
//
// This implementation wraps github.com/blang/semver/v4 to provide full SemVer 2.0.0
// compliance while maintaining a clean, dxrel-specific API.
//
// Version supports the full SemVer 2.0.0 format: Major.Minor.Patch[-Prerelease][+Metadata]
//
// Components:
//   - Major, Minor, Patch: Non-negative integers for version core
//   - Prerelease: Optional pre-release identifier (e.g., "-alpha.1", "-rc.2")
//   - Metadata: Optional build metadata (e.g., "+build.123", "+20130313144700")
//
// Ordering and comparison follow SemVer 2.0.0 rules:
//   - Versions with prerelease have lower precedence than release versions:
//     1.0.0-alpha < 1.0.0
//   - Prerelease identifiers are compared lexicographically by dot-separated parts:
//     1.0.0-alpha < 1.0.0-alpha.1 < 1.0.0-beta
//   - Build metadata does NOT affect version precedence:
//     1.0.0+build1 == 1.0.0+build2
//
// The zero value of Version corresponds to 0.0.0 and may be used as a sentinel
// for "no version" or "version not yet computed". dxrel typically derives
// concrete versions from existing Git tags. Callers SHOULD treat Major, Minor,
// and Patch as non-negative integers; negative values are considered invalid
// and MUST NOT be produced by normal dxrel code.
type Version struct {
	// Major is the first component of the semantic version.
	//
	// Incrementing Major indicates a breaking change according to semantic
	// versioning rules. Once a module has published a non-zero Major version,
	// dxrel MUST respect Major increments as the highest-precedence change
	// when computing new versions from Conventional Commits.
	Major int

	// Minor is the second component of the semantic version.
	//
	// Incrementing Minor indicates the addition of backwards-compatible
	// functionality. When no Major bump is required, dxrel increments Minor
	// in response to feature-level changes (for example, "feat:" commits) as
	// defined by the configured Conventional Commit mapping.
	Minor int

	// Patch is the third component of the semantic version.
	//
	// Incrementing Patch indicates backwards-compatible bug fixes or internal
	// changes that do not affect the public API surface. When neither Major
	// nor Minor increments are required, dxrel increments Patch in response
	// to fix-level or maintenance changes (for example, "fix:" commits),
	// depending on configuration.
	Patch int

	// Prerelease is an optional pre-release identifier according to SemVer 2.0.0.
	//
	// When non-empty, it MUST be a series of dot-separated identifiers containing
	// only ASCII alphanumerics and hyphens [0-9A-Za-z-]. Identifiers MUST NOT be
	// empty. Numeric identifiers MUST NOT include leading zeroes.
	//
	// A version with a non-empty Prerelease has lower precedence than the same
	// version without prerelease. For example: 1.0.0-alpha < 1.0.0
	//
	// Examples: "alpha", "alpha.1", "rc.1", "beta.2", "0.3.7"
	Prerelease string

	// Metadata is optional build metadata according to SemVer 2.0.0.
	//
	// When non-empty, it MUST be a series of dot-separated identifiers containing
	// only ASCII alphanumerics and hyphens [0-9A-Za-z-]. Identifiers MUST NOT be
	// empty.
	//
	// Build metadata SHOULD be ignored when determining version precedence. Two
	// versions that differ only in build metadata have the same precedence.
	//
	// Examples: "build.123", "20130313144700", "exp.sha.5114f85"
	//
	// Note: This field was previously named "Build" in earlier versions of dxrel.
	// It has been renamed to "Metadata" to better reflect SemVer 2.0.0 terminology
	// and avoid confusion with build processes.
	Metadata string
}

// ParseVersion parses a SemVer 2.0.0 version string into a Version value.
//
// This function uses github.com/blang/semver/v4 internally to ensure full
// SemVer 2.0.0 compliance and correctness.
//
// The expected input format is "Major.Minor.Patch[-Prerelease][+Metadata]", where:
//   - Major, Minor, Patch are non-negative integers
//   - Prerelease is an optional pre-release identifier after '-'
//   - Metadata is optional build metadata after '+'
//   - An optional leading "v" is tolerated and stripped
//
// Examples:
//
//	ParseVersion("1.2.3")  -> Version{Major: 1, Minor: 2, Patch: 3}
//	ParseVersion("v2.0.0") -> Version{Major: 2, Minor: 0, Patch: 0}
//	ParseVersion("1.0.0-alpha.1") -> Version{1, 0, 0, "alpha.1", ""}
//	ParseVersion("v2.0.0-rc.1+build.123") -> Version{2, 0, 0, "rc.1", "build.123"}
//	ParseVersion("1.0.0+20130313144700") -> Version{1, 0, 0, "", "20130313144700"}
//
// On error (wrong format, non-integer parts, negative components, or invalid
// prerelease/metadata identifiers), ParseVersion returns a zero Version and a
// descriptive error. Callers MUST check the error before using the returned value.
func ParseVersion(s string) (Version, error) {
	// Strip leading 'v' if present (blang/semver doesn't handle this)
	s = strings.TrimPrefix(s, "v")

	// Parse using blang/semver
	bv, err := bsemver.Parse(s)
	if err != nil {
		return Version{}, fmt.Errorf("invalid version format %q: %w", s, err)
	}

	// Convert to our Version type
	return fromBlangSemver(bv), nil
}

// String returns the canonical textual representation of the Version according
// to SemVer 2.0.0.
//
// The format is "Major.Minor.Patch[-Prerelease][+Metadata]" with the numeric
// components rendered as decimal integers. Prerelease and Metadata are included
// only when non-empty.
//
// Examples:
//
//	Version{Major: 1, Minor: 2, Patch: 3}.String()
//	// Output: "1.2.3"
//
//	Version{Major: 1, Minor: 0, Patch: 0, Prerelease: "alpha.1"}.String()
//	// Output: "1.0.0-alpha.1"
//
//	Version{Major: 2, Minor: 0, Patch: 0, Metadata: "build.123"}.String()
//	// Output: "2.0.0+build.123"
//
//	Version{Major: 1, Minor: 0, Patch: 0, Prerelease: "rc.1", Metadata: "exp.sha.5114f85"}.String()
//	// Output: "1.0.0-rc.1+exp.sha.5114f85"
func (v Version) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Prerelease != "" {
		s += "-" + v.Prerelease
	}
	if v.Metadata != "" {
		s += "+" + v.Metadata
	}
	return s
}

// toBlangSemver converts this Version to a blang/semver.Version for internal operations.
//
// This is a helper method used by ParseVersion, Validate, and Compare to leverage
// the battle-tested blang/semver implementation for parsing and comparison logic.
func (v Version) toBlangSemver() (bsemver.Version, error) {
	// Build the version string
	vstr := v.String()

	// Parse using blang/semver
	bv, err := bsemver.Parse(vstr)
	if err != nil {
		return bsemver.Version{}, fmt.Errorf("failed to convert to blang/semver: %w", err)
	}

	return bv, nil
}

// fromBlangSemver creates a Version from a blang/semver.Version.
//
// This is a helper method used to convert parsed blang/semver versions back
// to our Version type.
func fromBlangSemver(bv bsemver.Version) Version {
	// Extract prerelease as string
	var prerelease string
	if len(bv.Pre) > 0 {
		parts := make([]string, len(bv.Pre))
		for i, p := range bv.Pre {
			parts[i] = p.String()
		}
		prerelease = strings.Join(parts, ".")
	}

	// Extract build metadata as string
	var metadata string
	if len(bv.Build) > 0 {
		metadata = strings.Join(bv.Build, ".")
	}

	return Version{
		Major:      int(bv.Major),
		Minor:      int(bv.Minor),
		Patch:      int(bv.Patch),
		Prerelease: prerelease,
		Metadata:   metadata,
	}
}

// Validate checks that the Version components are well-formed and
// semantically acceptable for use in dxrel according to SemVer 2.0.0.
//
// This method uses github.com/blang/semver/v4 internally to ensure full
// SemVer 2.0.0 validation compliance.
//
// Validation enforces:
//   - Major, Minor, and Patch are non-negative
//   - Prerelease (if non-empty) contains only valid SemVer 2.0.0 identifiers:
//     dot-separated, each containing only [0-9A-Za-z-], no empty identifiers
//   - Metadata (if non-empty) contains only valid SemVer 2.0.0 identifiers:
//     dot-separated, each containing only [0-9A-Za-z-], no empty identifiers
//
// This method is intended for use at boundaries such as deserialization
// or before emitting a version into user-facing output.
func (v Version) Validate() error {
	// Check for negative values (blang/semver uses uint64, so we need this check)
	if v.Major < 0 {
		return fmt.Errorf("Major version component must be non-negative, got %d", v.Major)
	}
	if v.Minor < 0 {
		return fmt.Errorf("Minor version component must be non-negative, got %d", v.Minor)
	}
	if v.Patch < 0 {
		return fmt.Errorf("Patch version component must be non-negative, got %d", v.Patch)
	}

	// Use blang/semver for validation
	_, err := v.toBlangSemver()
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}

// IsZero reports whether the Version is exactly 0.0.0 with no prerelease
// or build metadata.
//
// This is useful for distinguishing between an explicit, meaningful version
// and an uninitialized or "no releases yet" baseline value in higher-level
// logic.
//
// Note: "0.0.0-alpha" or "0.0.0+build" are NOT considered zero because they
// carry semantic meaning beyond the numeric core.
func (v Version) IsZero() bool {
	return v.Major == 0 && v.Minor == 0 && v.Patch == 0 && v.Prerelease == "" && v.Metadata == ""
}

// Compare compares v with other and reports their ordering according to
// SemVer 2.0.0 precedence rules.
//
// This method uses github.com/blang/semver/v4 internally to ensure correct
// SemVer 2.0.0 comparison semantics.
//
// It returns:
//   - -1 if v <  other
//   - 0 if v == other
//   - +1 if v >  other
//
// Ordering follows SemVer 2.0.0 rules:
//  1. Major, Minor, and Patch are compared numerically
//  2. Versions with prerelease have LOWER precedence than release versions:
//     1.0.0-alpha < 1.0.0
//  3. Prerelease identifiers are compared lexicographically by dot-separated parts:
//     - Numeric identifiers are compared as integers: 1 < 2 < 10
//     - Alphanumeric identifiers are compared lexicographically: alpha < beta
//     - Numeric identifiers have lower precedence than alphanumeric: 1 < alpha
//     - Larger sets of identifiers have higher precedence if all else equal:
//     1.0.0-alpha < 1.0.0-alpha.1
//  4. Build metadata does NOT affect precedence and is ignored
//
// Examples:
//
//	1.0.0-alpha < 1.0.0-alpha.1 < 1.0.0-alpha.beta < 1.0.0-beta < 1.0.0-beta.2 < 1.0.0
func (v Version) Compare(other Version) int {
	// Convert both versions to blang/semver
	bv, err := v.toBlangSemver()
	if err != nil {
		// If conversion fails, fall back to simple comparison
		// This shouldn't happen if both versions are valid
		if v.Major != other.Major {
			if v.Major < other.Major {
				return -1
			}
			return 1
		}
		if v.Minor != other.Minor {
			if v.Minor < other.Minor {
				return -1
			}
			return 1
		}
		if v.Patch != other.Patch {
			if v.Patch < other.Patch {
				return -1
			}
			return 1
		}
		return 0
	}

	bother, err := other.toBlangSemver()
	if err != nil {
		// Same fallback for other version
		if v.Major != other.Major {
			if v.Major < other.Major {
				return -1
			}
			return 1
		}
		if v.Minor != other.Minor {
			if v.Minor < other.Minor {
				return -1
			}
			return 1
		}
		if v.Patch != other.Patch {
			if v.Patch < other.Patch {
				return -1
			}
			return 1
		}
		return 0
	}

	// Use blang/semver comparison
	return bv.Compare(bother)
}

// Less reports whether v is strictly less than other according to
// semantic versioning ordering (Major, then Minor, then Patch).
func (v Version) Less(other Version) bool {
	return v.Compare(other) < 0
}

// Equal reports whether v and other represent the same semantic version.
//
// Note: Per SemVer 2.0.0, build metadata is ignored for comparison purposes.
// Thus, 1.0.0+build1 equals 1.0.0+build2.
func (v Version) Equal(other Version) bool {
	return v.Compare(other) == 0
}

// Greater reports whether v is strictly greater than other according to
// semantic versioning ordering (Major, then Minor, then Patch).
func (v Version) Greater(other Version) bool {
	return v.Compare(other) > 0
}

// MarshalJSON implements json.Marshaler for Version.
//
// A valid Version is serialized as a JSON string in "Major.Minor.Patch"
// format (for example, "1.2.3"). Before encoding, MarshalJSON calls
// Validate; if the Version is not well-formed, it returns the validation
// error and produces no JSON output.
func (v Version) MarshalJSON() ([]byte, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(v.String())
}

// UnmarshalJSON implements json.Unmarshaler for Version.
//
// The method expects the JSON value to be a string in "Major.Minor.Patch"
// form, optionally prefixed with "v" (for example, "1.2.3" or "v1.2.3").
// The string is parsed via ParseVersion, and any parse error is returned
// directly to the caller.
func (v *Version) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return &dxerrors.UnmarshalError{
			Type:   "Version",
			Data:   data,
			Reason: err.Error(),
		}
	}

	parsed, err := ParseVersion(s)
	if err != nil {
		return err
	}

	*v = parsed
	return nil
}

// MarshalYAML implements yaml.Marshaler for Version.
//
// A valid Version is serialized as a scalar string in "Major.Minor.Patch"
// format. Validation is performed before encoding; if the Version is not
// well-formed, the validation error is returned and no YAML value is
// produced.
func (v Version) MarshalYAML() (interface{}, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	return v.String(), nil
}

// UnmarshalYAML implements yaml.Unmarshaler for Version.
//
// The YAML value is expected to be a scalar string in "Major.Minor.Patch"
// form, optionally prefixed with "v". The string is parsed via ParseVersion.
// Any parse error is returned to the caller, and in that case the Version
// MUST NOT be used.
func (v *Version) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return &dxerrors.UnmarshalError{
			Type:   "Version",
			Data:   nil,
			Reason: err.Error(),
		}
	}

	parsed, err := ParseVersion(s)
	if err != nil {
		return err
	}

	*v = parsed
	return nil
}
