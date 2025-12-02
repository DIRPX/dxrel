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

package conventional

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"dirpx.dev/dxrel/dxcore/model"
	"gopkg.in/yaml.v3"
)

const (
	// scopeFmt defines the canonical regular expression pattern used to validate
	// Conventional Commit scope identifiers in dxrel. This pattern enforces a
	// strict format that ensures scopes are machine-friendly identifiers suitable
	// for automated tooling, changelog generation, and semantic versioning systems.
	//
	// The pattern requires that scopes start and end with lowercase ASCII letters
	// or digits, creating clear boundaries for the identifier. The middle portion
	// MAY contain lowercase letters, digits, dots (.), underscores (_), forward
	// slashes (/), and hyphens (-), enabling hierarchical naming schemes such as
	// "core/io" or namespaced identifiers like "db.v2". Trailing punctuation is
	// explicitly forbidden to prevent ambiguity when scopes appear in sentences
	// or lists.
	//
	// Valid scope examples that match this pattern include "core", "api", "auth",
	// "http-router", "core/io", "db.v2", and "pkg_utils". Invalid scopes that
	// are rejected include empty strings, strings with whitespace, uppercase
	// letters ("Core"), special characters ("core*"), leading punctuation ("-core"),
	// and trailing punctuation ("core/").
	//
	// This regular expression expects that the input has already been normalized
	// through trimming of leading and trailing whitespace and conversion to
	// lowercase. Parsing functions MUST apply normalization before validation
	// against this pattern to ensure consistent behavior regardless of input
	// formatting variations.
	scopeFmt = `^[a-z0-9]([a-z0-9._/-]*[a-z0-9])?$`
)

const (
	// ScopeMinLen defines the minimum length in Unicode code points for a
	// non-empty scope value. Scope values shorter than this length (excluding
	// the zero value empty string) are considered invalid and MUST be rejected
	// during parsing and validation.
	//
	// The minimum length of 1 ensures that scopes are meaningful identifiers
	// rather than accidental empty submissions. The zero value (empty string)
	// is treated specially and represents the absence of a scope, which is
	// valid in Conventional Commits when no specific subsystem or component
	// is being targeted.
	ScopeMinLen = 1

	// ScopeMaxLen defines the maximum allowed length in Unicode code points for
	// a scope value. Scope values longer than this limit MUST be rejected during
	// parsing and validation to maintain readability and prevent abuse.
	//
	// This limit is not specified by the Conventional Commits specification but
	// is a dxrel design decision aimed at keeping scopes concise and readable.
	// Scopes are intended to be compact identifiers for subsystems, modules,
	// packages, or logical areas of a codebase, not full sentences or paragraphs.
	// A limit of 32 code points provides sufficient expressiveness for hierarchical
	// namespaces (such as "platform/services/auth") while preventing excessively
	// long identifiers that harm readability in commit headers, changelogs, and
	// release notes.
	//
	// Callers SHOULD design scope naming conventions that fit comfortably within
	// this limit. If longer identifiers are needed, consider using hierarchical
	// scopes with forward slashes or abbreviating component names.
	ScopeMaxLen = 32
)

var (
	// ScopeRegexp is the compiled regular expression used to validate Conventional
	// Commit scope identifiers against the canonical format defined by scopeFmt.
	// This compiled regexp is safe for concurrent use by multiple goroutines and
	// SHOULD be treated as a read-only, process-wide singleton.
	//
	// The regexp enforces that scopes are lowercase alphanumeric identifiers
	// optionally containing dots, underscores, slashes, and hyphens, with strict
	// requirements on start and end characters. Callers SHOULD prefer using
	// higher-level functions such as ParseScope or Scope.Validate rather than
	// matching against this regexp directly, as those functions handle input
	// normalization, length validation, and provide better error messages.
	//
	// This variable is initialized at package load time and remains constant
	// throughout the program's execution. Direct use of this regexp is appropriate
	// in test code for asserting format compliance or in low-level parsing logic
	// where the caller has already performed normalization.
	ScopeRegexp = regexp.MustCompile(scopeFmt)
)

