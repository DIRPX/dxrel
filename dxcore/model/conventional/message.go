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
	// messageHeaderPattern is the regular expression pattern used to parse
	// Conventional Commit message headers according to the Conventional Commits
	// specification version 1.0.0.
	//
	// The pattern enforces the canonical header format:
	//   <type>[(<scope>)][!]: <subject>
	//
	// Where:
	//   - type MUST be lowercase letters only (enforced by [a-z]+)
	//   - scope is optional, enclosed in parentheses, can contain any characters except ")"
	//   - breaking change marker "!" is optional and MUST appear after scope (if present)
	//   - colon and space separate header from subject
	//   - subject MUST be non-empty and can contain any characters
	//
	// Capture groups (1-indexed):
	//   1. type     - commit type (feat, fix, docs, etc.) - lowercase only
	//   2. scope    - optional scope (without parentheses) - any chars except ")"
	//   3. breaking - optional "!" indicating breaking change
	//   4. subject  - commit subject/description - any non-empty string
	//
	// Examples that match this pattern:
	//   - "feat: add new feature"              -> type=feat, scope="", breaking="", subject="add new feature"
	//   - "fix(auth): resolve login issue"     -> type=fix, scope="auth", breaking="", subject="resolve login issue"
	//   - "feat!: breaking change"             -> type=feat, scope="", breaking="!", subject="breaking change"
	//   - "fix(api)!: breaking fix"            -> type=fix, scope="api", breaking="!", subject="breaking fix"
	//
	// Examples that do NOT match:
	//   - "FEAT: uppercase type"               -> uppercase type not allowed
	//   - "feat add feature"                   -> missing colon
	//   - "feat:"                              -> missing subject
	//   - "feat(scope"                         -> unclosed parenthesis
	messageHeaderPattern = `^([a-z]+)(?:\(([^)]+)\))?(!)?:\s*(.+)$`
)

var (
	// MessageHeaderRegexp is the compiled regular expression used for parsing
	// Conventional Commit message headers. This regex is pre-compiled at package
	// initialization for performance, allowing O(1) reuse across all ParseMessage
	// calls without recompilation overhead.
	//
	// The regex enforces the Conventional Commits specification format and extracts
	// the type, optional scope, optional breaking change marker, and subject from
	// the first line of a commit message.
	//
	// This variable is exported to allow external packages to validate message
	// headers without parsing the entire message structure.
	MessageHeaderRegexp = regexp.MustCompile(messageHeaderPattern)
)

