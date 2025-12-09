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

	"dirpx.dev/dxrel/dxcore/errors"
	"dirpx.dev/dxrel/dxcore/model"
	"gopkg.in/yaml.v3"
)

// Bump represents the concrete semantic version increment that dxrel will
// apply to a module's current version when producing a new release.
//
// While ChangeKind describes the semantic impact of a range of changes
// (ignore/patch/minor/major) in abstract terms, Bump encodes the actionable
// decision: whether to leave the version unchanged or to increment a specific
// component (BumpPatch, BumpMinor, or BumpMajor). In other words, ChangeKind
// is about classification of changes, whereas Bump is about the operation to
// perform.
//
// A Bump value is typically computed per module after aggregating all relevant
// commits and applying repository-specific policies (for example, "no major
// releases while still in v0.x"). Once computed, Bump is applied to a Version
// to derive the next release version for that module.
type Bump int

const (
	// BumpNone indicates that no version change should be performed.
	//
	// dxrel uses this value when all changes affecting a module are deemed
	// ignorable for versioning purposes (for example, documentation-only
	// updates) or when policy explicitly suppresses a version bump despite
	// observed changes. When BumpNone is selected, no new tag SHOULD be
	// created for the module.
	BumpNone Bump = iota

	// BumpPatch indicates that the Patch component of the version should be
	// incremented.
	//
	// This is used for backwards-compatible bug fixes, performance tweaks,
	// or other changes that do not add new features and do not break the
	// public API. When applied to a version X.Y.Z, BumpPatch yields X.Y.(Z+1),
	// assuming no higher-precedence bump (BumpMinor or BumpMajor) has been
	// requested.
	BumpPatch

	// BumpMinor indicates that the Minor component of the version should be
	// incremented and the Patch component reset to zero.
	//
	// This is used for backwards-compatible feature additions or other
	// additive changes. When applied to a version X.Y.Z, BumpMinor yields
	// X.(Y+1).0, assuming that a BumpMajor bump is not required or disallowed
	// by policy.
	BumpMinor

	// BumpMajor indicates that the Major component of the version should be
	// incremented and both Minor and Patch components reset to zero.
	//
	// This is the highest-precedence bump and is used for breaking changes or
	// other incompatible modifications according to semantic versioning rules.
	// When applied to a version X.Y.Z, BumpMajor yields (X+1).0.0, provided
	// that the module's version policy permits major releases.
	BumpMajor
)

// String constants for Bump values used in serialization, parsing, and
// human-facing output.
//
// These names form the stable, external representation of Bump and MAY be
// persisted in configuration files, CLI flags, and JSON/YAML documents.
// Changing them is a breaking change for any consumer that relies on textual
// configuration.
const (
	BumpNoneStr  = "none"
	BumpPatchStr = "patch"
	BumpMinorStr = "minor"
	BumpMajorStr = "major"
)

// ParseBump converts a textual representation into a Bump value.
//
// The function accepts a small, case-insensitive vocabulary of strings and
// maps them to the corresponding constants:
//
//	"none",  "None",  "NONE"  -> BumpNone
//	"patch", "Patch", "PATCH" -> BumpPatch
//	"minor", "Minor", "MINOR" -> BumpMinor
//	"major", "Major", "MAJOR" -> BumpMajor
//
// Any other input is treated as invalid, and ParseBump returns a *ParseError.
// The returned error includes the original string value, which can be used in
// diagnostics or surfaced back to the user.
func ParseBump(s string) (Bump, error) {
	switch s {
	case BumpNoneStr, "None", "NONE":
		return BumpNone, nil
	case BumpPatchStr, "Patch", "PATCH":
		return BumpPatch, nil
	case BumpMinorStr, "Minor", "MINOR":
		return BumpMinor, nil
	case BumpMajorStr, "Major", "MAJOR":
		return BumpMajor, nil
	default:
		return BumpNone, &errors.ParseError{Type: "Bump", Value: s}
	}
}

// String returns the canonical string representation of the Bump value.
//
// The returned value is always lowercase and suitable for use in
// configuration files, command-line flags, logs, and API responses.
// The mapping is:
//
//	BumpNone  -> "none"
//	BumpPatch -> "patch"
//	BumpMinor -> "minor"
//	BumpMajor -> "major"
//
// If the Bump value is not one of the defined constants, String returns
// "unknown". Callers that need to ensure only valid values are emitted SHOULD
// call Valid before invoking String, or treat "unknown" as an indicator of a
// configuration or programming error.
func (b Bump) String() string {
	switch b {
	case BumpNone:
		return BumpNoneStr
	case BumpPatch:
		return BumpPatchStr
	case BumpMinor:
		return BumpMinorStr
	case BumpMajor:
		return BumpMajorStr
	default:
		return "unknown"
	}
}

// Valid reports whether the Bump value is one of the defined constants.
//
// This method is primarily useful when Bump values may have been created via
// deserialization, numeric casts, or untrusted input. Code that relies on
// Bump being well-formed SHOULD call Valid before using the value in logic
// that assumes a known semantic meaning.
func (b Bump) Valid() bool {
	return b == BumpNone || b == BumpPatch || b == BumpMinor || b == BumpMajor
}

// TypeName returns "Bump", the name of the type for logging and debugging.
//
// This method implements part of the model.Model interface, allowing Bump
// values to be used consistently with other model types in error messages,
// logs, and reflection-based code.
func (b Bump) TypeName() string {
	return "Bump"
}

// Redacted returns the same string representation as String().
//
// Bump values contain no sensitive information (they are simply enum
// constants), so the redacted form is identical to the regular string form.
// This method implements part of the model.Model interface.
func (b Bump) Redacted() string {
	return b.String()
}

