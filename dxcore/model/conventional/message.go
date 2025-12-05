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
//   <type>[(<scope>)][!]: <subject>
//   [blank line]
//   [body]
//   [blank line]
//   [Trailer-Key: trailer value]
//   [Trailer-Key: trailer value]
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
//   Simple message (header only):
//     "feat: add user authentication"
//
//   With scope:
//     "fix(api): resolve timeout issue"
//
//   With breaking change:
//     "feat!: remove deprecated endpoint"
//     "feat(api)!: change authentication flow"
//
//   With body:
//     "feat: add caching\n\nImproves performance significantly"
//
//   With trailers:
//     "fix: resolve bug\n\nFixes: #123\nReviewed-by: Alice"
//
//   Complete message:
//     "feat(api)!: add user endpoint\n\nAdds new REST API.\n\nFixes: #456\nSigned-off-by: Bob"
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
	//   - "BREAKING CHANGE:" (space) is the canonical Conventional Commits format but is
	//     NOT a valid git trailer (git-interpret-trailers rejects spaces in keys)
	//   - "BREAKING-CHANGE:" (hyphen) is git-compatible and can be parsed as a Trailer
	//   - ParseMessage detects BOTH formats and sets Breaking=true for either
	//   - Only "BREAKING-CHANGE:" (hyphen) will appear in the Trailers slice
	//   - "BREAKING CHANGE:" (space) is detected but not added to Trailers
	//
	// When true, this commit SHOULD trigger a major version bump in semantic versioning
	// (unless version is 0.x.y, where breaking changes only bump minor version).
	// Breaking changes SHOULD be explained in the Body or in the BREAKING CHANGE/BREAKING-CHANGE value.
	//
	// Examples:
	//   - "feat!: remove endpoint" -> Breaking=true (from "!" marker)
	//   - "feat: add\n\nBREAKING CHANGE: removes API" -> Breaking=true (space format)
	//   - "feat: add\n\nBREAKING-CHANGE: removes API" -> Breaking=true (hyphen format, also in Trailers)
	//   - "feat!: change\n\nBREAKING CHANGE: explanation" -> Breaking=true (multiple sources)
	//   - "feat: add feature" -> Breaking=false (no markers)
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
//   <type>[(<scope>)][!]: <subject>
//   [blank line]
//   [body paragraphs]
//   [blank line]
//   [Trailer-Key: trailer value]
//   [Trailer-Key: trailer value]
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
//   1. Header contains "!" after type/scope: "feat!:" or "feat(scope)!:"
//   2. Footer contains "BREAKING CHANGE:" (with space, Conventional Commits canonical format)
//   3. Footer contains "BREAKING-CHANGE:" (with hyphen, git trailer compatible format)
//
// All three mechanisms are equally valid and result in Breaking=true. The "!" marker
// is detected during header parsing (Stage 4), while footer formats are detected
// during trailer extraction (Stage 8).
//
// Important: "BREAKING CHANGE:" (with space) is NOT a valid git trailer because
// git-interpret-trailers rejects spaces in keys. However, it's the canonical format
// in the Conventional Commits spec, so we detect it via string matching. Only
// "BREAKING-CHANGE:" (hyphen) can be parsed as a proper Trailer and added to the
// Trailers slice.
//
// Trailer Detection Algorithm:
//
// ParseMessage uses a backwards-scan algorithm to distinguish body from trailers:
//   1. Find last non-blank line
//   2. Scan backwards checking if each line matches trailer format (Key: value)
//   3. Stop at first non-trailer line or blank line separating trailers from body
//   4. If ALL non-blank lines are trailers, entire content is trailers (no body)
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
	// === Stage 1: Input validation ===
	if s == "" {
		return Message{}, fmt.Errorf("message cannot be empty")
	}

	// === Stage 2: Normalization ===
	// Convert Windows (CRLF) line endings to Unix (LF) for consistent parsing.
	// Also remove any leading/trailing whitespace from the entire message.
	normalized := strings.ReplaceAll(s, "\r\n", "\n")
	normalized = strings.TrimSpace(normalized)

	// Split normalized message into lines for line-by-line processing
	lines := strings.Split(normalized, "\n")
	if len(lines) == 0 {
		return Message{}, fmt.Errorf("message cannot be empty")
	}

	// === Stage 3: Header parsing ===
	// The first line MUST be a valid Conventional Commit header.
	// Format: <type>[(<scope>)][!]: <subject>
	header := strings.TrimSpace(lines[0])
	matches := MessageHeaderRegexp.FindStringSubmatch(header)
	if matches == nil {
		return Message{}, fmt.Errorf("invalid Conventional Commit header format: %q", header)
	}

	// Extract components from regex capture groups (see messageHeaderPattern for group definitions)
	typeStr := matches[1]       // Group 1: type (feat, fix, etc.)
	scopeStr := matches[2]      // Group 2: scope (optional, without parens)
	breakingMarker := matches[3] // Group 3: breaking change marker "!" (optional)
	subjectStr := matches[4]    // Group 4: subject (remainder after colon)

	// === Stage 4: Component validation ===
	// Parse and validate the type string into a Type constant
	commitType, err := ParseType(typeStr)
	if err != nil {
		return Message{}, fmt.Errorf("invalid type: %w", err)
	}

	// Parse and validate scope if present (empty string means no scope)
	var scope Scope
	if scopeStr != "" {
		scope, err = ParseScope(scopeStr)
		if err != nil {
			return Message{}, fmt.Errorf("invalid scope: %w", err)
		}
	}

	// Parse and validate the subject (required, must be non-empty)
	subject, err := ParseSubject(subjectStr)
	if err != nil {
		return Message{}, fmt.Errorf("invalid subject: %w", err)
	}

	// Convert breaking change marker to boolean
	breaking := breakingMarker == "!"

	// Initialize message with all header components
	msg := Message{
		Type:     commitType,
		Scope:    scope,
		Subject:  subject,
		Breaking: breaking,
	}

	// If message is header-only (single line), parsing is complete
	if len(lines) == 1 {
		return msg, nil
	}

	// === Stage 5: Content detection ===
	// Find the first non-blank line after the header. This marks the start
	// of either body or trailers (we'll determine which in the next stage).
	contentStartIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "" {
			contentStartIdx = i
			break
		}
	}

	if contentStartIdx == -1 {
		// No body or trailers, only blank lines after header
		return msg, nil
	}

	// === Stage 6: Trailer detection (backwards scan algorithm) ===
	// We use backwards scanning to distinguish trailers from body because:
	// 1. Trailers MUST be at the end of the message
	// 2. Trailers are separated from body by a blank line
	// 3. Body can contain text that looks like trailers, but only end-of-message trailers count
	//
	// This helper checks if a line looks like a trailer (either git format or BREAKING CHANGE).
	// Two formats are considered trailers:
	//   1. Git trailer format: "Key: value" where Key matches ^[A-Za-z][A-Za-z0-9-]*$
	//   2. BREAKING CHANGE format: "BREAKING CHANGE: ..." (Conventional Commits canonical format)
	isTrailerLine := func(line string) bool {
		line = strings.TrimSpace(line)
		if line == "" {
			return false
		}

		// Check for BREAKING CHANGE with space (Conventional Commits format)
		if strings.HasPrefix(line, "BREAKING CHANGE:") || strings.HasPrefix(line, "BREAKING CHANGE ") {
			return true
		}

		// Check for standard git trailer format
		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			return false // No colon, can't be a trailer
		}
		key := strings.TrimSpace(line[:colonIdx])
		return TrailerKeyRegexp.MatchString(key) // Check key against ^[A-Za-z][A-Za-z0-9-]*$
	}

	trailerStartIdx := -1 // Will hold the line index where trailers begin (or -1 if no trailers)
	lastNonBlankIdx := -1 // Last non-blank line in the message

	// Find the last non-blank line (working backwards from end of message)
	for i := len(lines) - 1; i >= contentStartIdx; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			lastNonBlankIdx = i
			break
		}
	}

	if lastNonBlankIdx != -1 {
		// Scan backwards from last non-blank line to find where trailer block starts
		inTrailers := true // Assume we're in trailers until we find a non-trailer
		for i := lastNonBlankIdx; i >= contentStartIdx; i-- {
			line := strings.TrimSpace(lines[i])

			if line == "" {
				// Blank line encountered while scanning backwards
				if inTrailers && trailerStartIdx == -1 {
					// This blank line separates trailers from body
					// Trailers start on the next line (i+1)
					trailerStartIdx = i + 1
					break
				}
				// Skip blank lines within trailer block
				continue
			}

			if !isTrailerLine(lines[i]) {
				// Found a non-trailer line, so we're no longer in the trailer block
				inTrailers = false
			}
		}

		// Special case: if ALL non-blank lines from contentStartIdx to end are trailers,
		// then the entire content is trailers (no body). This handles messages like:
		//   "fix: bug\n\nFixes: #123\nReviewed-by: Alice"
		if inTrailers && trailerStartIdx == -1 {
			trailerStartIdx = contentStartIdx
		}
	}

	// === Stage 7: Body extraction ===
	// Extract body text from the region between header and trailers.
	// Body is everything after the header and before the trailer block (if present).
	var bodyEndIdx int

	if trailerStartIdx != -1 && trailerStartIdx > contentStartIdx {
		// Trailers detected: body ends where trailers start.
		// Trim any blank lines between last body line and trailer separator.
		bodyEndIdx = trailerStartIdx
		for i := trailerStartIdx - 1; i >= contentStartIdx; i-- {
			if strings.TrimSpace(lines[i]) != "" {
				bodyEndIdx = i + 1
				break
			}
		}
	} else if trailerStartIdx == -1 {
		// No trailers detected: everything after header is body
		bodyEndIdx = len(lines)
	} else {
		// All content is trailers (trailerStartIdx == contentStartIdx)
		bodyEndIdx = contentStartIdx
	}

	// Extract and parse body if present
	if contentStartIdx < bodyEndIdx {
		bodyLines := lines[contentStartIdx:bodyEndIdx]
		bodyText := strings.Join(bodyLines, "\n")
		bodyText = strings.TrimSpace(bodyText) // Remove leading/trailing whitespace

		if bodyText != "" {
			body, err := ParseBody(bodyText)
			if err != nil {
				return Message{}, fmt.Errorf("invalid body: %w", err)
			}
			msg.Body = body
		}
	}

	// === Stage 8: Trailer extraction and breaking change detection ===
	// Parse each trailer line into a Trailer struct. Invalid trailer lines
	// are silently skipped to allow flexibility in formatting.
	//
	// Additionally, detect BREAKING CHANGE markers in footer. Per Conventional Commits
	// spec v1.0.0, both "BREAKING CHANGE:" and "BREAKING-CHANGE:" indicate breaking changes.
	if trailerStartIdx != -1 {
		for i := trailerStartIdx; i < len(lines); i++ {
			line := strings.TrimSpace(lines[i])
			if line == "" {
				continue // Skip blank lines within trailer block
			}

			// Check for BREAKING CHANGE with space (Conventional Commits spec format).
			// This format is widely used but doesn't conform to git trailer format,
			// so it won't parse as a valid Trailer. We detect it separately.
			if strings.HasPrefix(line, "BREAKING CHANGE:") || strings.HasPrefix(line, "BREAKING CHANGE ") {
				msg.Breaking = true
				// Don't try to parse as trailer since space in key is invalid
				continue
			}

			// Try to parse as standard git trailer (e.g., "BREAKING-CHANGE:", "Fixes:", etc.)
			trailer, err := ParseTrailer(line)
			if err != nil {
				// Skip malformed trailer lines rather than failing the entire parse.
				// This allows messages with some invalid trailers to still be parsed.
				continue
			}
			msg.Trailers = append(msg.Trailers, trailer)

			// Check for BREAKING-CHANGE trailer (git-compatible format with hyphen).
			// This is the git-interpret-trailers compatible version of "BREAKING CHANGE".
			if trailer.Key == "BREAKING-CHANGE" {
				msg.Breaking = true
			}
		}
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
//   <type>[(<scope>)][!]: <subject>
//   [blank line]
//   [body]
//   [blank line]
//   [Trailer-Key: trailer value]
//   [Trailer-Key: trailer value]
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
//   Simple message:
//     Input:  Message{Type: Feat, Subject: "add feature"}
//     Output: "feat: add feature"
//
//   With scope and breaking:
//     Input:  Message{Type: Fix, Scope: "api", Breaking: true, Subject: "change flow"}
//     Output: "fix(api)!: change flow"
//
//   Complete message:
//     Input:  Message{Type: Feat, Scope: "api", Subject: "add endpoint",
//                     Body: "Adds new API", Trailers: [{Key: "Fixes", Value: "#123"}]}
//     Output: "feat(api): add endpoint\n\nAdds new API\n\nFixes: #123"
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
		parts = append(parts, "")            // Blank line before body
		parts = append(parts, m.Body.String())
	}

	// Add trailers if present, with blank line separator
	if len(m.Trailers) > 0 {
		parts = append(parts, "")            // Blank line before trailers
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
//   <type>[(<scope>)][!]: <subject>
//
// This format is identical to the first line of String() output.
//
// Example Outputs:
//
//   Input:  Message{Type: Feat, Subject: "add feature"}
//   Output: "feat: add feature"
//
//   Input:  Message{Type: Fix, Scope: "api", Breaking: true, Subject: "change flow",
//                   Body: "Long explanation...", Trailers: [...]}
//   Output: "fix(api)!: change flow"
//
// Use Cases:
//
//   - Structured logging: log.Info("processing commit", "message", msg.Redacted())
//   - Error messages: fmt.Errorf("failed to apply %s", msg.Redacted())
//   - Metrics labels: metrics.RecordCommit(msg.Type.String(), msg.Redacted())
//   - Audit logs: audit.Log("commit parsed", "header", msg.Redacted())
func (m Message) Redacted() string {
	// Build header only: type[(scope)][!]: subject
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
