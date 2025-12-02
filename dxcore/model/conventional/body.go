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

	"dirpx.dev/dxrel/dxcore/model"
	"gopkg.in/yaml.v3"
)

const (
	// BodyMaxBytes defines the maximum allowed size in bytes for a Conventional
	// Commit body when encoded as UTF-8. Bodies exceeding this limit MUST be
	// rejected during parsing and validation to maintain predictable memory
	// usage and prevent performance degradation in tools that process commit
	// messages.
	//
	// This limit is not mandated by the Conventional Commits specification but
	// is a dxrel design decision aimed at keeping commit messages reasonably
	// sized and suitable for display in various contexts including terminal
	// output, web interfaces, changelogs, and API responses. An 8 KiB limit
	// provides sufficient space for detailed explanations, code examples,
	// references, and context while preventing abuse cases where excessively
	// large bodies degrade user experience.
	//
	// Bodies are intended to provide additional context and explanation beyond
	// the short description line, not to serve as arbitrary data containers.
	// Very large bodies harm readability in generated changelogs, inflate
	// repository sizes, and slow down parsing operations. Detailed information
	// that exceeds this limit SHOULD be placed in external documentation, issue
	// trackers, or linked resources rather than embedded directly in commit
	// messages.
	//
	// This constraint applies to the byte length of the UTF-8 encoded string,
	// not the number of runes or characters. Multi-byte UTF-8 sequences such
	// as emojis or non-ASCII characters consume more bytes than their visual
	// representation might suggest.
	BodyMaxBytes = 8 * 1024

	// BodyMaxLines defines the maximum number of logical lines allowed in a
	// Conventional Commit body. Lines are separated by LF characters (newline)
	// in the normalized representation. Bodies exceeding this line count MUST
	// be rejected during parsing and validation to maintain readability and
	// prevent unwieldy commit messages.
	//
	// This limit is not mandated by the Conventional Commits specification but
	// is a dxrel design decision aimed at keeping commit bodies concise and
	// readable. A limit of 100 lines provides ample space for detailed
	// explanations including multiple paragraphs, bullet lists, code examples,
	// and references while preventing commit messages that are difficult to
	// navigate or comprehend.
	//
	// Bodies with hundreds of lines are rarely useful in the context of commit
	// history and harm readability in generated changelogs, release notes, and
	// git log output. Limiting the line count encourages authors to write
	// focused, well-structured commit messages. If extensive documentation is
	// needed, it SHOULD be placed in separate documentation files, wikis, or
	// issue trackers rather than in commit message bodies.
	//
	// Empty lines within the body count toward this limit. Each occurrence of
	// the LF character increments the line count by one. A body without any
	// newlines is considered to have one line.
	BodyMaxLines = 100
)

// Body represents the optional multi-line body portion of a Conventional Commit
// message, providing detailed context, explanation, motivation, and additional
// information beyond what fits in the single-line description. The body appears
// after a blank line following the commit header and before any footers or
// trailers.
//
// In standard Conventional Commit syntax, the body is separated from the header
// by a blank line and may contain multiple paragraphs, bullet lists, code
// snippets, references, or any other explanatory text: "<type>[!][(<scope>)]:
// <description>\n\n<body line 1>\n<body line 2>\n...". The body provides space
// for authors to explain the what, why, and how of changes in detail, while
// the description remains concise.
//
// This type implements the model.Model interface, providing validation,
// serialization to JSON and YAML, safe logging, type identification, and
// zero-value detection. The zero value of Body (empty string "") is valid and
// represents "no body present", indicating that the commit header provides
// sufficient information without additional context.
//
// Body values are multi-line text allowing arbitrary UTF-8 characters including
// emojis, punctuation, code examples, markdown formatting, and text in any
// script or language. Bodies are human-facing documentation and support the
// full expressiveness of natural language and technical writing. Lines within
// the body are separated by LF characters (newline, '\n') in the normalized
// representation.
//
// Bodies MUST NOT contain raw CR characters ('\r') in the normalized form.
// Parsers are expected to normalize CRLF line endings to LF and either reject
// or strip lone CR characters during input processing. This normalization
// ensures consistent line-ending handling across platforms and prevents issues
// with tools that expect Unix-style line endings.
//
// Leading and trailing blank lines are removed during parsing via normalization,
// while internal blank lines are preserved to allow paragraph separation,
// visual spacing, and structured formatting. Indentation, bullet points, code
// block markers, and other formatting within the body are preserved exactly as
// authored.
//
// Size constraints are enforced to maintain reasonable commit message sizes.
// Non-empty bodies MUST NOT exceed BodyMaxBytes bytes when encoded as UTF-8
// and MUST NOT contain more than BodyMaxLines lines. These limits prevent
// memory exhaustion, performance degradation, and poor user experience in tools
// that display or process commit messages.
//
// Example usage:
//
//	body := conventional.Body("This change improves performance.\n\nBenchmarks show 2x speedup.")
//	fmt.Println(body.String()) // Multi-line output
//
//	var parsed conventional.Body
//	if err := json.Unmarshal([]byte(`"Detailed explanation\nwith multiple lines"`), &parsed); err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(parsed.Validate()) // Output: <nil> (valid)
type Body string

