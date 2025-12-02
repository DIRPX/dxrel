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
	"strings"
	"unicode"

	"dirpx.dev/dxrel/dxcore/model"
	"gopkg.in/yaml.v3"
)

const (
	// DescriptionMinLen defines the minimum length in Unicode code points
	// (runes) for a non-empty Conventional Commit description. Descriptions
	// shorter than this length (excluding the zero value empty string) are
	// considered invalid and MUST be rejected during parsing and validation.
	//
	// The minimum length of 1 ensures that descriptions contain meaningful
	// content rather than being accidentally empty or consisting only of
	// whitespace that gets trimmed away. The zero value (empty string) is
	// treated specially at the AST level and represents "description not set",
	// which validation logic SHOULD reject when constructing complete commit
	// messages.
	//
	// This constraint applies to the number of Unicode code points (runes) in
	// the string, not the number of bytes. Multi-byte UTF-8 characters such as
	// emojis or non-ASCII letters count as single runes for length calculation
	// purposes.
	DescriptionMinLen = 1

	// DescriptionMaxLen defines the maximum allowed length in Unicode code
	// points (runes) for a Conventional Commit description. Descriptions longer
	// than this limit MUST be rejected during parsing and validation to maintain
	// readability and compatibility with tools that display commit summaries.
	//
	// This limit is not mandated by the Conventional Commits specification but
	// is a dxrel design decision inspired by widely adopted git best practices,
	// particularly the recommendation to keep commit subject lines within 72
	// characters for optimal display in terminal output, git log, and web
	// interfaces. A 72-character limit ensures that commit summaries remain
	// readable in standard 80-column terminals with room for decorations such
	// as branch names or commit hashes.
	//
	// Descriptions are intended to be short, human-readable summaries of changes,
	// not full paragraphs or essays. Detailed explanations SHOULD be placed in
	// the commit body rather than cramming them into the description line.
	// Keeping descriptions concise improves readability of logs, commit lists,
	// changelogs, and release notes.
	//
	// This constraint applies to the number of Unicode code points (runes) in
	// the string, not the number of bytes. Multi-byte UTF-8 characters such as
	// emojis or non-ASCII letters count as single runes for length calculation
	// purposes.
	DescriptionMaxLen = 72
)

// Description represents the short, single-line summary portion of a
// Conventional Commit header, concisely describing the change made in the
// commit. The description appears after the type, optional scope, and colon
// separator in commit message syntax, providing human-readable context about
// what the commit does.
//
// In standard Conventional Commit syntax, the description occupies the
// rightmost portion of the commit header: "<type>[!][(<scope>)]: <description>".
// Examples include "add user registration endpoint", "fix panic when config
// is nil", and "remove deprecated v1 API". The description SHOULD begin with
// a lowercase verb in imperative mood (add, fix, update, remove) for consistency,
// though this convention is not enforced by validation logic.
//
// This type implements the model.Model interface, providing validation,
// serialization to JSON and YAML, safe logging, type identification, and
// zero-value detection. The zero value of Description (empty string "") is
// valid at the Go type level but represents "description not set" at the AST
// level. Validation logic for complete commit messages SHOULD reject empty
// descriptions, as every commit MUST have a meaningful summary.
//
// Description values MUST be single-line text without newline characters
// (neither LF nor CRLF). Multi-line content belongs in the commit body, not
// the description. The description MUST contain at least one non-whitespace
// character to be considered valid. Leading and trailing whitespace is removed
// during parsing via strings.TrimSpace, while internal whitespace (spaces
// between words) is preserved as-is.
//
// Length constraints are enforced in Unicode code points (runes) rather than
// bytes, ensuring that multi-byte characters such as emojis, accented letters,
// and non-Latin scripts are counted fairly. Non-empty descriptions MUST be
// between DescriptionMinLen and DescriptionMaxLen runes in length. Descriptions
// exceeding DescriptionMaxLen harm readability in terminal output and git log
// displays, while descriptions shorter than DescriptionMinLen lack meaningful
// content.
//
// Unlike Scope identifiers, Description supports arbitrary UTF-8 text including
// non-ASCII characters, punctuation, emojis, and various scripts. Descriptions
// are human-facing text meant for readability and comprehension, not machine-
// friendly identifiers. Case sensitivity is preserved; descriptions are NOT
// automatically converted to lowercase during parsing.
//
// Example usage:
//
//	desc := conventional.Description("add user endpoint")
//	fmt.Println(desc.String()) // Output: "add user endpoint"
//
//	var parsed conventional.Description
//	if err := json.Unmarshal([]byte(`"fix panic when config is nil"`), &parsed); err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(parsed.Validate()) // Output: <nil> (valid)
type Description string