// Message represents a complete Conventional Commit message following the
// Conventional Commits specification version 1.0.0. It encapsulates all
// components of a structured commit message: required type and subject,
// optional scope, breaking change indicator, body, and trailers.
//
// This type implements the complete model.Model interface, providing:
//   - Validation: Validate() ensures all components meet specification requirements
//   - Serialization: MarshalJSON/UnmarshalJSON and MarshalYAML/UnmarshalYAML
//   - Safe logging: String() and Redacted() for full and header-only output
//   - Type identification: TypeName() returns "Message"
//   - Zero-value detection: IsZero() checks if Type and Subject are present
//   - Equality comparison: Equal() performs deep equality check
//
// Message Structure and Format:
//
// A Conventional Commit message follows this multi-line structure:
//
//	<type>[(<scope>)][!]: <subject>
//	[blank line]
//	[body]
//	[blank line]
//	[Trailer-Key: trailer value]
//	[Trailer-Key: trailer value]
//
// Where:
//   - Header (line 1): REQUIRED - contains type, optional scope, optional breaking marker, subject
//   - Blank line: separates header from body (if body present)
//   - Body: OPTIONAL - longer description, can be multiple paragraphs
//   - Blank line: separates body from trailers (if trailers present)
//   - Trailers: OPTIONAL - git-style trailers (Fixes, Signed-off-by, etc.)
//
// Examples:
//
//	Simple message (header only):
//	  "feat: add user authentication"
//
//	With scope:
//	  "fix(api): resolve timeout issue"
//
//	With breaking change:
//	  "feat!: remove deprecated endpoint"
//	  "feat(api)!: change authentication flow"
//
//	With body:
//	  "feat: add caching\n\nImproves performance significantly"
//
//	With trailers:
//	  "fix: resolve bug\n\nFixes: #123\nReviewed-by: Alice"
//
//	Complete message:
//	  "feat(api)!: add user endpoint\n\nAdds new REST API.\n\nFixes: #456\nSigned-off-by: Bob"
//
// Field Semantics:
//
//   - Type: The commit type (feat, fix, docs, etc.) - REQUIRED, MUST be valid Type constant
//   - Scope: Area of codebase affected - OPTIONAL, lowercase with hyphens
//   - Subject: Short description - REQUIRED, 1-72 characters, no period at end
//   - Breaking: Breaking change indicator - derived from "!" in header
//   - Body: Long description - OPTIONAL, provides context and rationale
//   - Trailers: Key-value metadata - OPTIONAL, follows git trailer format
//
// Zero Value:
//
// The zero value of Message is NOT valid. A valid Message MUST have:
//   - A non-zero Type (though Type's zero value Feat is valid, use IsZero to detect uninitialized)
//   - A non-zero Subject (non-empty string)
//
// Use ParseMessage to create Message instances from strings, or construct
// manually and call Validate() to ensure correctness.
type Message struct {
	// Type categorizes the nature of the change (feat, fix, docs, refactor, etc.).
	// This field is REQUIRED and MUST be a valid Type constant. The Type determines
	// semantic versioning impact: feat triggers minor version bump, fix triggers
	// patch version bump, and other types typically don't affect version.
	//
	// The json/yaml tag "type" serializes this field without omitempty, ensuring
	// it always appears in serialized output.
	Type Type `json:"type" yaml:"type"`

	// Scope indicates the area of the codebase affected by this change (e.g., "api",
	// "auth", "parser"). This field is OPTIONAL. When empty, the message applies to
	// the project as a whole rather than a specific component.
	//
	// Scope MUST be lowercase, can contain hyphens, and SHOULD be concise (typically
	// one word). The scope helps readers quickly identify which subsystem changed.
	//
	// The json/yaml tag "scope,omitempty" omits this field when empty (zero value).
	Scope Scope `json:"scope,omitempty" yaml:"scope,omitempty"`

	// Subject is a brief, imperative-mood description of the change (e.g., "add user
	// authentication", "fix memory leak"). This field is REQUIRED and MUST be 1-72
	// characters long.
	//
	// Subject SHOULD use imperative mood ("add" not "added" or "adds"), start with
	// lowercase, and NOT end with a period. The subject appears in commit logs and
	// changelogs, so it MUST be clear and concise.
	//
	// The json/yaml tag "subject" serializes this field without omitempty, ensuring
	// it always appears in serialized output.
	Subject Subject `json:"subject" yaml:"subject"`

	// Breaking indicates whether this commit introduces a breaking change (backwards-
	// incompatible modification to public APIs or behavior). This field is set to true
	// if ANY of the following conditions is met:
	//
	//   1. Header contains "!" marker: "feat!:" or "feat(scope)!:"
	//   2. Footer contains "BREAKING CHANGE:" (with space, Conventional Commits format)
	//   3. Footer contains "BREAKING-CHANGE:" (with hyphen, git trailer format)
	//
	// Per Conventional Commits specification v1.0.0, all three mechanisms are equivalent
	// and result in Breaking=true. If multiple are present, Breaking is still just true.
	//
	// Format Notes:
	//   - "BREAKING CHANGE:" (space) is the canonical Conventional Commits format
	//   - "BREAKING-CHANGE:" (hyphen) is git-compatible alternative
	//   - ParseMessage detects BOTH formats and sets Breaking=true for either
	//   - BOTH formats are added to the Trailers slice for completeness
	//   - The Key field preserves the original format ("BREAKING CHANGE" or "BREAKING-CHANGE")
	//
	// When true, this commit SHOULD trigger a major version bump in semantic versioning
	// (unless version is 0.x.y, where breaking changes only bump minor version).
	// Breaking changes SHOULD be explained in the Body or in the BREAKING CHANGE/BREAKING-CHANGE value.
	//
	// Examples:
	//   - "feat!: remove endpoint" -> Breaking=true (from "!" marker), Trailers=nil
	//   - "feat: add\n\nBREAKING CHANGE: removes API" -> Breaking=true (space format), Trailers=[{Key:"BREAKING CHANGE",...}]
	//   - "feat: add\n\nBREAKING-CHANGE: removes API" -> Breaking=true (hyphen format), Trailers=[{Key:"BREAKING-CHANGE",...}]
	//   - "feat!: change\n\nBREAKING CHANGE: explanation" -> Breaking=true (both), Trailers=[{Key:"BREAKING CHANGE",...}]
	//   - "feat: add feature" -> Breaking=false (no markers), Trailers=nil
	//
	// The json/yaml tag "breaking,omitempty" omits this field when false (zero value).
	Breaking bool `json:"breaking,omitempty" yaml:"breaking,omitempty"`

	// Body provides extended description, context, rationale, or implementation details
	// for the change. This field is OPTIONAL and can contain multiple paragraphs
	// separated by blank lines.
	//
	// The Body SHOULD explain:
	//   - Why the change was necessary (motivation)
	//   - How it addresses the problem (approach)
	//   - Any side effects or implications (consequences)
	//
	// Body MUST NOT contain CRLF line endings; use LF (\n) only. Maximum size is
	// defined by BodyMaxBytes (10KB) and BodyMaxLines (200 lines).
	//
	// The json/yaml tag "body,omitempty" omits this field when empty (zero value).
	Body Body `json:"body,omitempty" yaml:"body,omitempty"`

	// Trailers are git-style key-value metadata pairs appended to the message
	// (e.g., "Fixes: #123", "Signed-off-by: Alice <alice@example.com>"). This
	// field is OPTIONAL and can contain zero or more trailers.
	//
	// Common trailer keys include:
	//   - Fixes / Closes / Resolves: Issue references
	//   - Signed-off-by: Developer certificate of origin
	//   - Co-authored-by: Multiple authors
	//   - Reviewed-by: Code review approval
	//   - Acked-by: Acknowledgment
	//   - BREAKING CHANGE: Breaking change description (alternative to "!" marker)
	//
	// Trailers MUST appear at the end of the message, separated from the body by
	// a blank line. Trailer keys MUST start with uppercase letter and can contain
	// letters, numbers, and hyphens.
	//
	// The json/yaml tag "trailers,omitempty" omits this field when empty (nil/zero-length slice).
	Trailers []Trailer `json:"trailers,omitempty" yaml:"trailers,omitempty"`
}

