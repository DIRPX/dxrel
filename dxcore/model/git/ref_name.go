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

package git

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"dirpx.dev/dxrel/dxcore/model"
	"gopkg.in/yaml.v3"
)

const (
	// RefNameMinLen is the minimum number of runes allowed in a RefName value.
	//
	// A RefName MUST contain at least one character to be considered
	// non-zero. Single-character ref names are rare but technically valid
	// in Git (for example, a branch named "a").
	RefNameMinLen = 1

	// RefNameMaxLen is the maximum number of runes allowed in a RefName value.
	//
	// While Git itself supports longer reference names, dxrel enforces a
	// practical limit of 256 characters to prevent abuse and ensure
	// reasonable serialization sizes. This limit accommodates deeply nested
	// hierarchical refs such as refs/heads/feature/team/component/task-123.
	RefNameMaxLen = 256
)

const (
	// refNamePattern is the regular expression used to validate Git reference
	// names and revision expressions in dxrel.
	//
	// The pattern is intentionally permissive to support the full range of
	// revision specifiers accepted by 'git rev-parse', including:
	//   - Standard refs: refs/heads/main, refs/tags/v1.0.0, refs/remotes/origin/main
	//   - Special refs: HEAD, FETCH_HEAD, ORIG_HEAD, MERGE_HEAD
	//   - Full commit hashes: 40-character SHA-1 or 64-character SHA-256
	//   - Abbreviated hashes: a1b2c3d (7+ hex characters)
	//   - Revision expressions: HEAD~1, HEAD^, main@{upstream}, main@{2023-01-01}
	//   - Branch names: feature/new-thing, fix-123, user/alice/work
	//
	// The pattern requires:
	//   - At least one character
	//   - Only printable ASCII excluding control characters and certain special chars
	//   - No leading or trailing whitespace (handled by normalization)
	//
	// Characters explicitly forbidden by git-check-ref-format are validated
	// at a higher level if strict Git ref validation is required. This
	// pattern focuses on ensuring the string is well-formed and safe for
	// serialization and logging.
	refNamePattern = `^[a-zA-Z0-9._/@{}\-^~:]+$`
)

var (
	// RefNameRegexp is the compiled regular expression used to validate
	// Git reference names and revision expressions.
	//
	// It is safe for concurrent use by multiple goroutines. Callers SHOULD
	// prefer higher-level helpers such as ParseRefName, RefName.Validate,
	// or internal validation functions rather than using this regexp
	// directly in business logic, so that normalization and error reporting
	// remain consistent across the codebase.
	RefNameRegexp = regexp.MustCompile(refNamePattern)
)

// RefName represents a symbolic Git reference name or revision expression
// understood by 'git rev-parse'. This includes branch names, tag names,
// special refs like HEAD, full or abbreviated commit hashes, and complex
// revision expressions like HEAD~1 or branch@{upstream}.
//
// This type implements the model.Model interface, providing validation,
// serialization to JSON and YAML, safe logging, type identification, and
// zero-value detection. The zero value of RefName (empty string "") is valid
// and represents "no ref specified", indicating that a Git reference has
// not been provided or is not applicable.
//
// RefName values are stored in their original form as provided by the user
// or Git command output, preserving case and structure. The only normalization
// applied is trimming leading and trailing whitespace via strings.TrimSpace.
// Callers MUST resolve symbolic refs to concrete hashes or canonical forms
// via Git commands (git rev-parse, git symbolic-ref) when disambiguation or
// object id retrieval is required.
//
// RefName validation ensures the string is non-empty (when not zero),
// within length limits (1-256 runes), and contains only characters commonly
// accepted in Git reference names and revision expressions. Strict validation
// of git-check-ref-format rules (e.g., no .., no ending with /, no .lock
// suffix) is NOT enforced here, as RefName is intended to support arbitrary
// revision expressions, not just canonical Git refs.
//
// Example values:
//   - Branch: "refs/heads/main", "main", "feature/new-ui"
//   - Tag: "refs/tags/v1.0.0", "v2.1.3"
//   - Remote branch: "refs/remotes/origin/develop", "origin/main"
//   - Special ref: "HEAD", "FETCH_HEAD", "ORIG_HEAD"
//   - Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12"
//   - Revision: "HEAD~3", "main^2", "develop@{yesterday}", "v1.0.0^{}"
type RefName string

