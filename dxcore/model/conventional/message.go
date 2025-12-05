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
	// Conventional Commit message headers.
	//
	// The pattern matches the header format:
	//   <type>[(<scope>)][!]: <subject>
	//
	// Capture groups:
	//   1. type - commit type (feat, fix, etc.)
	//   2. scope - optional scope in parentheses (without the parens)
	//   3. breaking - optional "!" for breaking change
	//   4. subject - commit subject/description
	//
	// Examples that match:
	//   - "feat: add new feature"
	//   - "fix(auth): resolve login issue"
	//   - "feat!: breaking change"
	//   - "fix(api)!: breaking fix"
	messageHeaderPattern = `^([a-z]+)(?:\(([^)]+)\))?(!)?:\s*(.+)$`
)

var (
	// MessageHeaderRegexp is the compiled regular expression for parsing
	// Conventional Commit headers.
	MessageHeaderRegexp = regexp.MustCompile(messageHeaderPattern)
)

// Message represents a complete Conventional Commit message with all its
// components: type, optional scope, subject, optional body, and optional
// trailers.
//
// This type implements the model.Model interface, providing validation,
// serialization to JSON and YAML, safe logging, type identification, and
// zero-value detection. A Message encapsulates the full structure of a
// commit message following the Conventional Commits specification.
//
// The Message structure follows this format:
//   <type>[!][(<scope>)]: <subject>
//   [blank line]
//   [body]
//   [blank line]
//   [trailer: value]
//   [trailer: value]
//
// Examples:
//   - Simple: "feat: add user authentication"
//   - With scope: "fix(api): resolve timeout issue"
//   - With breaking change: "feat!: remove deprecated endpoint"
//   - With body: "feat: add caching\n\nImproves performance significantly"
//   - With trailers: "fix: bug\n\nFixes: #123\nReviewed-by: Alice"
//
// The zero value of Message is not valid; a valid message MUST have at least
// a Type and Subject.
type Message struct {
	// Type is the commit type (feat, fix, docs, etc.)
	Type Type `json:"type" yaml:"type"`

	// Scope is the optional scope indicating what area of the codebase is
	// affected. Empty string means no scope.
	Scope Scope `json:"scope,omitempty" yaml:"scope,omitempty"`

	// Subject is the short description of the change (required).
	Subject Subject `json:"subject" yaml:"subject"`

	// Breaking indicates if this is a breaking change (indicated by "!" in
	// the header).
	Breaking bool `json:"breaking,omitempty" yaml:"breaking,omitempty"`

	// Body is the optional longer description providing additional context.
	// Empty string means no body.
	Body Body `json:"body,omitempty" yaml:"body,omitempty"`

	// Trailers are optional key-value pairs at the end of the message
	// (Signed-off-by, Fixes, etc.).
	Trailers []Trailer `json:"trailers,omitempty" yaml:"trailers,omitempty"`
}

// Compile-time assertion that Message implements model.Model.
var _ model.Model = (*Message)(nil)