// Compile-time assertion that Message implements model.Model.
var _ model.Model = (*Message)(nil)

// ParseMessage parses a raw commit message string into a structured Message,
// extracting and validating all components according to the Conventional Commits
// specification version 1.0.0.
//
// This function implements a multi-stage parsing algorithm:
//
//  1. Normalization: Converts CRLF to LF, trims leading/trailing whitespace
//  2. Header parsing: Uses MessageHeaderRegexp to extract type, scope, breaking marker, subject
//  3. Content detection: Finds first non-blank line after header
//  4. Trailer detection: Scans backwards from end to locate trailer block
//  5. Body extraction: Extracts lines between header and trailers
//  6. Component validation: Validates each extracted component
//
// Expected Message Format:
//
//	<type>[(<scope>)][!]: <subject>
//	[blank line]
//	[body paragraphs]
//	[blank line]
//	[Trailer-Key: trailer value]
//	[Trailer-Key: trailer value]
//
// Where:
//   - Line 1 (header): REQUIRED - must match messageHeaderPattern regex
//   - Blank lines: separate header/body and body/trailers
//   - Body: OPTIONAL - any text between header and trailers
//   - Trailers: OPTIONAL - must match TrailerKeyRegexp pattern
//
// Parsing Rules:
//
//   - Type: REQUIRED, must be lowercase, must be valid Type constant
//   - Scope: OPTIONAL, extracted from parentheses, validated by ParseScope
//   - Breaking: Set to true if EITHER "!" marker in header OR "BREAKING CHANGE:"/"BREAKING-CHANGE:" trailer present
//   - Subject: REQUIRED, non-empty string after ":", validated by ParseSubject
//   - Body: OPTIONAL, all non-trailer content after header
//   - Trailers: OPTIONAL, detected by scanning backwards from end, must match trailer format
//
// Breaking Change Detection:
//
// Per Conventional Commits v1.0.0, a commit is breaking if ANY of these are true:
//  1. Header contains "!" after type/scope: "feat!:" or "feat(scope)!:"
//  2. Footer contains "BREAKING CHANGE:" (with space, Conventional Commits canonical format)
//  3. Footer contains "BREAKING-CHANGE:" (with hyphen, git trailer compatible format)
//
// All three mechanisms are equally valid and result in Breaking=true. The "!" marker
// is detected during header parsing (Stage 4), while footer formats are detected
// during trailer extraction (Stage 8).
//
// Important: "BREAKING CHANGE:" (with space) is NOT a valid git trailer format
// because git-interpret-trailers rejects spaces in keys. However, it's the canonical
// format in the Conventional Commits spec. ParseMessage handles this specially:
//   - Detects "BREAKING CHANGE:" via string matching (not git trailer parsing)
//   - Creates a special Trailer with Key="BREAKING CHANGE" (preserving space)
//   - Adds it to Trailers slice for completeness
//   - "BREAKING-CHANGE:" (hyphen) is parsed normally as a git trailer
//
// Trailer Detection Algorithm:
//
// ParseMessage uses a backwards-scan algorithm to distinguish body from trailers:
//  1. Find last non-blank line
//  2. Scan backwards checking if each line matches trailer format (Key: value)
//  3. Stop at first non-trailer line or blank line separating trailers from body
//  4. If ALL non-blank lines are trailers, entire content is trailers (no body)
//
// This approach correctly handles:
//   - Messages with only trailers (no body)
//   - Messages with only body (no trailers)
//   - Messages with both body and trailers
//   - Blank lines within body (not mistaken for trailer separator)
//
// Error Conditions:
//
// ParseMessage returns an error if:
//   - Input string is empty or contains only whitespace
//   - Header line does not match Conventional Commits format
//   - Type is invalid or unknown
//   - Scope format is invalid (if present)
//   - Subject is empty or invalid
//   - Body exceeds size limits (if present)
//
// Note: Invalid trailer lines are silently skipped rather than causing errors,
// allowing flexibility in trailer formatting while still extracting valid trailers.
//
// Line Ending Handling:
//
// ParseMessage normalizes all line endings to LF (\n) before parsing, accepting
// input with LF, CRLF, or mixed line endings. The resulting Message.Body will
// always use LF line endings.
//
// Example Usage:
//
//	// Simple message
//	msg, err := ParseMessage("feat: add user authentication")
//	// Result: Type=Feat, Subject="add user authentication"
//
//	// Message with scope and breaking change
//	msg, err := ParseMessage("fix(api)!: change authentication flow")
//	// Result: Type=Fix, Scope="api", Breaking=true, Subject="change authentication flow"
//
//	// Complete message
//	input := "feat(api): add endpoint\n\nAdds new user API\n\nFixes: #123"
//	msg, err := ParseMessage(input)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(msg.Type)             // Output: Feat
//	fmt.Println(msg.Scope)            // Output: Scope("api")
//	fmt.Println(msg.Subject)          // Output: Subject("add endpoint")
//	fmt.Println(msg.Body)             // Output: Body("Adds new user API")
//	fmt.Println(len(msg.Trailers))    // Output: 1
//	fmt.Println(msg.Trailers[0].Key)  // Output: "Fixes"
func ParseMessage(s string) (Message, error) {
	// Stage 1: Input validation
	if s == "" {
		return Message{}, fmt.Errorf("message cannot be empty")
	}

	// Stage 2: Normalize line endings and split into lines
	normalized := strings.ReplaceAll(s, "\r\n", "\n")
	normalized = strings.TrimSpace(normalized)
	lines := strings.Split(normalized, "\n")
	if len(lines) == 0 {
		return Message{}, fmt.Errorf("message cannot be empty")
	}

	// Stage 3: Parse and validate header
	commitType, scope, breaking, subject, err := parseMessageHeader(lines[0])
	if err != nil {
		return Message{}, err
	}

	// Initialize message with header components
	msg := Message{
		Type:     commitType,
		Scope:    scope,
		Subject:  subject,
		Breaking: breaking,
	}

	// If message is header-only, parsing is complete
	if len(lines) == 1 {
		return msg, nil
	}

	// Stage 4: Find where content (body/trailers) starts
	contentStartIdx := findContentStart(lines)
	if contentStartIdx == -1 {
		// No content, only blank lines after header
		return msg, nil
	}

	// Stage 5: Find where trailer block starts (using backwards scan)
	trailerStartIdx := findTrailerStart(lines, contentStartIdx)

	// Stage 6: Extract body (if exists)
	body, err := extractBody(lines, contentStartIdx, trailerStartIdx)
	if err != nil {
		return Message{}, fmt.Errorf("invalid body: %w", err)
	}
	msg.Body = body

	// Stage 7: Extract trailers and detect breaking changes in footer
	trailers, hasBreakingChange, err := extractTrailers(lines, trailerStartIdx)
	if err != nil {
		return Message{}, fmt.Errorf("invalid trailers: %w", err)
	}
	msg.Trailers = trailers

	// Set Breaking flag if any footer breaking change marker found
	// (this is idempotent with header "!" marker)
	if hasBreakingChange {
		msg.Breaking = true
	}

	return msg, nil
}

