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
	// trailerKeyPattern defines the canonical regular expression pattern used to
	// validate trailer key identifiers in dxrel. This pattern enforces git
	// interpret-trailers conventions ensuring keys are valid token identifiers
	// suitable for metadata attribution, sign-offs, issue references, and other
	// structured footer information.
	//
	// The pattern requires that keys start with an ASCII letter (uppercase or
	// lowercase) and may contain ASCII letters, digits, and hyphens in the
	// middle and end positions. This format matches standard git trailer
	// conventions used for Co-authored-by, Signed-off-by, Reviewed-by, Fixes,
	// Closes, Refs, and BREAKING CHANGE keys.
	//
	// Valid key examples that match this pattern include "Co-authored-by",
	// "Signed-off-by", "Reviewed-by", "Fixes", "Closes", "Refs", "Acked-by",
	// "Reported-by", and "BREAKING-CHANGE". Invalid keys that are rejected
	// include empty strings, keys starting with digits ("-123"), keys starting
	// with hyphens ("-key"), keys containing special characters ("key:value"),
	// and keys containing whitespace ("Co authored by").
	//
	// This regular expression expects that the input has already been normalized
	// through trimming of leading and trailing whitespace. Parsing functions
	// MUST apply normalization before validation against this pattern to ensure
	// consistent behavior regardless of input formatting variations.
	trailerKeyPattern = `^[A-Za-z][A-Za-z0-9-]*$`
)

const (
	// TrailerKeyMinLen defines the minimum length in Unicode code points for a
	// trailer key. Keys shorter than this length MUST be rejected during parsing
	// and validation to ensure meaningful metadata attribution.
	//
	// The minimum length of 1 ensures that keys are non-empty identifiers. Empty
	// keys have no semantic meaning in git trailers and would cause parsing
	// ambiguities when rendering commit messages.
	TrailerKeyMinLen = 1

	// TrailerKeyMaxLen defines the maximum allowed length in Unicode code points
	// for a trailer key. Keys longer than this limit MUST be rejected during
	// parsing and validation to maintain readability and prevent abuse.
	//
	// This limit is not specified by git interpret-trailers but is a dxrel
	// design decision aimed at keeping trailer keys concise and readable. Keys
	// are intended to be compact identifiers for metadata types, not full
	// sentences or paragraphs. A limit of 64 code points provides sufficient
	// expressiveness for hyphenated identifiers (such as "Co-authored-by" or
	// "BREAKING-CHANGE") while preventing excessively long identifiers that harm
	// readability in commit footers.
	TrailerKeyMaxLen = 64

	// TrailerValueMaxLen defines the maximum allowed length in Unicode code
	// points for a trailer value. Values longer than this limit MUST be rejected
	// during parsing and validation to maintain reasonable commit message sizes.
	//
	// This limit is not specified by git interpret-trailers but is a dxrel
	// design decision aimed at keeping trailer values concise. A limit of 256
	// code points provides sufficient space for names with email addresses
	// ("Jane Doe <jane@example.com>"), issue references with context ("fixes
	// #123 - memory leak in parser"), and URLs while preventing abuse cases
	// where excessively long values degrade readability.
	//
	// Values are intended to be compact metadata, not full paragraphs. Detailed
	// information SHOULD be placed in the commit body rather than in trailer
	// values.
	TrailerValueMaxLen = 256
)

var (
	// TrailerKeyRegexp is the compiled regular expression used to validate
	// trailer key identifiers against the canonical format defined by
	// trailerKeyPattern. This compiled regexp is safe for concurrent use by multiple
	// goroutines and SHOULD be treated as a read-only, process-wide singleton.
	//
	// The regexp enforces that keys are ASCII alphanumeric identifiers optionally
	// containing hyphens, starting with a letter. Callers SHOULD prefer using
	// higher-level functions such as ParseTrailer or Trailer.Validate rather
	// than matching against this regexp directly, as those functions handle
	// input normalization, length validation, and provide better error messages.
	TrailerKeyRegexp = regexp.MustCompile(trailerKeyPattern)
)