// ParseBody parses a string into a Body value, normalizing and validating the
// input before returning. This function provides a unified parsing entry point
// for converting external string representations into Body values with
// comprehensive input normalization and validation.
//
// ParseBody applies multi-stage normalization to the input. First, line endings
// are normalized by replacing all CRLF sequences with LF and removing any
// remaining lone CR characters, ensuring consistent Unix-style line endings.
// Second, leading and trailing blank lines are removed while preserving internal
// blank lines that separate paragraphs or provide visual structure. Third, the
// result is validated against all Body constraints.
//
// After normalization, ParseBody validates the result against all Body
// constraints. The normalized string MUST NOT exceed BodyMaxBytes bytes in
// UTF-8 encoding and MUST NOT contain more than BodyMaxLines lines. The
// normalized string MUST NOT contain any raw CR characters. If any constraint
// is violated, ParseBody returns an error describing the specific validation
// failure.
//
// ParseBody returns an error in the following cases: if the normalized result
// exceeds BodyMaxBytes, if the normalized result has too many lines, or if
// normalization somehow failed to remove all CR characters (indicating a bug).
// The error message includes relevant metrics to aid debugging and provide
// clear feedback to users.
//
// The empty string is a valid input and parses successfully to the zero value
// Body, representing "no body present". Strings containing only whitespace
// (spaces, tabs, newlines) also parse to the zero value Body after normalization
// removes blank lines.
//
// Callers MUST check the returned error before using the Body value. This
// function is pure and has no side effects. It is safe to call concurrently
// from multiple goroutines.
//
// Example:
//
//	body, err := conventional.ParseBody("\r\nLine 1\r\nLine 2\r\n\r\n")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(body.String()) // Output: "Line 1\nLine 2" (normalized)
func ParseBody(s string) (Body, error) {
	// Normalize line endings: CRLF -> LF, remove lone CR
	normalized := strings.ReplaceAll(s, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "")

	// Trim leading and trailing blank lines
	normalized = trimBlankLines(normalized)

	// Create body and validate
	body := Body(normalized)
	if err := body.Validate(); err != nil {
		return "", fmt.Errorf("invalid body: %w", err)
	}

	return body, nil
}

// String returns the string representation of the Body, which is the body text
// itself without any additional formatting or decoration. This method satisfies
// the model.Loggable interface's String requirement, providing a human-readable
// representation suitable for display and debugging.
//
// For non-empty bodies, the returned string is the normalized body text with
// LF line separators, trimmed of leading and trailing blank lines but preserving
// internal formatting. For the zero value (empty body), the returned string is
// an empty string. When rendering commit messages, callers SHOULD check IsZero()
// to determine whether to include the body section.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently. The returned string is the Body value
// itself, ensuring zero allocations for this operation.
//
// Example:
//
//	body := conventional.Body("Line 1\nLine 2")
//	fmt.Println(body.String()) // Output: "Line 1\nLine 2"
//
//	empty := conventional.Body("")
//	fmt.Println(empty.String()) // Output: ""
func (b Body) String() string {
	return string(b)
}