// String returns the complete commit message as a properly formatted string
// following the Conventional Commits specification.
//
// This method implements the fmt.Stringer interface and satisfies the
// model.Loggable contract's String() requirement. The output is a valid
// Conventional Commit message that can be re-parsed by ParseMessage without
// loss of information (round-trip compatible).
//
// Output Format:
//
//	<type>[(<scope>)][!]: <subject>
//	[blank line]
//	[body]
//	[blank line]
//	[Trailer-Key: trailer value]
//	[Trailer-Key: trailer value]
//
// Where:
//   - Header: Always present, format is type[(scope)][!]: subject
//   - Blank line: Only inserted if body or trailers follow
//   - Body: Only included if Body is non-zero
//   - Blank line: Only inserted if both body and trailers are present
//   - Trailers: Only included if Trailers slice is non-empty
//
// Component Ordering:
//
// The breaking change marker "!" appears AFTER the scope (if present) and
// BEFORE the colon, matching the Conventional Commits specification:
//   - "feat!: breaking change" (no scope)
//   - "feat(api)!: breaking change" (with scope)
//
// Empty Components:
//
// Zero-value components are omitted from the output:
//   - If Scope.IsZero(), no "(scope)" appears
//   - If Body.IsZero(), no body section appears
//   - If Trailers is empty, no trailer section appears
//
// This ensures minimal, clean output for simple messages while supporting
// full message structure when needed.
//
// Example Outputs:
//
//	Simple message:
//	  Input:  Message{Type: Feat, Subject: "add feature"}
//	  Output: "feat: add feature"
//
//	With scope and breaking:
//	  Input:  Message{Type: Fix, Scope: "api", Breaking: true, Subject: "change flow"}
//	  Output: "fix(api)!: change flow"
//
//	Complete message:
//	  Input:  Message{Type: Feat, Scope: "api", Subject: "add endpoint",
//	                  Body: "Adds new API", Trailers: [{Key: "Fixes", Value: "#123"}]}
//	  Output: "feat(api): add endpoint\n\nAdds new API\n\nFixes: #123"
func (m Message) String() string {
	var parts []string

	// Build header: type[(scope)][!]: subject
	// Breaking marker comes after scope (if present) but before colon
	header := m.Type.String()
	if !m.Scope.IsZero() {
		header += "(" + m.Scope.String() + ")"
	}
	if m.Breaking {
		header += "!"
	}
	header += ": " + m.Subject.String()
	parts = append(parts, header)

	// Add body if present, with blank line separator
	if !m.Body.IsZero() {
		parts = append(parts, "") // Blank line before body
		parts = append(parts, m.Body.String())
	}

	// Add trailers if present, with blank line separator
	if len(m.Trailers) > 0 {
		parts = append(parts, "") // Blank line before trailers
		for _, trailer := range m.Trailers {
			parts = append(parts, trailer.String())
		}
	}

	return strings.Join(parts, "\n")
}

