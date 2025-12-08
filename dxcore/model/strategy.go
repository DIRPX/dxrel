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

	"dirpx.dev/dxrel/dxcore/errors"
	"gopkg.in/yaml.v3"
)

// Strategy controls how dxrel derives the next semantic version for a module
// from a sequence of commits.
//
// Conceptually, dxrel can interpret a range of commits (for example, from the
// last tag to HEAD) in two different ways:
//
//  1. As a single batch of changes, where only the maximum observed impact
//     (patch, minor, major) matters for the next release. This is analogous
//     to tools that compute a single bump from the "worst" commit in the
//     range and then cut one release.
//
//  2. As a chronological sequence of incremental releases, where each commit
//     is applied in order and the version is updated step by step, as if a
//     release could have been cut after each change. The final version is
//     the result of folding all bumps across the range.
//
// Strategy encapsulates this choice so that different repositories or modules
// can opt into the behavior that best matches their release philosophy.
type Strategy int

const (
	// MaxSeverity computes a single version bump from the maximum
	// severity of all commits in the analyzed range.
	//
	// With this strategy, dxrel first classifies each commit (for example, as
	// patch, minor, or major), then determines the highest-impact change and
	// applies exactly one bump of that kind to the last known version. This
	// produces behavior similar to semantic-release and other tools that treat
	// an entire range of commits as one release unit.
	//
	// Example:
	//   LastVersion = 1.0.0
	//   Commits = feat, fix, fix, feat!, fix
	//   Max impact = major
	//   NewVersion = 2.0.0
	MaxSeverity Strategy = iota

	// Sequential applies bump rules for each commit in chronological
	// order, updating the version step by step.
	//
	// With this strategy, dxrel simulates the effect of cutting a release
	// after every commit in the range: it starts from the last known version,
	// processes commits from oldest to newest, and for each commit applies the
	// corresponding bump (if any). The final version after all commits have
	// been processed becomes the new release version for the module.
	//
	// This approach can yield different results from MaxSeverity in
	// scenarios where lower-severity changes follow higher-severity ones,
	// because later commits may increment Patch or Minor on top of an already
	// bumped Major or Minor.
	//
	// Example:
	//   LastVersion = 1.0.0
	//   Commits = feat, fix, fix, feat!, fix
	//   Stepwise:
	//     1.0.0 --feat--> 1.1.0
	//     1.1.0 --fix--> 1.1.1
	//     1.1.1 --fix--> 1.1.2
	//     1.1.2 --feat!--> 2.0.0
	//     2.0.0 --fix--> 2.0.1
	//   NewVersion = 2.0.1
	Sequential
)

// Compile-time check that Strategy implements model.Model interface.
var _ Model = (*Strategy)(nil)

// String constants for Strategy values used in serialization, parsing,
// and human-facing output.
//
// These constants define the canonical external representation of Strategy
// and MAY be used in configuration files, CLI flags, and JSON/YAML payloads.
// Changing any of these strings is a breaking change for consumers that rely
// on textual configuration.
const (
	MaxSeverityStr = "max-severity"
	SequentialStr  = "sequential"
)

// String returns the canonical string representation of the Strategy value.
//
// The returned string is always in lowercase kebab-case and is suitable for
// use in configuration files, command-line flags, logs, and API responses.
// The mapping is:
//
//	MaxSeverity -> "max-severity"
//	Sequential  -> "sequential"
//
// If the Strategy value is not one of the defined constants, String returns
// "unknown". Callers that require only valid Strategy values SHOULD either
// check Valid before calling String or treat "unknown" as an indicator of a
// configuration or programming error.
func (s Strategy) String() string {
	switch s {
	case MaxSeverity:
		return MaxSeverityStr
	case Sequential:
		return SequentialStr
	default:
		return "unknown"
	}
}

// ParseStrategy converts a textual representation into a Strategy value.
//
// The function accepts a small set of case-insensitive and stylistic variants
// and maps them to the corresponding constants. This makes configuration
// more forgiving (kebab-case, CamelCase, snake_case) while still preserving
// a single canonical output form via String().
//
// Examples of accepted inputs:
//
//	"max-severity", "MaxSeverity", "max_severity", "MAX_SEVERITY" -> MaxSeverity
//	"sequential",  "Sequential", "SEQUENTIAL"                      -> Sequential
//
// If the input string does not match any known Strategy value, ParseStrategy
// returns a non-nil *ParseError. In that case the returned Strategy MUST NOT
// be used; only the error is meaningful.
func ParseStrategy(str string) (Strategy, error) {
	switch str {
	case MaxSeverityStr, "MaxSeverity", "max_severity", "MAX_SEVERITY":
		return MaxSeverity, nil
	case SequentialStr, "Sequential", "SEQUENTIAL":
		return Sequential, nil
	default:
		return Sequential, &errors.ParseError{Type: "Strategy", Value: str}
	}
}

// Valid reports whether the Strategy value is one of the defined constants.
//
// This method is primarily useful when Strategy values may have been created
// via deserialization, numeric casts, or other untrusted input. Code that
// relies on Strategy being well-formed SHOULD call Valid before using it to
// drive planning logic.
func (s Strategy) Valid() bool {
	return s == MaxSeverity || s == Sequential
}