// ParseRefName parses a string into a validated RefName value.
//
// ParseRefName applies normalization and validation to the input string.
// The normalization process trims leading and trailing whitespace via
// strings.TrimSpace. After normalization, the resulting RefName is validated
// according to the rules documented in RefName.Validate.
//
// If the input is empty or becomes empty after trimming whitespace,
// ParseRefName returns the zero value (empty RefName) with no error.
//
// If the normalized input fails validation, ParseRefName returns the zero
// value and an error describing the validation failure.
//
// Example usage:
//
//	ref, err := git.ParseRefName("  refs/heads/main  ")
//	// ref = "refs/heads/main", err = nil
//
//	ref, err := git.ParseRefName("HEAD~1")
//	// ref = "HEAD~1", err = nil
//
//	ref, err := git.ParseRefName("")
//	// ref = "", err = nil
//
//	ref, err := git.ParseRefName("invalid\x00ref")
//	// ref = "", err = error
func ParseRefName(s string) (RefName, error) {
	// Normalize: trim whitespace
	normalized := strings.TrimSpace(s)

	// Empty string is valid (zero value)
	if normalized == "" {
		return RefName(""), nil
	}

	refName := RefName(normalized)
	if err := refName.Validate(); err != nil {
		return "", fmt.Errorf("invalid RefName: %w", err)
	}

	return refName, nil
}

// Compile-time assertion that RefName implements model.Model.
var _ model.Model = (*RefName)(nil)

// String returns the RefName value as a string for display or logging.
//
// This method implements the fmt.Stringer interface and the model.Loggable
// contract. For a RefName, String returns the raw ref name or revision
// expression exactly as stored, with no redaction or abbreviation.
//
// If the RefName is zero (empty string), String returns an empty string.
func (rn RefName) String() string {
	return string(rn)
}

// Redacted returns a redacted form of the RefName suitable for logging
// in security-sensitive contexts.
//
// This method implements the model.Loggable contract. For RefName, the
// redacted form is identical to the full value, as Git reference names
// and revision expressions are not considered sensitive information.
// Unlike commit hashes (which use Short() for brevity), refs are
// displayed in full because they are human-readable identifiers.
//
// If the RefName is zero (empty string), Redacted returns an empty string.
func (rn RefName) Redacted() string {
	return string(rn)
}

// TypeName returns the name of this type for error messages, logging,
// and debugging.
//
// This method implements the model.Identifiable contract. It always
// returns "RefName", which is used by higher-level error handling and
// serialization code to construct clear diagnostic messages.
func (rn RefName) TypeName() string {
	return "RefName"
}

// IsZero reports whether this RefName is the zero value.
//
// This method implements the model.ZeroCheckable contract. A RefName is
// considered zero when it is an empty string, representing "no ref specified"
// or "ref not applicable". The zero value is valid and MAY appear in data
// structures where a Git reference is optional.
//
// Example usage:
//
//	var ref git.RefName
//	if ref.IsZero() {
//	    // Handle missing ref
//	}
func (rn RefName) IsZero() bool {
	return rn == ""
}

// Equal reports whether this RefName is equal to another RefName value.
//
// Two RefName values are equal if and only if their string contents are
// identical, using case-sensitive comparison. Git reference names are
// case-sensitive on case-sensitive filesystems (Linux, macOS with case
// sensitivity enabled) and case-preserving on case-insensitive filesystems
// (Windows, macOS default). This method uses exact string comparison to
// match Git's behavior on case-sensitive systems.
//
// Example usage:
//
//	ref1 := git.RefName("refs/heads/main")
//	ref2 := git.RefName("refs/heads/main")
//	ref3 := git.RefName("refs/heads/Main")
//	ref1.Equal(ref2) // returns true
//	ref1.Equal(ref3) // returns false (case differs)
func (rn RefName) Equal(other RefName) bool {
	return rn == other
}