// Redacted returns a safe, concise representation of the Message suitable for
// production logging, metrics, and error messages.
//
// This method implements the model.Loggable contract's Redacted() requirement.
// For Message, only the header components are included in the redacted output:
// type, scope (if present), breaking change marker (if present), and subject.
// The body and trailers are intentionally excluded.
//
// Rationale for Exclusion:
//
//   - Body: May contain sensitive implementation details, internal references,
//     or lengthy explanations that would bloat log files
//   - Trailers: May contain developer names, email addresses (Signed-off-by),
//     internal issue tracker URLs, or other metadata not suitable for logs
//
// The header alone provides sufficient context for most logging use cases:
//   - Identifies the type of change (feat, fix, etc.)
//   - Identifies the affected scope/component
//   - Indicates if change is breaking
//   - Provides concise description of what changed
//
// Redacted Output Format:
//
//	<type>[(<scope>)][!]: <subject>
//
// This format is identical to the first line of String() output.
//
// Example Outputs:
//
//	Input:  Message{Type: Feat, Subject: "add feature"}
//	Output: "feat: add feature"
//
//	Input:  Message{Type: Fix, Scope: "api", Breaking: true, Subject: "change flow",
//	                Body: "Long explanation...", Trailers: [...]}
//	Output: "fix(api)!: change flow"
//
// Use Cases:
//
//   - Structured logging: log.Info("processing commit", "message", msg.Redacted())
//   - Error messages: fmt.Errorf("failed to apply %s", msg.Redacted())
//   - Metrics labels: metrics.RecordCommit(msg.Type.String(), msg.Redacted())
//   - Audit logs: audit.Log("commit parsed", "header", msg.Redacted())
func (m Message) Redacted() string {
	// Build header only: type[(scope)][!]: subject
	header := m.Type.Redacted()
	if !m.Scope.IsZero() {
		header += "(" + m.Scope.Redacted() + ")"
	}
	if m.Breaking {
		header += "!"
	}
	header += ": " + m.Subject.Redacted()
	return header
}