// Trailer represents a single structured trailer (footer) line at the end
// of a Git commit message, providing metadata such as attribution, sign-offs,
// issue references, and other structured information following git
// interpret-trailers conventions.
//
// Trailers appear after the commit body in the form "Key: Value" and are
// commonly used for Co-authored-by, Signed-off-by, Reviewed-by attribution,
// issue tracking (Fixes, Closes, Refs), and special metadata like BREAKING
// CHANGE notifications. Trailers enable automated tooling to extract
// structured information from commit messages for changelog generation,
// contributor attribution, and issue tracker integration.
//
// This type implements the model.Model interface, providing validation,
// serialization to JSON and YAML, safe logging, type identification, and
// zero-value detection. The zero value of Trailer (empty Key and Value) is
// valid at the Go type level but represents "trailer not set" at the AST
// level. Validation logic for complete commit messages SHOULD reject trailers
// with empty keys, as every trailer MUST have a meaningful identifier.
//
// Trailer keys MUST follow git interpret-trailers conventions: start with an
// ASCII letter, contain only ASCII letters, digits, and hyphens, and be
// reasonably short (1-64 code points). Keys MUST NOT contain colons, as the
// colon is the separator between key and value. While git performs
// case-insensitive key matching, dxrel preserves the original casing for
// display and round-tripping.
//
// Trailer values are arbitrary text that MAY contain names, email addresses,
// issue numbers, URLs, or free-form notes. Values MUST be single-line text
// without newline characters (neither LF nor CRLF). Multi-line continuation
// is handled by higher-level parsing logic. Leading and trailing whitespace
// is removed during parsing via strings.TrimSpace, while internal whitespace
// is preserved as-is.
//
// Example usage:
//
//	trailer := conventional.Trailer{
//	    Key:   "Co-authored-by",
//	    Value: "Jane Doe <jane@example.com>",
//	}
//	fmt.Println(trailer.String()) // Output: "Co-authored-by: Jane Doe <jane@example.com>"
//
//	var parsed conventional.Trailer
//	json.Unmarshal([]byte(`{"key":"Fixes","value":"#123"}`), &parsed)
//	fmt.Println(parsed.Validate()) // Output: <nil> (valid)
type Trailer struct {
	// Key is the trailer key identifier, for example "Co-authored-by",
	// "Signed-off-by", "Reviewed-by", "Fixes", "Closes", or "Refs".
	//
	// The key MUST be a non-empty string following git interpret-trailers
	// conventions: starting with an ASCII letter and containing only ASCII
	// letters, digits, and hyphens. The key MUST NOT include the trailing
	// colon, as the colon is a separator in the serialized representation.
	//
	// While git performs case-insensitive key matching for semantic equivalence,
	// dxrel preserves the original casing for display and round-tripping.
	// Higher-level code MAY normalize the key to a canonical case (such as
	// title-case "Co-Authored-By" or lowercase "co-authored-by") when matching
	// against known trailer names or generating output.
	Key string `json:"key" yaml:"key"`

	// Value is the trailer value, for example "Jane Doe <jane@example.com>"
	// for attribution trailers or "#123" for issue references.
	//
	// The value is stored as-is after parsing with leading and trailing
	// whitespace removed. The value MAY contain arbitrary text including names,
	// email addresses (in angle brackets), issue numbers (with # prefix), URLs,
	// or free-form notes. The value MUST be single-line text; newline characters
	// are not permitted.
	//
	// dxrel does not interpret the structure of Value beyond treating it as an
	// opaque string. Semantic parsing (such as extracting issue numbers from
	// "fixes #123" or parsing email addresses from "Name <email>") happens at
	// higher layers in the application.
	Value string `json:"value" yaml:"value"`
}