// ParseDescription parses a string into a Description value, normalizing and
// validating the input before returning. This function provides a unified
// parsing entry point for converting external string representations into
// Description values with comprehensive input validation and normalization.
//
// ParseDescription applies normalization to the input by removing leading and
// trailing whitespace using strings.TrimSpace. Internal whitespace between
// words is preserved as-is. Unlike Scope parsing, descriptions are NOT converted
// to lowercase; original case is preserved to maintain human-readable text
// quality and respect author intent for capitalization.
//
// After normalization, ParseDescription validates the result against all
// Description constraints. The normalized string MUST be between DescriptionMinLen
// and DescriptionMaxLen runes in length (or empty, which is valid at the type
// level), MUST NOT contain newline characters, and MUST contain at least one
// non-whitespace character. If any constraint is violated, ParseDescription
// returns an error describing the specific validation failure.
//
// ParseDescription returns an error in the following cases: if the normalized
// result is longer than DescriptionMaxLen, if the normalized result contains
// newline characters, or if the normalized result consists entirely of whitespace.
// The error message includes relevant details to aid debugging and provide clear
// feedback to users about what they provided.
//
// The empty string is a valid input and parses successfully to the zero value
// Description, representing "description not set". Strings containing only
// whitespace also parse to the zero value Description after normalization
// removes the whitespace.
//
// Callers MUST check the returned error before using the Description value.
// This function is pure and has no side effects. It is safe to call concurrently
// from multiple goroutines.
//
// Example:
//
//	desc, err := conventional.ParseDescription("  Add User Endpoint  ")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(desc.String()) // Output: "Add User Endpoint" (case preserved)
func ParseDescription(s string) (Description, error) {
	// Normalize input: trim whitespace (but do NOT convert to lowercase)
	normalized := strings.TrimSpace(s)

	// Create description and validate
	desc := Description(normalized)
	if err := desc.Validate(); err != nil {
		return "", fmt.Errorf("invalid description: %w", err)
	}

	return desc, nil
}

// String returns the string representation of the Description, which is simply
// the description text itself without any additional formatting or decoration.
// This method satisfies the model.Loggable interface's String requirement,
// providing a human-readable representation suitable for display and debugging.
//
// For non-empty descriptions, the returned string is the trimmed description
// text preserving internal whitespace and original case. For the zero value
// (empty description), the returned string is an empty string. When rendering
// commit messages, callers SHOULD check IsZero() and validate that a non-empty
// description exists before constructing the full commit header.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently. The returned string is the Description
// value itself, ensuring zero allocations for this operation.
//
// Example:
//
//	desc := conventional.Description("add user endpoint")
//	fmt.Println(desc.String()) // Output: "add user endpoint"
//
//	empty := conventional.Description("")
//	fmt.Println(empty.String()) // Output: ""
func (d Description) String() string {
	return string(d)
}

// Redacted returns a safe string representation suitable for logging in
// production environments. For Description, which contains no sensitive data,
// Redacted is identical to String and returns the description text without
// modification.
//
// This method satisfies the model.Loggable interface's Redacted requirement,
// ensuring that Description can be safely logged without risk of exposing
// sensitive information. Commit descriptions are public metadata about code
// changes and do not contain passwords, tokens, API keys, or personally
// identifiable information, making redaction unnecessary.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently.
//
// Example:
//
//	desc := conventional.Description("fix authentication bug")
//	log.Info("processing commit", "desc", desc.Redacted()) // Safe for production logs
func (d Description) Redacted() string {
	return d.String()
}

// TypeName returns the canonical name of this model type for debugging,
// logging, and reflection purposes. This method satisfies the model.Identifiable
// interface's TypeName requirement.
//
// The returned value is always the constant string "Description", uniquely
// identifying this type within the dxrel domain. The name follows CamelCase
// convention and omits the package prefix as required by the Model contract.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently. The returned string is a constant literal,
// ensuring zero allocations.
func (d Description) TypeName() string {
	return "Description"
}