// Scope represents the optional component or subsystem identifier in a
// Conventional Commit message, qualifying where a change applies within the
// codebase. Scopes enable automated tooling to categorize commits, generate
// component-specific changelogs, and track changes to individual subsystems
// for semantic versioning and release planning purposes.
//
// In commit message syntax, scopes appear in parentheses between the type
// and the colon separator, as in "feat(api): add user endpoint" or
// "fix(core/io): handle EOF correctly". The scope identifies the specific
// area of the codebase affected by the commit, such as a module name, package
// path, architectural layer, or functional component.
//
// This type implements the model.Model interface, providing validation,
// serialization to JSON and YAML, safe logging, type identification, and
// zero-value detection. The zero value of Scope (empty string "") is valid
// and represents the absence of a scope, indicating that the commit affects
// the codebase broadly or does not target a specific component.
//
// Scope values are stored and transmitted in normalized lowercase form with
// no surrounding whitespace. Non-empty scopes MUST match the pattern defined
// by ScopeRegexp, MUST be between ScopeMinLen and ScopeMaxLen code points in
// length, and MUST NOT contain whitespace characters. These constraints ensure
// that scopes remain machine-friendly identifiers suitable for use in URLs,
// file names, and automated filtering logic.
//
// Scopes support hierarchical naming using forward slashes (such as "core/io"
// or "platform/services/auth"), enabling logical grouping of related components.
// Other separators including dots ("db.v2"), underscores ("pkg_utils"), and
// hyphens ("http-router") are also permitted for compatibility with various
// naming conventions.
//
// When rendering commit messages, implementations SHOULD omit the parentheses
// entirely when Scope.IsZero() returns true, producing "feat: add feature"
// rather than "feat(): add feature". Non-zero scopes are always rendered with
// surrounding parentheses.
//
// Example usage:
//
//	scope := conventional.Scope("api")
//	fmt.Println(scope.String()) // Output: "api"
//
//	var parsed conventional.Scope
//	if err := json.Unmarshal([]byte(`"core/io"`), &parsed); err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(parsed.Validate()) // Output: <nil> (valid)
type Scope string

// String returns the string representation of the Scope, which is simply the
// scope identifier itself without any additional formatting or decoration.
// This method satisfies the model.Loggable interface's String requirement,
// providing a human-readable representation suitable for display and debugging.
//
// For non-empty scopes, the returned string is the normalized lowercase
// identifier. For the zero value (empty scope), the returned string is an
// empty string. When rendering commit messages, callers SHOULD check IsZero()
// and omit parentheses for empty scopes rather than rendering "()".
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently. The returned string is the Scope value
// itself, ensuring zero allocations for this operation.
//
// Example:
//
//	scope := conventional.Scope("api")
//	fmt.Println(scope.String()) // Output: "api"
//
//	empty := conventional.Scope("")
//	fmt.Println(empty.String()) // Output: ""
func (s Scope) String() string {
	return string(s)
}

// Redacted returns a safe string representation suitable for logging in
// production environments. For Scope, which contains no sensitive data,
// Redacted is identical to String and returns the scope identifier without
// modification.
//
// This method satisfies the model.Loggable interface's Redacted requirement,
// ensuring that Scope can be safely logged without risk of exposing sensitive
// information. Scope identifiers are public metadata about code structure and
// do not contain passwords, tokens, API keys, or personally identifiable
// information, making redaction unnecessary.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently.
//
// Example:
//
//	scope := conventional.Scope("auth")
//	log.Info("processing commit", "scope", scope.Redacted()) // Safe for production logs
func (s Scope) Redacted() string {
	return s.String()
}

// TypeName returns the canonical name of this model type for debugging,
// logging, and reflection purposes. This method satisfies the model.Identifiable
// interface's TypeName requirement.
//
// The returned value is always the constant string "Scope", uniquely identifying
// this type within the dxrel domain. The name follows CamelCase convention and
// omits the package prefix as required by the Model contract.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently. The returned string is a constant literal,
// ensuring zero allocations.
func (s Scope) TypeName() string {
	return "Scope"
}

// IsZero reports whether this Scope instance is in a zero or empty state,
// meaning no scope has been specified. For Scope, the zero value (empty string)
// is valid and represents the absence of a scope qualifier in a commit message.
//
// This method satisfies the model.ZeroCheckable interface's IsZero requirement.
// Unlike many model types where zero values indicate uninitialized or invalid
// states, Scope treats the empty string as a legitimate value meaning "no scope",
// which is explicitly permitted by the Conventional Commits specification.
//
// Callers can use IsZero to determine whether to render scope parentheses in
// commit messages. When IsZero returns true, implementations SHOULD omit the
// parentheses, rendering "feat: description" rather than "feat(): description".
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently.
//
// Example:
//
//	scope := conventional.Scope("")
//	fmt.Println(scope.IsZero()) // Output: true
//
//	scope = conventional.Scope("api")
//	fmt.Println(scope.IsZero()) // Output: false
func (s Scope) IsZero() bool {
	return s == ""
}