// ParseTrailer parses a string into a Trailer value, splitting on the first
// colon and validating the resulting key and value components. This function
// provides a unified parsing entry point for converting git interpret-trailers
// formatted strings into Trailer values with comprehensive normalization and
// validation.
//
// ParseTrailer expects input in the format "Key: Value" or "Key:Value" (with
// or without space after colon). The input is split on the first colon
// character, with everything before the colon becoming the Key and everything
// after becoming the Value. Leading and trailing whitespace is trimmed from
// both the key and value using strings.TrimSpace.
//
// After parsing and normalization, ParseTrailer validates the result against
// all Trailer constraints. The Key MUST be non-empty, match TrailerKeyRegexp,
// and fall within length limits. The Value MUST NOT contain newlines and MUST
// NOT exceed the maximum length. If any constraint is violated, ParseTrailer
// returns an error describing the specific validation failure.
//
// ParseTrailer returns an error if the input does not contain a colon (required
// for key-value separation), if the key is empty after trimming, if the key
// contains invalid characters, or if the value contains newlines or exceeds
// length limits. The error message includes relevant metrics to aid debugging
// and provide clear feedback to users.
//
// The empty string input returns an error because trailers MUST have a key.
// Strings containing only whitespace also return an error after normalization.
//
// Callers MUST check the returned error before using the Trailer value. This
// function is pure and has no side effects. It is safe to call concurrently
// from multiple goroutines.
//
// Example:
//
//	trailer, err := conventional.ParseTrailer("Fixes: #123")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(trailer.Key)   // Output: "Fixes"
//	fmt.Println(trailer.Value) // Output: "#123"
func ParseTrailer(s string) (Trailer, error) {
	// Trim input
	normalized := strings.TrimSpace(s)

	if normalized == "" {
		return Trailer{}, fmt.Errorf("trailer string cannot be empty")
	}

	// Find first colon
	colonIdx := strings.Index(normalized, ":")
	if colonIdx == -1 {
		return Trailer{}, fmt.Errorf("trailer string must contain colon separator: %q", s)
	}

	// Split on first colon
	key := strings.TrimSpace(normalized[:colonIdx])
	value := ""
	if colonIdx+1 < len(normalized) {
		value = strings.TrimSpace(normalized[colonIdx+1:])
	}

	// Create and validate trailer
	trailer := Trailer{
		Key:   key,
		Value: value,
	}

	if err := trailer.Validate(); err != nil {
		return Trailer{}, fmt.Errorf("invalid trailer: %w", err)
	}

	return trailer, nil
}

// String returns the string representation of the Trailer in git
// interpret-trailers format: "Key: Value". This method satisfies the
// model.Loggable interface's String requirement, providing a human-readable
// representation suitable for display, debugging, and rendering in commit
// messages.
//
// For non-zero trailers with both key and value, the returned string is the
// formatted trailer line with a colon and space separator. For zero-value
// trailers (empty key and value), the returned string is an empty string.
// For trailers with a key but no value, the format is "Key:" without trailing
// space.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently. The returned string is constructed from
// the Key and Value fields, allocating a new string for the result.
//
// Example:
//
//	trailer := conventional.Trailer{Key: "Fixes", Value: "#123"}
//	fmt.Println(trailer.String()) // Output: "Fixes: #123"
//
//	empty := conventional.Trailer{}
//	fmt.Println(empty.String()) // Output: ""
func (tr Trailer) String() string {
	if tr.IsZero() {
		return ""
	}
	if tr.Value == "" {
		return tr.Key + ":"
	}
	return tr.Key + ": " + tr.Value
}

// Redacted returns a safe string representation suitable for logging in
// production environments. For Trailer, which MAY contain sensitive data such
// as email addresses or URLs, Redacted returns only the key portion without
// the value to protect potentially sensitive information.
//
// This method satisfies the model.Loggable interface's Redacted requirement,
// ensuring that Trailer can be safely logged without risk of exposing
// sensitive information. Trailer values often contain email addresses in
// Co-authored-by lines, URLs in reference links, and other potentially
// sensitive metadata. Logging only the key allows monitoring of which trailer
// types are used without exposing the actual values.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently.
//
// Example:
//
//	trailer := conventional.Trailer{Key: "Co-authored-by", Value: "Jane Doe <jane@example.com>"}
//	log.Info("processing trailer", "trailer", trailer.Redacted()) // Output: "Co-authored-by"
func (tr Trailer) Redacted() string {
	return tr.Key
}