// Validate checks whether this RefName satisfies all structural and
// content requirements for a well-formed Git reference name or revision
// expression.
//
// This method implements the model.Validatable contract. Validate returns
// nil if the RefName is valid, or an error describing the first validation
// failure encountered.
//
// Validation rules:
//   - The zero value (empty string) is valid and represents "no ref specified".
//   - Non-zero values MUST contain at least RefNameMinLen runes (1).
//   - Non-zero values MUST NOT exceed RefNameMaxLen runes (256).
//   - The string MUST match RefNameRegexp (printable ASCII, common ref chars).
//   - Leading and trailing whitespace MUST have been removed via normalization.
//
// Validate does NOT enforce strict git-check-ref-format rules such as:
//   - No consecutive dots (..)
//   - No ending with /
//   - No .lock suffix
//   - No starting with / or containing //
//
// These rules are intentionally relaxed because RefName supports arbitrary
// revision expressions (HEAD~1, branch@{upstream}), not just canonical refs.
// Callers that require strict ref validation SHOULD use git check-ref-format
// or equivalent Git commands.
//
// Example usage:
//
//	ref := git.RefName("refs/heads/feature/new-thing")
//	if err := ref.Validate(); err != nil {
//	    return fmt.Errorf("invalid ref: %w", err)
//	}
func (rn RefName) Validate() error {
	// Zero value is always valid
	if rn.IsZero() {
		return nil
	}

	str := string(rn)

	// Check for leading/trailing whitespace (should have been normalized)
	if strings.TrimSpace(str) != str {
		return fmt.Errorf("RefName %q contains leading or trailing whitespace", str)
	}

	// Validate length constraints
	runeCount := len([]rune(str))
	if runeCount < RefNameMinLen {
		return fmt.Errorf("RefName %q is too short: %d runes (minimum %d)", str, runeCount, RefNameMinLen)
	}
	if runeCount > RefNameMaxLen {
		return fmt.Errorf("RefName %q is too long: %d runes (maximum %d)", str, runeCount, RefNameMaxLen)
	}

	// Validate character set
	if !RefNameRegexp.MatchString(str) {
		return fmt.Errorf("RefName %q contains invalid characters (must match pattern %s)", str, refNamePattern)
	}

	// Check for ASCII control characters and other problematic chars
	for _, r := range str {
		if unicode.IsControl(r) {
			return fmt.Errorf("RefName %q contains control character (U+%04X)", str, r)
		}
		if r > unicode.MaxASCII {
			return fmt.Errorf("RefName %q contains non-ASCII character %q (U+%04X)", str, r, r)
		}
	}

	return nil
}

// MarshalJSON serializes this RefName to JSON.
//
// This method implements the json.Marshaler interface and the
// model.Serializable contract. The RefName is validated before marshaling
// to ensure only well-formed values are written to JSON. If validation
// fails, MarshalJSON returns an error.
//
// The RefName is encoded as a JSON string containing the ref name or
// revision expression exactly as stored. The zero value (empty string)
// marshals to the JSON string "".
//
// Example output:
//
//	{"ref": "refs/heads/main"}
//	{"ref": "HEAD~1"}
//	{"ref": ""}
func (rn RefName) MarshalJSON() ([]byte, error) {
	if err := rn.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", rn.TypeName(), err)
	}
	return json.Marshal(string(rn))
}

// UnmarshalJSON deserializes a RefName from JSON.
//
// This method implements the json.Unmarshaler interface and the
// model.Serializable contract. The input JSON MUST be a string value.
// UnmarshalJSON applies normalization (strings.TrimSpace) and then
// validates the result. If validation fails, the RefName is not modified
// and an error is returned.
//
// Example input:
//
//	{"ref": "refs/heads/main"}
//	{"ref": "  feature/new-ui  "}  // normalized to "feature/new-ui"
//	{"ref": ""}
func (rn *RefName) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return fmt.Errorf("cannot unmarshal JSON into %s: %w", rn.TypeName(), err)
	}

	parsed, err := ParseRefName(str)
	if err != nil {
		return fmt.Errorf("unmarshaled %s is invalid: %w", rn.TypeName(), err)
	}

	*rn = parsed
	return nil
}

// MarshalYAML serializes this RefName to YAML.
//
// This method implements the yaml.Marshaler interface and the
// model.Serializable contract. The RefName is validated before marshaling
// to ensure only well-formed values are written to YAML. If validation
// fails, MarshalYAML returns an error.
//
// The RefName is encoded as a YAML string containing the ref name or
// revision expression exactly as stored. The zero value (empty string)
// marshals to the YAML string "".
//
// A type alias is used internally to prevent infinite recursion during
// marshaling.
//
// Example output:
//
//	ref: refs/heads/main
//	ref: HEAD~1
//	ref: ""
func (rn RefName) MarshalYAML() (interface{}, error) {
	if err := rn.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", rn.TypeName(), err)
	}
	// Use type alias to avoid infinite recursion
	type refName RefName
	return refName(rn), nil
}

// UnmarshalYAML deserializes a RefName from YAML.
//
// This method implements the yaml.Unmarshaler interface and the
// model.Serializable contract. The input YAML MUST be a string value.
// UnmarshalYAML applies normalization (strings.TrimSpace) and then
// validates the result. If validation fails, the RefName is not modified
// and an error is returned.
//
// Example input:
//
//	ref: refs/heads/main
//	ref: "  feature/new-ui  "  # normalized to "feature/new-ui"
//	ref: ""
func (rn *RefName) UnmarshalYAML(node *yaml.Node) error {
	var str string
	if err := node.Decode(&str); err != nil {
		return fmt.Errorf("cannot unmarshal YAML into %s: %w", rn.TypeName(), err)
	}

	parsed, err := ParseRefName(str)
	if err != nil {
		return fmt.Errorf("unmarshaled %s is invalid: %w", rn.TypeName(), err)
	}

	*rn = parsed
	return nil
}