// Validate checks that the Scope value conforms to all constraints defined by
// the Conventional Commits specification and dxrel conventions. This method
// satisfies the model.Validatable interface's Validate requirement, enforcing
// data integrity.
//
// Validate returns nil if the Scope is either the zero value (empty string,
// representing no scope) or a non-empty string that satisfies all of the
// following requirements: the length in Unicode code points MUST be between
// ScopeMinLen and ScopeMaxLen inclusive; the value MUST match the regular
// expression pattern defined by ScopeRegexp after normalization; the value
// MUST NOT contain any whitespace characters.
//
// Validate returns an error if any constraint is violated. The error message
// describes which specific constraint failed and includes the invalid value
// to aid debugging. Common validation failures include scopes that are too
// long, scopes containing uppercase letters or whitespace, scopes with leading
// or trailing punctuation, and scopes containing characters outside the allowed
// set (dots, underscores, slashes, hyphens, and alphanumerics).
//
// This method MUST be fast, deterministic, and idempotent. It MUST NOT mutate
// the receiver, MUST NOT have side effects, and MUST be safe to call concurrently.
// Validation does not perform I/O or allocate memory except when constructing
// error messages for invalid values.
//
// Callers SHOULD invoke Validate after deserializing Scope from external sources
// (JSON, YAML, databases, user input) to ensure data integrity. The ToJSON,
// ToYAML, FromJSON, and FromYAML helper functions automatically call Validate
// to enforce this contract.
//
// Example:
//
//	scope := conventional.Scope("api")
//	if err := scope.Validate(); err != nil {
//	    log.Error("invalid scope", "error", err)
//	}
func (s Scope) Validate() error {
	// Empty scope is valid (represents no scope)
	if s.IsZero() {
		return nil
	}

	str := string(s)

	// Check length constraints
	if len(str) < ScopeMinLen {
		return fmt.Errorf("Scope %q is too short (minimum length: %d)", str, ScopeMinLen)
	}
	if len(str) > ScopeMaxLen {
		return fmt.Errorf("Scope %q is too long (maximum length: %d)", str, ScopeMaxLen)
	}

	// Check for whitespace (not allowed)
	if strings.ContainsAny(str, " \t\n\r") {
		return fmt.Errorf("Scope %q contains whitespace (not allowed)", str)
	}

	// Check format against regexp
	if !ScopeRegexp.MatchString(str) {
		return fmt.Errorf("Scope %q does not match required format (must be lowercase alphanumeric with optional dots, underscores, slashes, hyphens)", str)
	}

	return nil
}

// MarshalJSON implements json.Marshaler, serializing the Scope to its string
// representation as a JSON string. This method satisfies part of the
// model.Serializable interface requirement.
//
// MarshalJSON first validates that the Scope conforms to all constraints by
// calling Validate. If validation fails, marshaling fails with the validation
// error, preventing invalid data from being serialized. If validation succeeds,
// the Scope is converted to its string representation and marshaled as a JSON
// string.
//
// Empty scopes (zero values) marshal to the JSON string "" (empty string),
// maintaining the distinction between "no scope" and absent fields in JSON
// objects. Non-empty scopes marshal to their normalized lowercase form.
//
// This method MUST NOT mutate the receiver except as required by the json.Marshaler
// interface contract. It MUST be safe to call concurrently on immutable receivers.
//
// Example:
//
//	scope := conventional.Scope("api")
//	data, _ := json.Marshal(scope)
//	fmt.Println(string(data)) // Output: "api"
func (s Scope) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", s.TypeName(), err)
	}
	return json.Marshal(string(s))
}

// UnmarshalJSON implements json.Unmarshaler, deserializing a JSON string into
// a Scope value. This method satisfies part of the model.Serializable interface
// requirement.
//
// UnmarshalJSON accepts JSON strings containing scope identifiers and applies
// normalization before validation. The input undergoes trimming of leading and
// trailing whitespace using strings.TrimSpace, followed by conversion to
// lowercase using strings.ToLower. This normalization ensures that inputs like
// "  API  ", "Api", and "api" all unmarshal to the same Scope value.
//
// After unmarshaling and normalization, Validate is called to ensure the
// resulting Scope conforms to all constraints. If the normalized string is
// invalid (for example, too long, contains disallowed characters, or fails
// the regexp match), unmarshaling fails with an error describing the validation
// failure. This fail-fast behavior prevents invalid data from entering the
// system through external inputs.
//
// Empty JSON strings unmarshal successfully to the zero value Scope, representing
// no scope. JSON null values are rejected as invalid input.
//
// The method mutates the receiver to store the unmarshaled value. It is not
// safe for concurrent use during unmarshaling, though the resulting Scope value
// is safe for concurrent reads after unmarshaling completes.
//
// Example:
//
//	var scope conventional.Scope
//	json.Unmarshal([]byte(`"core/io"`), &scope)
//	fmt.Println(scope.String()) // Output: "core/io"
func (s *Scope) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return fmt.Errorf("cannot unmarshal JSON: %w", err)
	}

	parsed, err := ParseScope(str)
	if err != nil {
		return fmt.Errorf("unmarshaled model is invalid: %w", err)
	}

	*s = parsed
	return s.Validate()
}