// IsZero reports whether the Bump has its zero value.
//
// For Bump (an enum type), the zero value is BumpNone (constant 0).
// This method implements part of the model.Model interface and is useful
// when checking if a Bump field was explicitly set or left at its
// default value.
//
// Note: The zero value (BumpNone) is a valid Bump, so IsZero returning
// true does not indicate an error condition.
func (b Bump) IsZero() bool {
	return b == BumpNone
}

// Equal reports whether this Bump is equal to another value.
//
// The method accepts any type for other and uses type assertion to check if
// it is a Bump or *Bump. Two Bump values are equal if they represent the same
// enum constant.
//
// This method implements part of the model.Model interface and is useful
// for comparisons in tests and validation logic.
func (b Bump) Equal(other any) bool {
	switch v := other.(type) {
	case Bump:
		return b == v
	case *Bump:
		if v == nil {
			return false
		}
		return b == *v
	default:
		return false
	}
}

// Validate checks whether the Bump value is one of the defined constants.
//
// This method returns nil if the Bump is valid (BumpNone, BumpPatch,
// BumpMinor, or BumpMajor), and returns a *ValidationError if the value is
// outside the valid range.
//
// This method implements part of the model.Model interface and is typically
// called after deserialization or numeric casts to ensure the value is
// well-formed before using it in planning logic.
func (b Bump) Validate() error {
	if !b.Valid() {
		return &errors.ValidationError{
			Type:   "Bump",
			Field:  "",
			Reason: "invalid Bump value",
			Value:  int(b),
		}
	}
	return nil
}

// MarshalJSON implements json.Marshaler for Bump.
//
// A valid Bump is serialized as its lowercase string representation
// (for example, "patch" or "minor"). If the value is not valid, MarshalJSON
// returns a *MarshalError and does not produce any JSON output.
//
// This behavior ensures that invalid Bump values do not silently appear in
// JSON payloads and instead surface as explicit failures during encoding.
func (b Bump) MarshalJSON() ([]byte, error) {
	if !b.Valid() {
		return nil, &errors.MarshalError{Type: "Bump", Value: int(b)}
	}
	return []byte(`"` + b.String() + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler for Bump.
//
// The method accepts both string and numeric JSON representations:
//
//   - String: "none", "patch", "minor", "major" (case-insensitive variants
//     accepted via ParseBump).
//
//   - Number: 0 (BumpNone), 1 (BumpPatch), 2 (BumpMinor), 3 (BumpMajor).
//
// String input is the preferred, stable representation. Numeric input is
// accepted for compatibility with configurations that store enum values as
// integers. If the input cannot be parsed as either string or number, or if
// it resolves to an invalid Bump, UnmarshalJSON returns an *UnmarshalError
// describing the failure.
func (b *Bump) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return &errors.UnmarshalError{Type: "Bump", Data: data, Reason: "empty data"}
	}

	// Try string format first.
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return &errors.UnmarshalError{Type: "Bump", Data: data, Reason: err.Error()}
		}
		parsed, err := ParseBump(s)
		if err != nil {
			return err
		}
		*b = parsed
		return nil
	}

	// Fallback to numeric format.
	var i int
	if err := json.Unmarshal(data, &i); err != nil {
		return &errors.UnmarshalError{Type: "Bump", Data: data, Reason: err.Error()}
	}
	*b = Bump(i)
	if !b.Valid() {
		return &errors.UnmarshalError{Type: "Bump", Data: data, Reason: "invalid numeric value"}
	}
	return nil
}

// MarshalYAML implements yaml.Marshaler for Bump.
//
// A valid Bump is serialized as its canonical string representation
// (for example, "patch"). If the value is not valid, MarshalYAML
// returns a *MarshalError.
func (b Bump) MarshalYAML() (any, error) {
	if !b.Valid() {
		return nil, &errors.MarshalError{Type: "Bump", Value: int(b)}
	}
	return b.String(), nil
}

// UnmarshalYAML implements yaml.Unmarshaler for Bump.
//
// The method accepts string representations of Bump values
// (for example, "patch", "minor") and resolves them via ParseBump.
// On failure, it returns the underlying *ParseError.
func (b *Bump) UnmarshalYAML(node *yaml.Node) error {
	var str string
	if err := node.Decode(&str); err != nil {
		return &errors.UnmarshalError{Type: "Bump", Data: []byte(node.Value), Reason: err.Error()}
	}
	parsed, err := ParseBump(str)
	if err != nil {
		return err
	}
	*b = parsed
	return nil
}

// MarshalText implements encoding.TextMarshaler for Bump.
//
// Textual form is the same lowercase string representation as used by
// String() (for example, "patch", "minor"). This encoding is commonly used
// by YAML and other text-based configuration formats. If the Bump value is
// invalid, MarshalText returns a *MarshalError.
func (b Bump) MarshalText() ([]byte, error) {
	if !b.Valid() {
		return nil, &errors.MarshalError{Type: "Bump", Value: int(b)}
	}
	return []byte(b.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler for Bump.
//
// The method accepts the same textual vocabulary as ParseBump, using it as
// the single source of truth for mapping strings to Bump values. The input
// is treated case-insensitively in the same way as ParseBump. On failure,
// UnmarshalText returns the underlying *ParseError.
func (b *Bump) UnmarshalText(text []byte) error {
	parsed, err := ParseBump(string(text))
	if err != nil {
		return err
	}
	*b = parsed
	return nil
}

// Compile-time check that Bump implements model.Model interface.
var _ model.Model = (*Bump)(nil)