// Redacted returns a safe string representation suitable for logging in
// production environments. For Body, which contains no sensitive data by
// convention, Redacted is identical to String and returns the body text without
// modification.
//
// This method satisfies the model.Loggable interface's Redacted requirement,
// ensuring that Body can be safely logged without risk of exposing sensitive
// information. Commit bodies are public documentation about code changes and
// SHOULD NOT contain passwords, tokens, API keys, or personally identifiable
// information. Authors are responsible for ensuring that sensitive data is not
// included in commit messages.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently.
//
// Example:
//
//	body := conventional.Body("Fix authentication bug\n\nDetails here")
//	log.Info("processing commit", "body", body.Redacted()) // Safe for production logs
func (b Body) Redacted() string {
	return b.String()
}

// TypeName returns the canonical name of this model type for debugging,
// logging, and reflection purposes. This method satisfies the model.Identifiable
// interface's TypeName requirement.
//
// The returned value is always the constant string "Body", uniquely identifying
// this type within the dxrel domain. The name follows CamelCase convention and
// omits the package prefix as required by the Model contract.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently. The returned string is a constant literal,
// ensuring zero allocations.
func (b Body) TypeName() string {
	return "Body"
}

// IsZero reports whether this Body instance is in a zero or empty state,
// meaning no body content has been provided. For Body, the zero value (empty
// string) represents "no body present", indicating that the commit header
// provides sufficient information without additional context.
//
// This method satisfies the model.ZeroCheckable interface's IsZero requirement.
// Unlike Description where empty values typically indicate incomplete commits,
// empty Body values are common and acceptable. Many commits require only a
// concise description without detailed explanation, making zero-value bodies
// a normal occurrence rather than an error condition.
//
// Callers can use IsZero to determine whether to include the body section when
// rendering commit messages. When IsZero returns true, implementations SHOULD
// omit the body entirely rather than rendering unnecessary blank lines.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently.
//
// Example:
//
//	body := conventional.Body("")
//	fmt.Println(body.IsZero()) // Output: true
//
//	body = conventional.Body("Details here")
//	fmt.Println(body.IsZero()) // Output: false
func (b Body) IsZero() bool {
	return b == ""
}

// Validate checks that the Body value conforms to all constraints defined by
// dxrel conventions for commit message bodies. This method satisfies the
// model.Validatable interface's Validate requirement, enforcing data integrity.
//
// Validate returns nil if the Body is either the zero value (empty string,
// representing "no body present") or a non-empty string that satisfies all of
// the following requirements: the byte length when encoded as UTF-8 MUST NOT
// exceed BodyMaxBytes; the number of lines (separated by LF) MUST NOT exceed
// BodyMaxLines; the value MUST NOT contain raw CR characters ('\r'), as all
// line endings MUST be normalized to LF during parsing.
//
// Validate returns an error if any constraint is violated. The error message
// describes which specific constraint failed and includes relevant metrics
// about the invalid value to aid debugging. Common validation failures include
// bodies that exceed the byte limit (often due to large code examples or
// excessive verbosity), bodies with too many lines (often due to included
// output logs or stack traces that belong in external documentation), and
// bodies with un-normalized CRLF line endings.
//
// This method MUST be fast, deterministic, and idempotent. It MUST NOT mutate
// the receiver, MUST NOT have side effects, and MUST be safe to call concurrently.
// Validation does not perform I/O or allocate memory except when constructing
// error messages for invalid values and when counting lines.
//
// Callers SHOULD invoke Validate after deserializing Body from external sources
// (JSON, YAML, databases, user input) to ensure data integrity. The ToJSON,
// ToYAML, FromJSON, and FromYAML helper functions automatically call Validate
// to enforce this contract.
//
// Example:
//
//	body := conventional.Body("Detailed explanation\nwith context")
//	if err := body.Validate(); err != nil {
//	    log.Error("invalid body", "error", err)
//	}
func (b Body) Validate() error {
	// Empty body is valid (represents "not set")
	if b.IsZero() {
		return nil
	}

	str := string(b)

	// Check for raw CR characters (line endings must be normalized to LF)
	if strings.Contains(str, "\r") {
		return fmt.Errorf("Body contains raw CR characters (line endings must be normalized to LF)")
	}

	// Check byte size constraint
	byteLen := len(str)
	if byteLen > BodyMaxBytes {
		return fmt.Errorf("Body is too large: %d bytes (maximum: %d bytes)", byteLen, BodyMaxBytes)
	}

	// Count lines (split by '\n')
	lines := strings.Split(str, "\n")
	lineCount := len(lines)
	if lineCount > BodyMaxLines {
		return fmt.Errorf("Body has too many lines: %d lines (maximum: %d lines)", lineCount, BodyMaxLines)
	}

	return nil
}