// MarshalJSON implements json.Marshaler for Strategy.
//
// A valid Strategy is serialized as its canonical string representation
// (for example, "max-severity"). If the value is not valid, MarshalJSON
// returns a *MarshalError and does not produce JSON output. This behavior
// prevents invalid Strategy values from silently leaking into JSON payloads
// and surfaces configuration or programming errors at encoding time.
func (s Strategy) MarshalJSON() ([]byte, error) {
	if !s.Valid() {
		return nil, &errors.MarshalError{Type: "Strategy", Value: int(s)}
	}
	return []byte(`"` + s.String() + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler for Strategy.
//
// The method accepts both string and numeric JSON representations.
//
//   - String: "max-severity", "sequential" and their accepted variants,
//     which are resolved via ParseStrategy.
//
//   - Number: 0 (MaxSeverity), 1 (Sequential), corresponding to the enum
//     constants in their declaration order.
//
// String input is the preferred, stable representation. Numeric input is
// accepted for compatibility with configurations that store enum-like values
// as integers. If the input cannot be parsed as either string or number, or
// if it resolves to an invalid Strategy, UnmarshalJSON returns an
// *UnmarshalError.
func (s *Strategy) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return &errors.UnmarshalError{Type: "Strategy", Data: data, Reason: "empty data"}
	}

	// Try string format first.
	if data[0] == '"' {
		var str string
		if err := json.Unmarshal(data, &str); err != nil {
			return &errors.UnmarshalError{Type: "Strategy", Data: data, Reason: err.Error()}
		}
		parsed, err := ParseStrategy(str)
		if err != nil {
			return err
		}
		*s = parsed
		return nil
	}

	// Fallback to numeric format.
	var i int
	if err := json.Unmarshal(data, &i); err != nil {
		return &errors.UnmarshalError{Type: "Strategy", Data: data, Reason: err.Error()}
	}
	*s = Strategy(i)
	if !s.Valid() {
		return &errors.UnmarshalError{Type: "Strategy", Data: data, Reason: "invalid numeric value"}
	}
	return nil
}

// MarshalText implements encoding.TextMarshaler for Strategy.
//
// The textual form is the same lowercase kebab-case string returned by
// String() (for example, "max-severity"). This encoding is commonly used by
// YAML and other text-based configuration formats. If the Strategy value is
// invalid, MarshalText returns a *MarshalError.
func (s Strategy) MarshalText() ([]byte, error) {
	if !s.Valid() {
		return nil, &errors.MarshalError{Type: "Strategy", Value: int(s)}
	}
	return []byte(s.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler for Strategy.
//
// The method accepts the same textual vocabulary as ParseStrategy, using it
// as the single source of truth for mapping strings to Strategy values. The
// input is treated in the same permissive way (several common naming
// variants). On failure, UnmarshalText returns the underlying *ParseError.
func (s *Strategy) UnmarshalText(text []byte) error {
	parsed, err := ParseStrategy(string(text))
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

// TypeName returns "Strategy", the name of the type for logging and debugging.
//
// This method implements part of the model.Model interface, allowing Strategy
// values to be used consistently with other model types in error messages,
// logs, and reflection-based code.
func (s Strategy) TypeName() string {
	return "Strategy"
}

// Redacted returns the same string representation as String().
//
// Strategy values contain no sensitive information (they are simply enum
// constants), so the redacted form is identical to the regular string form.
// This method implements part of the model.Model interface.
func (s Strategy) Redacted() string {
	return s.String()
}

// IsZero reports whether the Strategy has its zero value.
//
// For Strategy (an enum type), the zero value is MaxSeverity (constant 0).
// This method implements part of the model.Model interface and is useful
// when checking if a Strategy field was explicitly set or left at its
// default value.
//
// Note: The zero value (MaxSeverity) is a valid Strategy, so IsZero returning
// true does not indicate an error condition.
func (s Strategy) IsZero() bool {
	return s == MaxSeverity
}

// Equal reports whether this Strategy is equal to another value.
//
// The method accepts any type for other and uses type assertion to check if
// it is a Strategy or *Strategy. Two Strategy values are equal if they
// represent the same enum constant.
//
// This method implements part of the model.Model interface and is useful
// for comparisons in tests and validation logic.
func (s Strategy) Equal(other any) bool {
	switch v := other.(type) {
	case Strategy:
		return s == v
	case *Strategy:
		if v == nil {
			return false
		}
		return s == *v
	default:
		return false
	}
}

// Validate checks whether the Strategy value is one of the defined constants.
//
// This method returns nil if the Strategy is valid (MaxSeverity or Sequential),
// and returns an error if the value is outside the valid range.
//
// This method implements part of the model.Model interface and is typically
// called after deserialization or numeric casts to ensure the value is
// well-formed before using it in planning logic.
func (s Strategy) Validate() error {
	if !s.Valid() {
		return &errors.MarshalError{
			Type:  "Strategy",
			Value: int(s),
		}
	}
	return nil
}

// MarshalYAML implements yaml.Marshaler for Strategy.
//
// A valid Strategy is serialized as its canonical string representation
// (for example, "max-severity"). If the value is not valid, MarshalYAML
// returns a *MarshalError. This method uses MarshalText internally.
func (s Strategy) MarshalYAML() (any, error) {
	if !s.Valid() {
		return nil, &errors.MarshalError{Type: "Strategy", Value: int(s)}
	}
	return s.String(), nil
}

// UnmarshalYAML implements yaml.Unmarshaler for Strategy.
//
// The method accepts string representations of Strategy values
// (for example, "max-severity", "sequential") and resolves them via
// ParseStrategy. On failure, it returns the underlying *ParseError.
func (s *Strategy) UnmarshalYAML(node *yaml.Node) error {
	var str string
	if err := node.Decode(&str); err != nil {
		return &errors.UnmarshalError{Type: "Strategy", Data: []byte(node.Value), Reason: err.Error()}
	}
	parsed, err := ParseStrategy(str)
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}