// TypeName returns the name of this type for error messages and debugging.
//
// This method implements the model.Identifiable contract.
func (m Message) TypeName() string {
	return "Message"
}

// IsZero reports whether this Message is the zero value.
//
// This method implements the model.ZeroCheckable contract. A Message is
// considered zero if it has no Type and no Subject (the minimum required
// fields).
func (m Message) IsZero() bool {
	return m.Type.IsZero() && m.Subject.IsZero()
}

// Equal reports whether this Message is equal to another Message.
//
// Two Messages are equal if all their fields are equal: Type, Scope, Subject,
// Breaking flag, Body, and Trailers (in the same order).
func (m Message) Equal(other Message) bool {
	if !m.Type.Equal(other.Type) {
		return false
	}
	if !m.Scope.Equal(other.Scope) {
		return false
	}
	if !m.Subject.Equal(other.Subject) {
		return false
	}
	if m.Breaking != other.Breaking {
		return false
	}
	if !m.Body.Equal(other.Body) {
		return false
	}
	if len(m.Trailers) != len(other.Trailers) {
		return false
	}
	for i := range m.Trailers {
		if !m.Trailers[i].Equal(other.Trailers[i]) {
			return false
		}
	}
	return true
}

// Validate checks whether this Message satisfies all Conventional Commits
// requirements.
//
// This method implements the model.Validatable contract. Validate returns
// nil if the Message is valid, or an error describing the validation failure.
//
// Validation rules:
//   - Type MUST be non-zero and valid
//   - Subject MUST be non-zero and valid
//   - Scope MUST be valid (if present)
//   - Body MUST be valid (if present)
//   - All Trailers MUST be valid (if present)
func (m Message) Validate() error {
	// Type is required
	if m.Type.IsZero() {
		return fmt.Errorf("Message Type is required")
	}
	if err := m.Type.Validate(); err != nil {
		return fmt.Errorf("invalid Type: %w", err)
	}

	// Subject is required
	if m.Subject.IsZero() {
		return fmt.Errorf("Message Subject is required")
	}
	if err := m.Subject.Validate(); err != nil {
		return fmt.Errorf("invalid Subject: %w", err)
	}

	// Scope is optional but must be valid if present
	if !m.Scope.IsZero() {
		if err := m.Scope.Validate(); err != nil {
			return fmt.Errorf("invalid Scope: %w", err)
		}
	}

	// Body is optional but must be valid if present
	if !m.Body.IsZero() {
		if err := m.Body.Validate(); err != nil {
			return fmt.Errorf("invalid Body: %w", err)
		}
	}

	// All trailers must be valid
	for i, trailer := range m.Trailers {
		if err := trailer.Validate(); err != nil {
			return fmt.Errorf("invalid Trailer at index %d: %w", i, err)
		}
	}

	return nil
}

// MarshalJSON serializes this Message to JSON.
//
// This method implements the json.Marshaler interface and the
// model.Serializable contract.
func (m Message) MarshalJSON() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", m.TypeName(), err)
	}

	// Use type alias to avoid infinite recursion
	type message Message
	return json.Marshal(message(m))
}