// MarshalJSON implements json.Marshaler, serializing the Body to its string
// representation as a JSON string. This method satisfies part of the
// model.Serializable interface requirement.
//
// MarshalJSON first validates that the Body conforms to all constraints by
// calling Validate. If validation fails, marshaling fails with the validation
// error, preventing invalid data from being serialized. If validation succeeds,
// the Body is converted to its string representation and marshaled as a JSON
// string.
//
// Empty bodies (zero values) marshal to the JSON string "" (empty string),
// representing "no body present". Non-empty bodies marshal to their normalized
// form with LF line separators, preserving internal formatting and blank lines.
//
// This method MUST NOT mutate the receiver except as required by the json.Marshaler
// interface contract. It MUST be safe to call concurrently on immutable receivers.
//
// Example:
//
//	body := conventional.Body("Line 1\nLine 2")
//	data, _ := json.Marshal(body)
//	fmt.Println(string(data)) // Output: "Line 1\nLine 2"
func (b Body) MarshalJSON() ([]byte, error) {
	if err := b.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", b.TypeName(), err)
	}
	return json.Marshal(string(b))
}

// UnmarshalJSON implements json.Unmarshaler, deserializing a JSON string into
// a Body value. This method satisfies part of the model.Serializable interface
// requirement.
//
// UnmarshalJSON accepts JSON strings containing body text and applies
// normalization before validation. The input undergoes line ending normalization
// where CRLF sequences are converted to LF, and lone CR characters are removed.
// Leading and trailing blank lines are stripped using a custom trimming function
// that preserves internal blank lines. This normalization ensures consistent
// representation regardless of the source platform or editor used to create the
// commit message.
//
// After unmarshaling and normalization, Validate is called to ensure the
// resulting Body conforms to all constraints. If the normalized string is
// invalid (for example, exceeds byte limit, has too many lines, or still
// contains CR characters due to normalization bugs), unmarshaling fails with
// an error describing the validation failure. This fail-fast behavior prevents
// invalid data from entering the system through external inputs.
//
// Empty JSON strings unmarshal successfully to the zero value Body, representing
// "no body present". JSON null values are rejected as invalid input.
//
// The method mutates the receiver to store the unmarshaled value. It is not
// safe for concurrent use during unmarshaling, though the resulting Body value
// is safe for concurrent reads after unmarshaling completes.
//
// Example:
//
//	var body conventional.Body
//	json.Unmarshal([]byte(`"Line 1\nLine 2"`), &body)
//	fmt.Println(body.String()) // Output: "Line 1\nLine 2"
func (b *Body) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return fmt.Errorf("cannot unmarshal JSON: %w", err)
	}

	parsed, err := ParseBody(str)
	if err != nil {
		return fmt.Errorf("unmarshaled model is invalid: %w", err)
	}

	*b = parsed
	return b.Validate()
}