// TypeName returns the canonical name of this model type for debugging,
// logging, and reflection purposes. This method satisfies the model.Identifiable
// interface's TypeName requirement.
//
// The returned value is always the constant string "Trailer", uniquely
// identifying this type within the dxrel domain. The name follows CamelCase
// convention and omits the package prefix as required by the Model contract.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently. The returned string is a constant literal,
// ensuring zero allocations.
func (tr Trailer) TypeName() string {
	return "Trailer"
}

// IsZero reports whether this Trailer instance is in a zero or empty state,
// meaning no trailer content has been provided. For Trailer, the zero value
// (empty Key and empty Value) represents "trailer not set", indicating that
// no footer metadata was specified.
//
// This method satisfies the model.ZeroCheckable interface's IsZero requirement.
// A trailer is considered zero if both the key and value are empty strings.
// Trailers with a key but no value are NOT considered zero, as they represent
// a semantically meaningful trailer (for example "BREAKING-CHANGE:" without
// additional text).
//
// Callers can use IsZero to determine whether to include trailers when
// rendering commit messages. When IsZero returns true, implementations SHOULD
// omit the trailer entirely rather than rendering unnecessary blank lines.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently.
//
// Example:
//
//	trailer := conventional.Trailer{}
//	fmt.Println(trailer.IsZero()) // Output: true
//
//	trailer = conventional.Trailer{Key: "Fixes", Value: ""}
//	fmt.Println(trailer.IsZero()) // Output: false (has key)
func (tr Trailer) IsZero() bool {
	return tr.Key == "" && tr.Value == ""
}

// Equal reports whether this Trailer is equal to another Trailer value,
// providing an explicit equality comparison method that follows common Go
// idioms for struct types. While Trailer values can be compared using the ==
// operator directly, this method offers a named alternative that improves code
// readability and maintains consistency with other model types in the dxrel
// codebase.
//
// Equal performs field-by-field comparison and returns true if both Trailer
// values have identical Key and Value fields. The comparison is case-sensitive
// and exact for both fields, considering each Unicode code point. Empty
// trailers (zero values) are equal to other empty trailers.
//
// This method is particularly useful in table-driven tests, assertion libraries,
// trailer deduplication, and comparison operations where a method-based approach
// is more idiomatic than operator-based comparison. It also provides a
// consistent interface across all model types, some of which MAY require more
// complex equality semantics than simple field comparison.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently. The comparison is fast, deterministic,
// and performs no allocations beyond standard string comparisons.
//
// Example:
//
//	tr1 := conventional.Trailer{Key: "Fixes", Value: "#123"}
//	tr2 := conventional.Trailer{Key: "Closes", Value: "#456"}
//	tr3 := conventional.Trailer{Key: "Fixes", Value: "#123"}
//	fmt.Println(tr1.Equal(tr2)) // Output: false
//	fmt.Println(tr1.Equal(tr3)) // Output: true
func (tr Trailer) Equal(other Trailer) bool {
	return tr.Key == other.Key && tr.Value == other.Value
}