// ParseMessage parses a raw commit message string into a Message structure.
//
// ParseMessage applies the Conventional Commits specification to extract
// type, scope, subject, body, and trailers from the input string. The
// function handles various formatting variations including different line
// endings (LF, CRLF), extra whitespace, and missing optional components.
//
// The expected format is:
//   <type>[!][(<scope>)]: <subject>
//   [blank line]
//   [body paragraphs]
//   [blank line]
//   [Trailer-Key: trailer value]
//
// If the input does not match the Conventional Commits header format,
// ParseMessage returns an error. The subject is required; type is required;
// scope, body, and trailers are optional.
//
// Example usage:
//
//	msg, err := conventional.ParseMessage("feat(api): add endpoint\n\nAdds new user API\n\nFixes: #123")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(msg.Type)    // Output: TypeFeat
//	fmt.Println(msg.Scope)   // Output: "api"
//	fmt.Println(msg.Subject) // Output: "add endpoint"
func ParseMessage(s string) (Message, error) {
	if s == "" {
		return Message{}, fmt.Errorf("message cannot be empty")
	}

	// Normalize line endings to LF
	normalized := strings.ReplaceAll(s, "\r\n", "\n")
	normalized = strings.TrimSpace(normalized)

	// Split into lines
	lines := strings.Split(normalized, "\n")
	if len(lines) == 0 {
		return Message{}, fmt.Errorf("message cannot be empty")
	}

	// Parse header (first line)
	header := strings.TrimSpace(lines[0])
	matches := MessageHeaderRegexp.FindStringSubmatch(header)
	if matches == nil {
		return Message{}, fmt.Errorf("invalid Conventional Commit header format: %q", header)
	}

	// Extract components from regex matches
	typeStr := matches[1]
	scopeStr := matches[2]
	breakingMarker := matches[3]
	subjectStr := matches[4]

	// Parse type
	commitType, err := ParseType(typeStr)
	if err != nil {
		return Message{}, fmt.Errorf("invalid type: %w", err)
	}

	// Parse scope (optional)
	var scope Scope
	if scopeStr != "" {
		scope, err = ParseScope(scopeStr)
		if err != nil {
			return Message{}, fmt.Errorf("invalid scope: %w", err)
		}
	}

	// Parse subject (required)
	subject, err := ParseSubject(subjectStr)
	if err != nil {
		return Message{}, fmt.Errorf("invalid subject: %w", err)
	}

	// Check for breaking change marker
	breaking := breakingMarker == "!"

	// Initialize message with header components
	msg := Message{
		Type:     commitType,
		Scope:    scope,
		Subject:  subject,
		Breaking: breaking,
	}

	// If there's only one line, we're done
	if len(lines) == 1 {
		return msg, nil
	}

	// Find body and trailers
	// Skip to first non-blank line after header
	contentStartIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "" {
			contentStartIdx = i
			break
		}
	}

	if contentStartIdx == -1 {
		// No body or trailers, just blank lines
		return msg, nil
	}

	// Helper function to check if a line looks like a trailer
	isTrailerLine := func(line string) bool {
		line = strings.TrimSpace(line)
		if line == "" {
			return false
		}
		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			return false
		}
		key := strings.TrimSpace(line[:colonIdx])
		return TrailerKeyRegexp.MatchString(key)
	}

	// Find where trailers start by scanning backwards from the end
	// Trailers are consecutive trailer-formatted lines at the end (possibly separated by blank lines)
	trailerStartIdx := -1
	lastNonBlankIdx := -1

	// Find last non-blank line
	for i := len(lines) - 1; i >= contentStartIdx; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			lastNonBlankIdx = i
			break
		}
	}

	if lastNonBlankIdx != -1 {
		// Scan backwards to find start of trailer block
		// Trailers must be at the end and separated from body by blank line
		inTrailers := true
		for i := lastNonBlankIdx; i >= contentStartIdx; i-- {
			line := strings.TrimSpace(lines[i])
			if line == "" {
				// Blank line - if we're in trailers, this separates them from body
				if inTrailers && trailerStartIdx == -1 {
					// First blank line before trailers, mark trailer start
					trailerStartIdx = i + 1
					break
				}
				continue
			}
			if !isTrailerLine(lines[i]) {
				// Not a trailer line
				inTrailers = false
			}
		}

		// If all non-blank lines from contentStartIdx are trailers, they're all trailers
		if inTrailers && trailerStartIdx == -1 {
			trailerStartIdx = contentStartIdx
		}
	}

	// Extract body (everything from contentStartIdx to before trailers)
	var bodyEndIdx int
	if trailerStartIdx != -1 && trailerStartIdx > contentStartIdx {
		// Find last non-blank line before trailers
		bodyEndIdx = trailerStartIdx
		for i := trailerStartIdx - 1; i >= contentStartIdx; i-- {
			if strings.TrimSpace(lines[i]) != "" {
				bodyEndIdx = i + 1
				break
			}
		}
	} else if trailerStartIdx == -1 {
		// No trailers, everything is body
		bodyEndIdx = len(lines)
	} else {
		// All content is trailers
		bodyEndIdx = contentStartIdx
	}

	if contentStartIdx < bodyEndIdx {
		bodyLines := lines[contentStartIdx:bodyEndIdx]
		bodyText := strings.Join(bodyLines, "\n")
		bodyText = strings.TrimSpace(bodyText)
		if bodyText != "" {
			body, err := ParseBody(bodyText)
			if err != nil {
				return Message{}, fmt.Errorf("invalid body: %w", err)
			}
			msg.Body = body
		}
	}

	// Extract trailers
	if trailerStartIdx != -1 {
		for i := trailerStartIdx; i < len(lines); i++ {
			line := strings.TrimSpace(lines[i])
			if line == "" {
				continue
			}
			trailer, err := ParseTrailer(line)
			if err != nil {
				// Skip invalid trailer lines
				continue
			}
			msg.Trailers = append(msg.Trailers, trailer)
		}
	}

	return msg, nil
}

// String returns the complete commit message as a formatted string.
//
// This method implements the fmt.Stringer interface and the model.Loggable
// contract. The output follows the Conventional Commits format:
//   <type>[!][(<scope>)]: <subject>
//   [blank line]
//   [body]
//   [blank line]
//   [trailers]
func (m Message) String() string {
	var parts []string

	// Build header: type[(scope)][!]: subject
	header := m.Type.String()
	if !m.Scope.IsZero() {
		header += "(" + m.Scope.String() + ")"
	}
	if m.Breaking {
		header += "!"
	}
	header += ": " + m.Subject.String()
	parts = append(parts, header)

	// Add body if present
	if !m.Body.IsZero() {
		parts = append(parts, "")
		parts = append(parts, m.Body.String())
	}

	// Add trailers if present
	if len(m.Trailers) > 0 {
		parts = append(parts, "")
		for _, trailer := range m.Trailers {
			parts = append(parts, trailer.String())
		}
	}

	return strings.Join(parts, "\n")
}

// Redacted returns a redacted form of the Message suitable for logging.
//
// This method implements the model.Loggable contract. For Message, only
// the header is included in the redacted form (type, scope, subject), without
// the body or trailers to keep logs concise.
func (m Message) Redacted() string {
	header := m.Type.String()
	if !m.Scope.IsZero() {
		header += "(" + m.Scope.String() + ")"
	}
	if m.Breaking {
		header += "!"
	}
	header += ": " + m.Subject.String()
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