// MarshalYAML implements yaml.Marshaler, serializing the Scope to its string
// representation for YAML encoding. This method satisfies part of the
// model.Serializable interface requirement.
//
// MarshalYAML first validates that the Scope conforms to all constraints by
// calling Validate. If validation fails, marshaling fails with the validation
// error, preventing invalid data from being serialized. If validation succeeds,
// the Scope is converted to its string representation.
//
// Empty scopes (zero values) marshal to the YAML scalar "" (empty string).
// Non-empty scopes marshal to their normalized lowercase form as YAML scalars.
//
// This method MUST NOT mutate the receiver except as required by the yaml.Marshaler
// interface contract. It MUST be safe to call concurrently on immutable receivers.
//
// Example:
//
//	scope := conventional.Scope("auth")
//	data, _ := yaml.Marshal(scope)
//	fmt.Println(string(data)) // Output: "auth\n"
func (s Scope) MarshalYAML() (interface{}, error) {
	if err := s.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", s.TypeName(), err)
	}
	return string(s), nil
}

// UnmarshalYAML implements yaml.Unmarshaler, deserializing a YAML scalar into
// a Scope value. This method satisfies part of the model.Serializable interface
// requirement.
//
// UnmarshalYAML accepts YAML scalars containing scope identifiers and applies
// normalization before validation. The input undergoes trimming of leading and
// trailing whitespace using strings.TrimSpace, followed by conversion to
// lowercase using strings.ToLower. This normalization ensures consistent
// behavior regardless of how the scope is formatted in YAML files.
//
// After unmarshaling and normalization, Validate is called to ensure the
// resulting Scope conforms to all constraints. If the normalized string is
// invalid, unmarshaling fails with an error describing the validation failure.
// This fail-fast behavior prevents invalid configuration data from corrupting
// system state.
//
// Empty YAML scalars unmarshal successfully to the zero value Scope. YAML null
// values are rejected as invalid input.
//
// The method mutates the receiver to store the unmarshaled value. It is not
// safe for concurrent use during unmarshaling, though the resulting Scope value
// is safe for concurrent reads after unmarshaling completes.
//
// Example:
//
//	var scope conventional.Scope
//	yaml.Unmarshal([]byte("db.v2"), &scope)
//	fmt.Println(scope.String()) // Output: "db.v2"
func (s *Scope) UnmarshalYAML(node *yaml.Node) error {
	var str string
	if err := node.Decode(&str); err != nil {
		return fmt.Errorf("cannot unmarshal YAML: %w", err)
	}

	parsed, err := ParseScope(str)
	if err != nil {
		return fmt.Errorf("unmarshaled model is invalid: %w", err)
	}

	*s = parsed
	return s.Validate()
}

// ParseScope parses a string into a Scope value, normalizing and validating
// the input before returning. This function provides a unified parsing entry
// point for converting external string representations into Scope values with
// comprehensive input validation and normalization.
//
// ParseScope applies a two-stage normalization process to the input. First,
// leading and trailing whitespace is removed using strings.TrimSpace. Second,
// the trimmed result is converted to lowercase using strings.ToLower. This
// normalization ensures that inputs like "  API  ", "Api", and "api" all parse
// to the same Scope value, and that scopes are stored consistently in lowercase
// form regardless of how they were originally provided.
//
// After normalization, ParseScope validates the result against all Scope
// constraints. The normalized string MUST be between ScopeMinLen and ScopeMaxLen
// code points in length (or empty, which is valid), MUST match the pattern
// defined by ScopeRegexp, and MUST NOT contain any whitespace characters. If
// any constraint is violated, ParseScope returns an error describing the
// specific validation failure.
//
// ParseScope returns an error in the following cases: if the normalized result
// is longer than ScopeMaxLen, if the normalized result contains characters not
// allowed by ScopeRegexp, or if the normalized result contains whitespace
// characters. The error message includes the original invalid input (before
// normalization) to aid debugging and provide clear feedback to users about
// what they provided.
//
// The empty string is a valid input and parses successfully to the zero value
// Scope, representing the absence of a scope. Strings containing only whitespace
// also parse to the zero value Scope after normalization removes the whitespace.
//
// Callers MUST check the returned error before using the Scope value. This
// function is pure and has no side effects. It is safe to call concurrently
// from multiple goroutines.
//
// Example:
//
//	scope, err := conventional.ParseScope("  Core/IO  ")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(scope.String()) // Output: "core/io"
func ParseScope(s string) (Scope, error) {
	// Normalize input: trim whitespace and convert to lowercase
	normalized := strings.ToLower(strings.TrimSpace(s))

	// Create scope and validate
	scope := Scope(normalized)
	if err := scope.Validate(); err != nil {
		return "", fmt.Errorf("invalid scope %q: %w", s, err)
	}

	return scope, nil
}

// Compile-time verification that Scope implements model.Model interface.
var _ model.Model = (*Scope)(nil)