// MarshalYAML implements yaml.Marshaler, serializing the Body to its string
// representation for YAML encoding. This method satisfies part of the
// model.Serializable interface requirement.
//
// MarshalYAML first validates that the Body conforms to all constraints by
// calling Validate. If validation fails, marshaling fails with the validation
// error, preventing invalid data from being serialized. If validation succeeds,
// the Body is converted to its string representation.
//
// Empty bodies (zero values) marshal to the YAML scalar "" (empty string).
// Non-empty bodies marshal to their normalized form as YAML scalars, preserving
// internal formatting, blank lines, and line separators. Multi-line bodies may
// be rendered as YAML multi-line strings (literal or folded style) depending on
// the YAML encoder's heuristics.
//
// This method MUST NOT mutate the receiver except as required by the yaml.Marshaler
// interface contract. It MUST be safe to call concurrently on immutable receivers.
//
// Example:
//
//	body := conventional.Body("Line 1\nLine 2")
//	data, _ := yaml.Marshal(body)
//	fmt.Println(string(data)) // Output: YAML multi-line string
func (b Body) MarshalYAML() (interface{}, error) {
	if err := b.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", b.TypeName(), err)
	}
	return string(b), nil
}

// UnmarshalYAML implements yaml.Unmarshaler, deserializing a YAML scalar into
// a Body value. This method satisfies part of the model.Serializable interface
// requirement.
//
// UnmarshalYAML accepts YAML scalars (including multi-line strings) containing
// body text and applies normalization before validation. The input undergoes
// line ending normalization where CRLF sequences are converted to LF, and lone
// CR characters are removed. Leading and trailing blank lines are stripped
// while preserving internal blank lines and formatting.
//
// After unmarshaling and normalization, Validate is called to ensure the
// resulting Body conforms to all constraints. If the normalized string is
// invalid, unmarshaling fails with an error describing the validation failure.
// This fail-fast behavior prevents invalid configuration data from corrupting
// system state.
//
// Empty YAML scalars unmarshal successfully to the zero value Body. YAML null
// values are rejected as invalid input.
//
// The method mutates the receiver to store the unmarshaled value. It is not
// safe for concurrent use during unmarshaling, though the resulting Body value
// is safe for concurrent reads after unmarshaling completes.
//
// Example:
//
//	var body conventional.Body
//	yaml.Unmarshal([]byte("Line 1\nLine 2"), &body)
//	fmt.Println(body.String()) // Output: "Line 1\nLine 2"
func (b *Body) UnmarshalYAML(node *yaml.Node) error {
	var str string
	if err := node.Decode(&str); err != nil {
		return fmt.Errorf("cannot unmarshal YAML: %w", err)
	}

	parsed, err := ParseBody(str)
	if err != nil {
		return fmt.Errorf("unmarshaled model is invalid: %w", err)
	}

	*b = parsed
	return b.Validate()
}

// trimBlankLines removes leading and trailing blank lines from a string while
// preserving internal blank lines and all other content. This function is used
// during body normalization to clean up input without destroying intentional
// formatting.
//
// A blank line is defined as a line containing only whitespace characters
// (spaces, tabs) or an empty line. This function splits the input into lines,
// finds the first and last non-blank lines, and returns the substring containing
// only those lines plus any content between them.
//
// If the input contains only blank lines or is empty, trimBlankLines returns
// an empty string. Internal blank lines are never removed, allowing authors to
// use blank lines for paragraph separation and visual structure.
func trimBlankLines(s string) string {
	if s == "" {
		return ""
	}

	lines := strings.Split(s, "\n")

	// Find first non-blank line
	start := 0
	for start < len(lines) && isBlankLine(lines[start]) {
		start++
	}

	// If all lines are blank, return empty
	if start == len(lines) {
		return ""
	}

	// Find last non-blank line
	end := len(lines) - 1
	for end >= 0 && isBlankLine(lines[end]) {
		end--
	}

	// Join the non-blank range
	return strings.Join(lines[start:end+1], "\n")
}

// isBlankLine reports whether a line consists only of whitespace characters or
// is empty. This helper is used by trimBlankLines to identify lines that should
// be removed from the beginning and end of body text.
func isBlankLine(line string) bool {
	return strings.TrimSpace(line) == ""
}

// Compile-time verification that Body implements model.Model interface.
var _ model.Model = (*Body)(nil)