// Validate checks that the Trailer value conforms to all constraints defined
// by git interpret-trailers conventions and dxrel policies. This method
// satisfies the model.Validatable interface's Validate requirement, enforcing
// data integrity.
//
// Validate returns nil if the Trailer is either the zero value (empty key and
// value, representing "trailer not set" at the AST level) or a non-zero trailer
// that satisfies all of the following requirements: the Key MUST be non-empty
// after trimming; the Key length MUST be between TrailerKeyMinLen and
// TrailerKeyMaxLen inclusive; the Key MUST match TrailerKeyRegexp (ASCII
// letters, digits, hyphens, starting with letter); the Key MUST NOT contain
// colons; the Value length (if non-empty) MUST NOT exceed TrailerValueMaxLen;
// the Value MUST NOT contain newline characters (either LF or CRLF).
//
// Validate returns an error if any constraint is violated. The error message
// describes which specific constraint failed and includes relevant details
// about the invalid value to aid debugging. Common validation failures include
// keys that are too long, keys containing invalid characters, keys containing
// colons, values that are too long, and values containing newlines.
//
// This method MUST be fast, deterministic, and idempotent. It MUST NOT mutate
// the receiver, MUST NOT have side effects, and MUST be safe to call concurrently.
// Validation does not perform I/O or allocate memory except when constructing
// error messages for invalid values.
//
// Callers SHOULD invoke Validate after deserializing Trailer from external
// sources (JSON, YAML, databases, user input) to ensure data integrity.
//
// Example:
//
//	trailer := conventional.Trailer{Key: "Fixes", Value: "#123"}
//	if err := trailer.Validate(); err != nil {
//	    log.Error("invalid trailer", "error", err)
//	}
func (tr Trailer) Validate() error {
	// Empty trailer is valid (represents "not set")
	if tr.IsZero() {
		return nil
	}

	// Validate Key
	if tr.Key == "" {
		return fmt.Errorf("Trailer Key cannot be empty")
	}

	keyLen := len([]rune(tr.Key))
	if keyLen < TrailerKeyMinLen {
		return fmt.Errorf("Trailer Key %q is too short (minimum length: %d)", tr.Key, TrailerKeyMinLen)
	}
	if keyLen > TrailerKeyMaxLen {
		return fmt.Errorf("Trailer Key %q is too long (maximum length: %d)", tr.Key, TrailerKeyMaxLen)
	}

	if strings.Contains(tr.Key, ":") {
		return fmt.Errorf("Trailer Key %q contains colon (not allowed)", tr.Key)
	}

	if !TrailerKeyRegexp.MatchString(tr.Key) {
		return fmt.Errorf("Trailer Key %q does not match required format (must start with letter, contain only letters, digits, and hyphens)", tr.Key)
	}

	// Validate Value (may be empty, but if present must meet constraints)
	if strings.ContainsAny(tr.Value, "\n\r") {
		return fmt.Errorf("Trailer Value %q contains newline characters (not allowed)", tr.Value)
	}

	valueLen := len([]rune(tr.Value))
	if valueLen > TrailerValueMaxLen {
		return fmt.Errorf("Trailer Value is too long: %d runes (maximum: %d)", valueLen, TrailerValueMaxLen)
	}

	return nil
}

// MarshalJSON implements json.Marshaler, serializing the Trailer to a JSON
// object with "key" and "value" fields. This method satisfies part of the
// model.Serializable interface requirement.
//
// MarshalJSON first validates that the Trailer conforms to all constraints by
// calling Validate. If validation fails, marshaling fails with the validation
// error, preventing invalid data from being serialized. If validation succeeds,
// the Trailer is converted to a JSON object.
//
// Empty trailers (zero values) marshal to {"key":"","value":""}, representing
// "no trailer present". Non-empty trailers marshal to their field values
// preserving original casing and formatting.
//
// This method MUST NOT mutate the receiver except as required by the
// json.Marshaler interface contract. It MUST be safe to call concurrently on
// immutable receivers.
//
// Example:
//
//	trailer := conventional.Trailer{Key: "Fixes", Value: "#123"}
//	data, _ := json.Marshal(trailer)
//	fmt.Println(string(data)) // Output: {"key":"Fixes","value":"#123"}
func (tr Trailer) MarshalJSON() ([]byte, error) {
	if err := tr.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", tr.TypeName(), err)
	}
	type trailer Trailer
	return json.Marshal(trailer(tr))
}