// UnmarshalJSON deserializes a Message from JSON.
//
// This method implements the json.Unmarshaler interface and the
// model.Serializable contract.
func (m *Message) UnmarshalJSON(data []byte) error {
	type message Message
	var tmp message
	if err := json.Unmarshal(data, &tmp); err != nil {
		return fmt.Errorf("cannot unmarshal JSON into %s: %w", m.TypeName(), err)
	}

	*m = Message(tmp)
	return m.Validate()
}

// MarshalYAML serializes this Message to YAML.
//
// This method implements the yaml.Marshaler interface and the
// model.Serializable contract.
func (m Message) MarshalYAML() (interface{}, error) {
	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", m.TypeName(), err)
	}

	// Use type alias to avoid infinite recursion
	type message Message
	return message(m), nil
}

// UnmarshalYAML deserializes a Message from YAML.
//
// This method implements the yaml.Unmarshaler interface and the
// model.Serializable contract.
func (m *Message) UnmarshalYAML(node *yaml.Node) error {
	type message Message
	var tmp message
	if err := node.Decode(&tmp); err != nil {
		return fmt.Errorf("cannot unmarshal YAML into %s: %w", m.TypeName(), err)
	}

	*m = Message(tmp)
	return m.Validate()
}

// parseMessageHeader parses and validates the first line of a commit message,
// extracting type, scope, breaking marker, and subject components.
//
// Returns the parsed components or an error if the header is invalid.
func parseMessageHeader(headerLine string) (commitType Type, scope Scope, breaking bool, subject Subject, err error) {
	header := strings.TrimSpace(headerLine)
	matches := MessageHeaderRegexp.FindStringSubmatch(header)
	if matches == nil {
		return Type(0), Scope(""), false, Subject(""), fmt.Errorf("invalid Conventional Commit header format: %q", header)
	}

	// Extract components from regex capture groups
	typeStr := matches[1]        // Group 1: type (feat, fix, etc.)
	scopeStr := matches[2]       // Group 2: scope (optional, without parens)
	breakingMarker := matches[3] // Group 3: breaking change marker "!" (optional)
	subjectStr := matches[4]     // Group 4: subject (remainder after colon)

	// Parse and validate type
	commitType, err = ParseType(typeStr)
	if err != nil {
		return Type(0), Scope(""), false, Subject(""), fmt.Errorf("invalid type: %w", err)
	}

	// Parse and validate scope if present
	if scopeStr != "" {
		scope, err = ParseScope(scopeStr)
		if err != nil {
			return Type(0), Scope(""), false, Subject(""), fmt.Errorf("invalid scope: %w", err)
		}
	}

	// Parse and validate subject
	subject, err = ParseSubject(subjectStr)
	if err != nil {
		return Type(0), Scope(""), false, Subject(""), fmt.Errorf("invalid subject: %w", err)
	}

	// Convert breaking marker to boolean
	breaking = breakingMarker == "!"

	return commitType, scope, breaking, subject, nil
}

// findContentStart finds the index of the first non-blank line after the header.
// Returns -1 if no content exists (only blank lines after header).
func findContentStart(lines []string) int {
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "" {
			return i
		}
	}
	return -1
}

// isTrailerOrBreakingChange checks if a line looks like a trailer or BREAKING CHANGE.
// This includes:
//   - Standard git trailer format: "Key: value" where Key matches ^[A-Za-z][A-Za-z0-9-]*$
//   - BREAKING CHANGE with space: "BREAKING CHANGE: ..."
//   - BREAKING-CHANGE with hyphen: "BREAKING-CHANGE: ..."
func isTrailerOrBreakingChange(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}

	// Check for BREAKING CHANGE with space (Conventional Commits canonical format)
	if strings.HasPrefix(line, "BREAKING CHANGE:") || strings.HasPrefix(line, "BREAKING CHANGE ") {
		return true
	}

	// Check for standard git trailer format
	colonIdx := strings.Index(line, ":")
	if colonIdx == -1 {
		return false
	}
	key := strings.TrimSpace(line[:colonIdx])
	return TrailerKeyRegexp.MatchString(key)
}