// IsZero reports whether this Description instance is in a zero or empty state,
// meaning no description has been provided. For Description, the zero value
// (empty string) represents "description not set" at the AST level, though it
// is a valid Go value.
//
// This method satisfies the model.ZeroCheckable interface's IsZero requirement.
// Unlike Scope where empty values are explicitly permitted by the Conventional
// Commits specification, empty descriptions typically indicate incomplete or
// invalid commit messages. Higher-level validation logic SHOULD reject commits
// with zero-value descriptions, as every commit MUST have a meaningful summary.
//
// Callers can use IsZero to determine whether a description has been provided
// when constructing or validating commit messages. When IsZero returns true,
// the commit message is incomplete and SHOULD NOT be rendered or persisted.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently.
//
// Example:
//
//	desc := conventional.Description("")
//	fmt.Println(desc.IsZero()) // Output: true
//
//	desc = conventional.Description("add feature")
//	fmt.Println(desc.IsZero()) // Output: false
func (d Description) IsZero() bool {
	return d == ""
}

// Validate checks that the Description value conforms to all constraints
// defined by Conventional Commits best practices and dxrel conventions. This
// method satisfies the model.Validatable interface's Validate requirement,
// enforcing data integrity.
//
// Validate returns nil if the Description is either the zero value (empty
// string, representing "description not set" at the AST level) or a non-empty
// string that satisfies all of the following requirements: the length in
// Unicode code points (runes) MUST be between DescriptionMinLen and
// DescriptionMaxLen inclusive; the value MUST NOT contain newline characters
// (either LF or CRLF); the value MUST contain at least one non-whitespace
// character (checked using unicode.IsSpace).
//
// Validate returns an error if any constraint is violated. The error message
// describes which specific constraint failed and includes relevant details
// about the invalid value to aid debugging. Common validation failures include
// descriptions that are too long for terminal display, descriptions containing
// newlines (which belong in the commit body), descriptions consisting entirely
// of whitespace, and empty strings when a non-zero description is expected.
//
// This method MUST be fast, deterministic, and idempotent. It MUST NOT mutate
// the receiver, MUST NOT have side effects, and MUST be safe to call concurrently.
// Validation does not perform I/O or allocate memory except when constructing
// error messages for invalid values.
//
// Callers SHOULD invoke Validate after deserializing Description from external
// sources (JSON, YAML, databases, user input) to ensure data integrity. The
// ToJSON, ToYAML, FromJSON, and FromYAML helper functions automatically call
// Validate to enforce this contract.
//
// Example:
//
//	desc := conventional.Description("add user endpoint")
//	if err := desc.Validate(); err != nil {
//	    log.Error("invalid description", "error", err)
//	}
func (d Description) Validate() error {
	// Empty description is valid at the type level (represents "not set")
	if d.IsZero() {
		return nil
	}

	str := string(d)

	// Check for newlines (not allowed in single-line descriptions)
	if strings.ContainsAny(str, "\n\r") {
		return fmt.Errorf("Description %q contains newline characters (not allowed in single-line descriptions)", str)
	}

	// Count runes (Unicode code points) for length validation
	runeCount := len([]rune(str))

	// Check length constraints
	if runeCount < DescriptionMinLen {
		return fmt.Errorf("Description is too short: %d runes (minimum: %d)", runeCount, DescriptionMinLen)
	}
	if runeCount > DescriptionMaxLen {
		return fmt.Errorf("Description is too long: %d runes (maximum: %d)", runeCount, DescriptionMaxLen)
	}

	// Check that description contains at least one non-whitespace character
	hasNonWhitespace := false
	for _, r := range str {
		if !unicode.IsSpace(r) {
			hasNonWhitespace = true
			break
		}
	}
	if !hasNonWhitespace {
		return fmt.Errorf("Description %q contains only whitespace (must have meaningful content)", str)
	}

	return nil
}

// MarshalJSON implements json.Marshaler, serializing the Description to its
// string representation as a JSON string. This method satisfies part of the
// model.Serializable interface requirement.
//
// MarshalJSON first validates that the Description conforms to all constraints
// by calling Validate. If validation fails, marshaling fails with the validation
// error, preventing invalid data from being serialized. If validation succeeds,
// the Description is converted to its string representation and marshaled as a
// JSON string.
//
// Empty descriptions (zero values) marshal to the JSON string "" (empty string),
// representing "description not set". Non-empty descriptions marshal to their
// trimmed form preserving internal whitespace and original case.
//
// This method MUST NOT mutate the receiver except as required by the json.Marshaler
// interface contract. It MUST be safe to call concurrently on immutable receivers.
//
// Example:
//
//	desc := conventional.Description("add user endpoint")
//	data, _ := json.Marshal(desc)
//	fmt.Println(string(data)) // Output: "add user endpoint"
func (d Description) MarshalJSON() ([]byte, error) {
	if err := d.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", d.TypeName(), err)
	}
	return json.Marshal(string(d))
}