// UnmarshalJSON implements json.Unmarshaler, deserializing a JSON object into
// a Trailer value. This method satisfies part of the model.Serializable
// interface requirement.
//
// UnmarshalJSON accepts JSON objects with "key" and "value" string fields and
// applies normalization before validation. The Key and Value fields undergo
// trimming of leading and trailing whitespace using strings.TrimSpace. This
// normalization ensures consistent representation regardless of the source
// formatting.
//
// After unmarshaling and normalization, Validate is called to ensure the
// resulting Trailer conforms to all constraints. If the normalized values are
// invalid (for example, key contains invalid characters, value contains
// newlines), unmarshaling fails with an error describing the validation failure.
// This fail-fast behavior prevents invalid data from entering the system
// through external inputs.
//
// The method mutates the receiver to store the unmarshaled value. It is not
// safe for concurrent use during unmarshaling, though the resulting Trailer
// value is safe for concurrent reads after unmarshaling completes.
//
// Example:
//
//	var trailer conventional.Trailer
//	json.Unmarshal([]byte(`{"key":"Fixes","value":"#123"}`), &trailer)
//	fmt.Println(trailer.String()) // Output: "Fixes: #123"
func (tr *Trailer) UnmarshalJSON(data []byte) error {
	type trailer Trailer
	var t trailer
	if err := json.Unmarshal(data, &t); err != nil {
		return fmt.Errorf("cannot unmarshal JSON: %w", err)
	}

	parsed := Trailer{
		Key:   strings.TrimSpace(t.Key),
		Value: strings.TrimSpace(t.Value),
	}

	if err := parsed.Validate(); err != nil {
		return fmt.Errorf("unmarshaled model is invalid: %w", err)
	}

	*tr = parsed
	return nil
}

// MarshalYAML implements yaml.Marshaler, serializing the Trailer to a YAML
// mapping with "key" and "value" fields. This method satisfies part of the
// model.Serializable interface requirement.
//
// MarshalYAML first validates that the Trailer conforms to all constraints by
// calling Validate. If validation fails, marshaling fails with the validation
// error, preventing invalid data from being serialized. If validation succeeds,
// the Trailer is converted to a YAML mapping.
//
// Empty trailers (zero values) marshal to YAML with empty strings for both
// fields. Non-empty trailers marshal to their field values preserving original
// casing and formatting.
//
// This method MUST NOT mutate the receiver except as required by the
// yaml.Marshaler interface contract. It MUST be safe to call concurrently on
// immutable receivers.
//
// Example:
//
//	trailer := conventional.Trailer{Key: "Fixes", Value: "#123"}
//	data, _ := yaml.Marshal(trailer)
//	fmt.Println(string(data)) // Output: key: Fixes\nvalue: '#123'
func (tr Trailer) MarshalYAML() (interface{}, error) {
	if err := tr.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", tr.TypeName(), err)
	}
	type trailer Trailer
	return trailer(tr), nil
}

// UnmarshalYAML implements yaml.Unmarshaler, deserializing a YAML mapping into
// a Trailer value. This method satisfies part of the model.Serializable
// interface requirement.
//
// UnmarshalYAML accepts YAML mappings with "key" and "value" string fields and
// applies normalization before validation. The Key and Value fields undergo
// trimming of leading and trailing whitespace. This normalization ensures
// consistent representation regardless of the source formatting.
//
// After unmarshaling and normalization, Validate is called to ensure the
// resulting Trailer conforms to all constraints. If the normalized values are
// invalid, unmarshaling fails with an error describing the validation failure.
// This fail-fast behavior prevents invalid configuration data from corrupting
// system state.
//
// The method mutates the receiver to store the unmarshaled value. It is not
// safe for concurrent use during unmarshaling, though the resulting Trailer
// value is safe for concurrent reads after unmarshaling completes.
//
// Example:
//
//	var trailer conventional.Trailer
//	yaml.Unmarshal([]byte("key: Fixes\nvalue: '#123'"), &trailer)
//	fmt.Println(trailer.String()) // Output: "Fixes: #123"
func (tr *Trailer) UnmarshalYAML(node *yaml.Node) error {
	type trailer Trailer
	var t trailer
	if err := node.Decode(&t); err != nil {
		return fmt.Errorf("cannot unmarshal YAML: %w", err)
	}

	parsed := Trailer{
		Key:   strings.TrimSpace(t.Key),
		Value: strings.TrimSpace(t.Value),
	}

	if err := parsed.Validate(); err != nil {
		return fmt.Errorf("unmarshaled model is invalid: %w", err)
	}

	*tr = parsed
	return nil
}

// Compile-time verification that Trailer implements model.Model interface.
var _ model.Model = (*Trailer)(nil)