// findTrailerStart uses backwards scanning to find where the trailer block starts.
// Returns -1 if no trailers are found.
//
// Algorithm:
//  1. Find last non-blank line
//  2. Scan backwards checking if each line is a trailer
//  3. Stop at first non-trailer line or blank line separating trailers from body
//  4. If ALL non-blank lines are trailers, entire content is trailers (no body)
func findTrailerStart(lines []string, contentStartIdx int) int {
	if contentStartIdx == -1 {
		return -1
	}

	// Find last non-blank line
	lastNonBlankIdx := -1
	for i := len(lines) - 1; i >= contentStartIdx; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			lastNonBlankIdx = i
			break
		}
	}

	if lastNonBlankIdx == -1 {
		return -1
	}

	// Scan backwards to find start of trailer block
	trailerStartIdx := -1
	inTrailers := true

	for i := lastNonBlankIdx; i >= contentStartIdx; i-- {
		line := strings.TrimSpace(lines[i])

		if line == "" {
			// Blank line encountered
			if inTrailers && trailerStartIdx == -1 {
				// This blank line separates trailers from body
				trailerStartIdx = i + 1
				break
			}
			continue
		}

		if !isTrailerOrBreakingChange(lines[i]) {
			// Not a trailer line
			inTrailers = false
		}
	}

	// Special case: all non-blank lines are trailers
	if inTrailers && trailerStartIdx == -1 {
		trailerStartIdx = contentStartIdx
	}

	return trailerStartIdx
}

// extractBody extracts body text from lines between contentStart and trailerStart.
// Returns empty Body if no body content exists.
func extractBody(lines []string, contentStartIdx, trailerStartIdx int) (Body, error) {
	if contentStartIdx == -1 {
		return Body(""), nil
	}

	var bodyEndIdx int

	if trailerStartIdx != -1 && trailerStartIdx > contentStartIdx {
		// Trailers exist: body ends before trailers
		// Trim blank lines between last body line and trailer separator
		bodyEndIdx = trailerStartIdx
		for i := trailerStartIdx - 1; i >= contentStartIdx; i-- {
			if strings.TrimSpace(lines[i]) != "" {
				bodyEndIdx = i + 1
				break
			}
		}
	} else if trailerStartIdx == -1 {
		// No trailers: everything is body
		bodyEndIdx = len(lines)
	} else {
		// All content is trailers
		return Body(""), nil
	}

	if contentStartIdx >= bodyEndIdx {
		return Body(""), nil
	}

	bodyLines := lines[contentStartIdx:bodyEndIdx]
	bodyText := strings.Join(bodyLines, "\n")
	bodyText = strings.TrimSpace(bodyText)

	if bodyText == "" {
		return Body(""), nil
	}

	return ParseBody(bodyText)
}

// extractTrailers extracts and parses all trailer lines from the trailer block.
// Also detects BREAKING CHANGE/BREAKING-CHANGE and sets the breaking flag.
//
// Returns:
//   - trailers: slice of parsed Trailer objects (includes BREAKING CHANGE as special Trailer)
//   - hasBreakingChange: true if any BREAKING CHANGE/BREAKING-CHANGE trailer found
func extractTrailers(lines []string, trailerStartIdx int) ([]Trailer, bool, error) {
	if trailerStartIdx == -1 {
		return nil, false, nil
	}

	var trailers []Trailer
	hasBreakingChange := false

	for i := trailerStartIdx; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Special handling for BREAKING CHANGE with space
		// This is not a valid git trailer format, but we still want to capture it
		if strings.HasPrefix(line, "BREAKING CHANGE:") {
			hasBreakingChange = true
			// Extract the value after "BREAKING CHANGE:"
			value := strings.TrimSpace(strings.TrimPrefix(line, "BREAKING CHANGE:"))
			// Create a special Trailer with the space preserved in display
			trailers = append(trailers, Trailer{
				Key:   "BREAKING CHANGE",
				Value: value,
			})
			continue
		}

		// Try to parse as standard git trailer
		trailer, err := ParseTrailer(line)
		if err != nil {
			// Skip malformed trailer lines
			continue
		}

		trailers = append(trailers, trailer)

		// Check for BREAKING-CHANGE with hyphen
		if trailer.Key == "BREAKING-CHANGE" {
			hasBreakingChange = true
		}
	}

	return trailers, hasBreakingChange, nil
}