// UnmarshalJSON implements json.Unmarshaler, deserializing a JSON string into
// a Description value. This method satisfies part of the model.Serializable
// interface requirement.
//
// UnmarshalJSON accepts JSON strings containing description text and applies
// normalization before validation. The input undergoes trimming of leading and
// trailing whitespace using strings.TrimSpace. Unlike Scope, Description values
// are NOT converted to lowercase; original case is preserved to maintain
// human-readable text quality.
//
// After unmarshaling and normalization, Validate is called to ensure the
// resulting Description conforms to all constraints. If the normalized string
// is invalid (for example, too long, contains newlines, or consists only of
// whitespace), unmarshaling fails with an error describing the validation
// failure. This fail-fast behavior prevents invalid data from entering the
// system through external inputs.
//
// Empty JSON strings unmarshal successfully to the zero value Description,
// representing "description not set". JSON null values are rejected as invalid
// input.
//
// The method mutates the receiver to store the unmarshaled value. It is not
// safe for concurrent use during unmarshaling, though the resulting Description
// value is safe for concurrent reads after unmarshaling completes.
//
// Example:
//
//	var desc conventional.Description
//	json.Unmarshal([]byte(`"fix authentication bug"`), &desc)
//	fmt.Println(desc.String()) // Output: "fix authentication bug"
func (d *Description) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return fmt.Errorf("cannot unmarshal JSON: %w", err)
	}

	parsed, err := ParseDescription(str)
	if err != nil {
		return fmt.Errorf("unmarshaled model is invalid: %w", err)
	}

	*d = parsed
	return d.Validate()
}

// MarshalYAML implements yaml.Marshaler, serializing the Description to its
// string representation for YAML encoding. This method satisfies part of the
// model.Serializable interface requirement.
//
// MarshalYAML first validates that the Description conforms to all constraints
// by calling Validate. If validation fails, marshaling fails with the validation
// error, preventing invalid data from being serialized. If validation succeeds,
// the Description is converted to its string representation.
//
// Empty descriptions (zero values) marshal to the YAML scalar "" (empty string).
// Non-empty descriptions marshal to their trimmed form as YAML scalars,
// preserving internal whitespace and original case.
//
// This method MUST NOT mutate the receiver except as required by the yaml.Marshaler
// interface contract. It MUST be safe to call concurrently on immutable receivers.
//
// Example:
//
//	desc := conventional.Description("update documentation")
//	data, _ := yaml.Marshal(desc)
//	fmt.Println(string(data)) // Output: "update documentation\n"
func (d Description) MarshalYAML() (interface{}, error) {
	if err := d.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", d.TypeName(), err)
	}
	return string(d), nil
}

// UnmarshalYAML implements yaml.Unmarshaler, deserializing a YAML scalar into
// a Description value. This method satisfies part of the model.Serializable
// interface requirement.
//
// UnmarshalYAML accepts YAML scalars containing description text and applies
// normalization before validation. The input undergoes trimming of leading and
// trailing whitespace using strings.TrimSpace. Original case is preserved to
// maintain human-readable text quality.
//
// After unmarshaling and normalization, Validate is called to ensure the
// resulting Description conforms to all constraints. If the normalized string
// is invalid, unmarshaling fails with an error describing the validation failure.
// This fail-fast behavior prevents invalid configuration data from corrupting
// system state.
//
// Empty YAML scalars unmarshal successfully to the zero value Description.
// YAML null values are rejected as invalid input.
//
// The method mutates the receiver to store the unmarshaled value. It is not
// safe for concurrent use during unmarshaling, though the resulting Description
// value is safe for concurrent reads after unmarshaling completes.
//
// Example:
//
//	var desc conventional.Description
//	yaml.Unmarshal([]byte("remove deprecated API"), &desc)
//	fmt.Println(desc.String()) // Output: "remove deprecated API"
func (d *Description) UnmarshalYAML(node *yaml.Node) error {
	var str string
	if err := node.Decode(&str); err != nil {
		return fmt.Errorf("cannot unmarshal YAML: %w", err)
	}

	parsed, err := ParseDescription(str)
	if err != nil {
		return fmt.Errorf("unmarshaled model is invalid: %w", err)
	}

	*d = parsed
	return d.Validate()
}

// Compile-time verification that Description implements model.Model interface.
var _ model.Model = (*Description)(nil)
