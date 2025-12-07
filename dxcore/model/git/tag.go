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
	// TagNameMinLen is the minimum number of runes allowed in a TagName value.
	//
	// A TagName MUST contain at least one character to be considered non-zero.
	// Single-character tag names are rare but technically valid in Git (for
	// example, a tag named "v").
	//
	// This constraint applies to the number of Unicode code points (runes) in
	// the string, not the number of bytes. Multi-byte UTF-8 characters count
	// as single runes for length calculation purposes.
	TagNameMinLen = 1

	// TagNameMaxLen is the maximum number of runes allowed in a TagName value.
	//
	// While Git itself supports longer tag names, dxrel enforces a practical
	// limit of 256 characters to prevent abuse and ensure reasonable
	// serialization sizes. This limit accommodates deeply nested hierarchical
	// tags such as "moduleA/v1.2.3" or "experimental/feature/test-tag".
	//
	// This constraint applies to the number of Unicode code points (runes) in
	// the string, not the number of bytes. Multi-byte UTF-8 characters count
	// as single runes for length calculation purposes.
	TagNameMaxLen = 256

	// TagMessageMaxLen is the maximum allowed length for a tag message in
	// annotated tags, measured in bytes.
	//
	// This limit prevents abuse and ensures that tag messages remain
	// reasonable in size. A limit of 64KB (65536 bytes) provides ample space
	// for detailed release notes, changelogs, and other tag annotations while
	// preventing excessively large messages that could impact performance or
	// storage.
	//
	// For lightweight tags, the Message field SHOULD be empty and this limit
	// does not apply.
	TagMessageMaxLen = 65536 // 64KB
)

const (
	// tagNamePattern is the regular expression used to validate Git tag names
	// in dxrel.
	//
	// The pattern is intentionally permissive to support the full range of
	// tag naming conventions used in Git repositories, including:
	//   - Simple version tags: v1.0.0, v2.3.4-rc.1
	//   - Hierarchical tags: moduleA/v1.2.3, platform/services/v1.0.0
	//   - Custom naming: release-2023-01-15, build-42, experimental
	//   - Semver with build metadata: v1.2.3+build.42, v2.0.0-rc.1+sha.abc123
	//
	// The pattern requires:
	//   - At least one character
	//   - Only printable ASCII excluding control characters and certain special chars
	//   - No leading or trailing whitespace (handled by normalization)
	//
	// Characters explicitly forbidden by git-check-ref-format are validated
	// at a higher level if strict Git tag validation is required. This pattern
	// focuses on ensuring the string is well-formed and safe for serialization
	// and logging.
	tagNamePattern = `^[a-zA-Z0-9._/@{}\-^~:+]+$`
)

var (
	// TagNameRegexp is the compiled regular expression used to validate
	// Git tag names.
	//
	// It is safe for concurrent use by multiple goroutines. Callers SHOULD
	// prefer higher-level helpers such as ParseTagName, TagName.Validate,
	// or internal validation functions rather than using this regexp directly
	// in business logic, so that normalization and error reporting remain
	// consistent across the codebase.
	TagNameRegexp = regexp.MustCompile(tagNamePattern)
)

// TagName represents a Git tag name without the "refs/tags/" prefix.
//
// This type implements the model.Model interface, providing validation,
// serialization to JSON and YAML, safe logging, type identification, and
// zero-value detection. The zero value of TagName (empty string "") is valid
// and represents "no tag specified", indicating that a Git tag has not been
// provided or is not applicable.
//
// TagName values are stored in their original form as provided by the user
// or Git command output, preserving case and structure. The only normalization
// applied is trimming leading and trailing whitespace via strings.TrimSpace.
//
// TagName validation ensures the string is non-empty (when not zero), within
// length limits (1-256 runes), and contains only characters commonly accepted
// in Git tag names. Strict validation of git-check-ref-format rules (e.g.,
// no .., no ending with /, no .lock suffix) is NOT enforced here, as TagName
// is intended to support various tag naming conventions.
//
// Example values:
//   - Semantic version: "v1.2.3", "v2.0.0-rc.1"
//   - Hierarchical: "moduleA/v1.2.3", "platform/services/v1.0.0"
//   - Custom: "release-2023-01-15", "build-42", "experimental"
type TagName string

// ParseTagName parses a string into a validated TagName value, normalizing
// and validating the input before returning. This function provides a unified
// parsing entry point for converting external string representations into
// TagName values with comprehensive input validation.
//
// ParseTagName applies normalization to the input by removing leading and
// trailing whitespace via strings.TrimSpace. After normalization, the
// resulting TagName is validated according to the rules documented in
// TagName.Validate.
//
// If the input is empty or becomes empty after trimming whitespace,
// ParseTagName returns the zero value (empty TagName) with no error.
//
// If the normalized input fails validation, ParseTagName returns the zero
// value and an error describing the validation failure.
//
// Callers MUST check the returned error before using the TagName value.
// This function is pure and has no side effects. It is safe to call
// concurrently from multiple goroutines.
//
// Example:
//
//	tag, err := git.ParseTagName("  v1.2.3  ")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(tag) // Output: v1.2.3
//
//	tag, err := git.ParseTagName("")
//	// tag = "", err = nil (zero value)
//
//	tag, err := git.ParseTagName("invalid\x00tag")
//	// tag = "", err = error
func ParseTagName(s string) (TagName, error) {
	// Normalize: trim whitespace
	normalized := strings.TrimSpace(s)

	// Empty string is valid (zero value)
	if normalized == "" {
		return TagName(""), nil
	}

	tagName := TagName(normalized)
	if err := tagName.Validate(); err != nil {
		return "", fmt.Errorf("invalid TagName: %w", err)
	}

	return tagName, nil
}

// Compile-time assertion that TagName implements model.Model.
var _ model.Model = (*TagName)(nil)

// String returns the TagName value as a string for display or logging.
//
// This method implements the fmt.Stringer interface and the model.Loggable
// contract. For a TagName, String returns the raw tag name exactly as stored,
// with no redaction or abbreviation.
//
// If the TagName is zero (empty string), String returns an empty string.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently.
//
// Example:
//
//	tag := git.TagName("v1.2.3")
//	fmt.Println(tag.String()) // Output: v1.2.3
func (tn TagName) String() string {
	return string(tn)
}

// Redacted returns a redacted form of the TagName suitable for logging
// in security-sensitive contexts.
//
// This method implements the model.Loggable contract. For TagName, the
// redacted form is identical to the full value, as Git tag names are not
// considered sensitive information. Tag names are public identifiers for
// releases and versions.
//
// If the TagName is zero (empty string), Redacted returns an empty string.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently.
//
// Example:
//
//	tag := git.TagName("v1.2.3")
//	log.Info("processing tag", "tag", tag.Redacted()) // Safe for production logs
func (tn TagName) Redacted() string {
	return string(tn)
}

// TypeName returns the name of this type for error messages, logging,
// and debugging.
//
// This method implements the model.Identifiable contract. It always
// returns "TagName", which is used by higher-level error handling and
// serialization code to construct clear diagnostic messages.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently. The returned string is a constant literal,
// ensuring zero allocations.
func (tn TagName) TypeName() string {
	return "TagName"
}

// IsZero reports whether this TagName is the zero value.
//
// This method implements the model.ZeroCheckable contract. A TagName is
// considered zero when it is an empty string, representing "no tag specified"
// or "tag not applicable". The zero value is valid and MAY appear in data
// structures where a Git tag is optional.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently.
//
// Example:
//
//	var tag git.TagName
//	if tag.IsZero() {
//	    // Handle missing tag
//	}
func (tn TagName) IsZero() bool {
	return tn == ""
}

// Equal reports whether this TagName is equal to another TagName value.
//
// Two TagName values are equal if and only if their string contents are
// identical, using case-sensitive comparison. Git tag names are case-sensitive
// on case-sensitive filesystems (Linux, macOS with case sensitivity enabled)
// and case-preserving on case-insensitive filesystems (Windows, macOS default).
// This method uses exact string comparison to match Git's behavior on
// case-sensitive systems.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently. The comparison is fast, deterministic,
// and performs no allocations.
//
// Example:
//
//	tag1 := git.TagName("v1.2.3")
//	tag2 := git.TagName("v1.2.3")
//	tag3 := git.TagName("V1.2.3")
//	fmt.Println(tag1.Equal(tag2)) // Output: true
//	fmt.Println(tag1.Equal(tag3)) // Output: false (case differs)
func (tn TagName) Equal(other TagName) bool {
	return tn == other
}

// Validate checks whether this TagName satisfies all structural and
// content requirements for a well-formed Git tag name.
//
// This method implements the model.Validatable contract. Validate returns
// nil if the TagName is valid, or an error describing the first validation
// failure encountered.
//
// Validation rules:
//   - The zero value (empty string) is valid and represents "no tag specified".
//   - Non-zero values MUST contain at least TagNameMinLen runes (1).
//   - Non-zero values MUST NOT exceed TagNameMaxLen runes (256).
//   - The string MUST match TagNameRegexp (printable ASCII, common tag chars).
//   - Leading and trailing whitespace MUST have been removed via normalization.
//
// Validate does NOT enforce strict git-check-ref-format rules such as:
//   - No consecutive dots (..)
//   - No ending with /
//   - No .lock suffix
//   - No starting with / or containing //
//
// These rules are intentionally relaxed because TagName supports various
// tag naming conventions. Callers that require strict tag validation SHOULD
// use git check-ref-format or equivalent Git commands.
//
// This method MUST be fast, deterministic, and idempotent. It MUST NOT mutate
// the receiver, MUST NOT have side effects, and MUST be safe to call concurrently.
// Validation does not perform I/O or allocate memory except when constructing
// error messages for invalid values.
//
// Example:
//
//	tag := git.TagName("v1.2.3")
//	if err := tag.Validate(); err != nil {
//	    return fmt.Errorf("invalid tag: %w", err)
//	}
func (tn TagName) Validate() error {
	// Zero value is always valid
	if tn.IsZero() {
		return nil
	}

	str := string(tn)

	// Check for leading/trailing whitespace (should have been normalized)
	if strings.TrimSpace(str) != str {
		return fmt.Errorf("TagName %q contains leading or trailing whitespace", str)
	}

	// Validate length constraints
	runeCount := len([]rune(str))
	if runeCount < TagNameMinLen {
		return fmt.Errorf("TagName %q is too short: %d runes (minimum %d)", str, runeCount, TagNameMinLen)
	}
	if runeCount > TagNameMaxLen {
		return fmt.Errorf("TagName %q is too long: %d runes (maximum %d)", str, runeCount, TagNameMaxLen)
	}

	// Validate character set
	if !TagNameRegexp.MatchString(str) {
		return fmt.Errorf("TagName %q contains invalid characters (must match pattern %s)", str, tagNamePattern)
	}

	// Check for ASCII control characters and other problematic chars
	for _, r := range str {
		if unicode.IsControl(r) {
			return fmt.Errorf("TagName %q contains control character (U+%04X)", str, r)
		}
		if r > unicode.MaxASCII {
			return fmt.Errorf("TagName %q contains non-ASCII character %q (U+%04X)", str, r, r)
		}
	}

	return nil
}

// MarshalJSON implements json.Marshaler, serializing the TagName to its
// string representation as a JSON string. This method satisfies part of the
// model.Serializable interface requirement.
//
// MarshalJSON first validates that the TagName conforms to all constraints
// by calling Validate. If validation fails, marshaling fails with the
// validation error, preventing invalid data from being serialized. If
// validation succeeds, the TagName is encoded as a JSON string.
//
// The zero value (empty string) marshals to the JSON string "".
//
// This method MUST NOT mutate the receiver except as required by the
// json.Marshaler interface contract. It MUST be safe to call concurrently
// on immutable receivers.
//
// Example:
//
//	tag := git.TagName("v1.2.3")
//	data, _ := json.Marshal(tag)
//	fmt.Println(string(data)) // Output: "v1.2.3"
func (tn TagName) MarshalJSON() ([]byte, error) {
	if err := tn.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", tn.TypeName(), err)
	}
	return json.Marshal(string(tn))
}

// UnmarshalJSON implements json.Unmarshaler, deserializing a JSON string
// into a TagName value. This method satisfies part of the model.Serializable
// interface requirement.
//
// UnmarshalJSON accepts JSON strings containing tag names and applies
// normalization (strings.TrimSpace) before validation. If validation fails,
// the TagName is not modified and an error is returned.
//
// Empty JSON strings unmarshal successfully to the zero value TagName.
//
// The method mutates the receiver to store the unmarshaled value. It is not
// safe for concurrent use during unmarshaling, though the resulting TagName
// value is safe for concurrent reads after unmarshaling completes.
//
// Example:
//
//	var tag git.TagName
//	json.Unmarshal([]byte(`"v1.2.3"`), &tag)
//	fmt.Println(tag.String()) // Output: v1.2.3
func (tn *TagName) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return fmt.Errorf("cannot unmarshal JSON into %s: %w", tn.TypeName(), err)
	}

	parsed, err := ParseTagName(str)
	if err != nil {
		return fmt.Errorf("unmarshaled %s is invalid: %w", tn.TypeName(), err)
	}

	*tn = parsed
	return nil
}

// MarshalYAML implements yaml.Marshaler, serializing the TagName to its
// string representation for YAML encoding. This method satisfies part of the
// model.Serializable interface requirement.
//
// MarshalYAML first validates that the TagName conforms to all constraints
// by calling Validate. If validation fails, marshaling fails with the
// validation error, preventing invalid data from being serialized. If
// validation succeeds, the TagName is encoded as a YAML string.
//
// The zero value (empty string) marshals to the YAML string "".
//
// A type alias is used internally to prevent infinite recursion during
// marshaling.
//
// This method MUST NOT mutate the receiver except as required by the
// yaml.Marshaler interface contract. It MUST be safe to call concurrently
// on immutable receivers.
//
// Example:
//
//	tag := git.TagName("v1.2.3")
//	data, _ := yaml.Marshal(tag)
//	fmt.Println(string(data)) // Output: v1.2.3\n
func (tn TagName) MarshalYAML() (interface{}, error) {
	if err := tn.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", tn.TypeName(), err)
	}
	// Use type alias to avoid infinite recursion
	type tagName TagName
	return tagName(tn), nil
}

// UnmarshalYAML implements yaml.Unmarshaler, deserializing a YAML scalar
// into a TagName value. This method satisfies part of the model.Serializable
// interface requirement.
//
// UnmarshalYAML accepts YAML scalars containing tag names and applies
// normalization (strings.TrimSpace) before validation. If validation fails,
// the TagName is not modified and an error is returned.
//
// Empty YAML scalars unmarshal successfully to the zero value TagName.
//
// The method mutates the receiver to store the unmarshaled value. It is not
// safe for concurrent use during unmarshaling, though the resulting TagName
// value is safe for concurrent reads after unmarshaling completes.
//
// Example:
//
//	var tag git.TagName
//	yaml.Unmarshal([]byte("v1.2.3"), &tag)
//	fmt.Println(tag.String()) // Output: v1.2.3
func (tn *TagName) UnmarshalYAML(node *yaml.Node) error {
	var str string
	if err := node.Decode(&str); err != nil {
		return fmt.Errorf("cannot unmarshal YAML into %s: %w", tn.TypeName(), err)
	}

	parsed, err := ParseTagName(str)
	if err != nil {
		return fmt.Errorf("unmarshaled %s is invalid: %w", tn.TypeName(), err)
	}

	*tn = parsed
	return nil
}

// Tag represents a Git tag resolved from the repository, combining the tag
// name with object and commit hashes, annotation status, and optional message.
//
// This type implements the model.Model interface, providing validation,
// serialization to JSON and YAML, safe logging, type identification, and
// zero-value detection.
//
// A Tag intentionally does not embed any semantic versioning or dxrel-specific
// semantics. Higher layers are responsible for parsing version information
// from Tag.Name if needed (e.g., extracting a semver.Version from "v1.2.3").
//
// Tags in Git come in two forms:
//   - Lightweight tags: Simple references that point directly to a commit.
//     For these, Object == Commit and Annotated == false.
//   - Annotated tags: Tag objects that contain metadata (tagger, date, message)
//     and point to another object (usually a commit). For these, Object is the
//     tag object hash, Commit is the peeled commit hash (after dereferencing),
//     and Annotated == true.
//
// The zero value of Tag (all fields zero) is valid at the Go type level but
// represents "no tag specified" and will fail validation if Validate() is called.
//
// Example values:
//
//	// Lightweight tag
//	Tag{
//	    Name:      "v1.2.3",
//	    Object:    "a1b2c3d4e5f67890abcdef1234567890abcdef12",
//	    Commit:    "a1b2c3d4e5f67890abcdef1234567890abcdef12",
//	    Annotated: false,
//	    Message:   "",
//	}
//
//	// Annotated tag
//	Tag{
//	    Name:      "v2.0.0",
//	    Object:    "1234567890abcdef1234567890abcdef12345678",
//	    Commit:    "abcdef1234567890abcdef1234567890abcdef12",
//	    Annotated: true,
//	    Message:   "Release v2.0.0\n\nMajor release with breaking changes...",
//	}
type Tag struct {
	// Name is the short tag name without "refs/tags/" prefix.
	//
	// This preserves the exact form of the tag name as it appears in Git,
	// whether that's a simple version like "v1.2.3", a hierarchical name
	// like "moduleA/v1.2.3", or a custom identifier like "release-2023-01-15".
	//
	// Examples:
	//   - "v1.2.3"
	//   - "rxlog/v1.4.0-rc.1"
	//   - "experimental"
	Name TagName `json:"name" yaml:"name"`

	// Object is the hash of the tag object itself.
	//
	// For lightweight tags, Object equals Commit because lightweight tags
	// are just references that point directly to commits without creating
	// a separate tag object.
	//
	// For annotated tags, Object refers to the tag object hash (the object
	// that contains the tag metadata like tagger, date, and message), and
	// Commit refers to the underlying commit hash after peeling the tag.
	//
	// The Object hash MUST be a fully resolved Git object id in canonical
	// form (40-character SHA-1 or 64-character SHA-256 in lowercase hex).
	Object Hash `json:"object" yaml:"object"`

	// Commit is the commit that this tag ultimately refers to.
	//
	// For lightweight tags, this is equal to Object because the tag points
	// directly to the commit.
	//
	// For annotated tags, this is the peeled commit hash obtained by
	// dereferencing the tag object. Git's "tag^{}" or "tag^{commit}" notation
	// performs this peeling operation.
	//
	// The Commit hash MUST be a fully resolved Git object id in canonical
	// form (40-character SHA-1 or 64-character SHA-256 in lowercase hex).
	Commit Hash `json:"commit" yaml:"commit"`

	// Annotated reports whether this tag is an annotated tag (true) or a
	// lightweight tag (false).
	//
	// Annotated tags are tag objects that contain metadata (tagger identity,
	// timestamp, and message) in addition to pointing to a commit. They are
	// created with "git tag -a" or similar commands.
	//
	// Lightweight tags are simple references that point directly to commits
	// without additional metadata. They are created with "git tag" (without -a).
	//
	// This field determines how Object and Commit are interpreted:
	//   - If Annotated == false: Object == Commit (both point to the same commit)
	//   - If Annotated == true: Object != Commit (Object is tag, Commit is peeled)
	Annotated bool `json:"annotated" yaml:"annotated"`

	// Message is the tag message for annotated tags.
	//
	// For annotated tags, this field contains the message/annotation text
	// provided when the tag was created (typically release notes, changelog,
	// or other descriptive text).
	//
	// For lightweight tags, this field SHOULD be empty as lightweight tags
	// do not have associated messages. If Message is non-empty for a
	// lightweight tag (Annotated == false), validation will fail to prevent
	// inconsistent state.
	//
	// Maximum length is TagMessageMaxLen (64KB).
	Message string `json:"message,omitempty" yaml:"message,omitempty"`
}

// Compile-time assertion that Tag implements model.Model.
var _ model.Model = (*Tag)(nil)

// NewTag creates a new Tag with the given components, validating the result
// before returning.
//
// This is a convenience constructor that creates and validates a Tag in one
// step, ensuring that all components conform to the validation rules defined
// by Tag.Validate. If any of the components are invalid or the combination
// violates Tag invariants, NewTag returns a zero Tag and an error describing
// the validation failure.
//
// For lightweight tags, pass the same hash for both object and commit, set
// annotated to false, and pass an empty string for message.
//
// For annotated tags, pass different hashes for object and commit (where
// object is the tag object hash and commit is the peeled commit hash), set
// annotated to true, and provide the tag message.
//
// This function is pure and has no side effects. It is safe to call
// concurrently from multiple goroutines.
//
// Example usage:
//
//	// Lightweight tag
//	tag, err := git.NewTag("v1.2.3", hash, hash, false, "")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Annotated tag
//	tag, err := git.NewTag("v2.0.0", tagHash, commitHash, true, "Release notes...")
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewTag(name TagName, object Hash, commit Hash, annotated bool, message string) (Tag, error) {
	tag := Tag{
		Name:      name,
		Object:    object,
		Commit:    commit,
		Annotated: annotated,
		Message:   message,
	}

	if err := tag.Validate(); err != nil {
		return Tag{}, err
	}

	return tag, nil
}

// String returns the human-readable representation of the Tag for display
// and debugging purposes. This method implements the fmt.Stringer interface
// and satisfies the model.Loggable contract's String() requirement.
//
// The output format includes all components of the Tag:
//
//	Tag{Name:<name>, Object:<object>, Commit:<commit>, Annotated:<annotated>}
//
// The Message field is omitted from String() output for brevity, as tag
// messages can be very long. Callers needing the message should access the
// Message field directly.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently.
//
// Example:
//
//	tag := git.Tag{Name: "v1.2.3", Object: hash, Commit: hash, Annotated: false}
//	fmt.Println(tag.String())
//	// Output: Tag{Name:v1.2.3, Object:a1b2c3d..., Commit:a1b2c3d..., Annotated:false}
func (t Tag) String() string {
	return fmt.Sprintf("Tag{Name:%s, Object:%s, Commit:%s, Annotated:%t}",
		t.Name.String(),
		t.Object.String(),
		t.Commit.String(),
		t.Annotated)
}

// Redacted returns a safe string representation suitable for logging in
// production environments. This method satisfies the model.Loggable
// interface's Redacted requirement.
//
// For Tag, all components except the Message are safe to log in production
// as they are public Git metadata. The Message field might contain sensitive
// information in some contexts, so it is omitted from the redacted output.
//
// The implementation delegates to the Redacted() methods of the component
// types (TagName, Hash), which may apply their own formatting for brevity.
// For example, Hash.Redacted() abbreviates hashes to 7 characters.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently.
//
// Example:
//
//	tag := git.Tag{Name: "v1.2.3", Object: hash, Commit: hash, Annotated: false}
//	log.Info("processing tag", "tag", tag.Redacted())
//	// Output (in logs): Tag{Name:v1.2.3, Object:a1b2c3d, Commit:a1b2c3d, Annotated:false}
func (t Tag) Redacted() string {
	return fmt.Sprintf("Tag{Name:%s, Object:%s, Commit:%s, Annotated:%t}",
		t.Name.Redacted(),
		t.Object.Redacted(),
		t.Commit.Redacted(),
		t.Annotated)
}

// TypeName returns the canonical name of this model type for debugging,
// logging, and reflection purposes. This method satisfies the model.Identifiable
// interface's TypeName requirement.
//
// The returned value is always the constant string "Tag", uniquely identifying
// this type within the dxrel domain. The name follows CamelCase convention
// and omits the package prefix as required by the Model contract.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently. The returned string is a constant literal,
// ensuring zero allocations.
func (t Tag) TypeName() string {
	return "Tag"
}

// IsZero reports whether this Tag instance is in a zero or empty state,
// meaning no tag has been specified. For Tag, the zero value represents
// "no tag specified" or "tag not initialized".
//
// This method satisfies the model.ZeroCheckable interface's IsZero requirement.
// A Tag is considered zero if all fields are their zero values: empty Name,
// empty Object and Commit hashes, Annotated == false, and empty Message.
//
// Zero-value Tags are semantically invalid for most operations and will fail
// validation if Validate() is called. However, the zero value is useful as
// a sentinel for "no tag specified" in optional fields or when initializing
// data structures.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently.
//
// Example:
//
//	var tag git.Tag // Zero value
//	fmt.Println(tag.IsZero()) // Output: true
//
//	tag = git.Tag{Name: "v1.2.3", Object: hash, Commit: hash}
//	fmt.Println(tag.IsZero()) // Output: false
func (t Tag) IsZero() bool {
	return t.Name.IsZero() &&
		t.Object.IsZero() &&
		t.Commit.IsZero() &&
		!t.Annotated &&
		t.Message == ""
}

// Equal reports whether this Tag is equal to another Tag value.
//
// Two Tags are equal if and only if all components match:
//   - Name must be equal (case-sensitive string comparison)
//   - Object must be equal (case-sensitive string comparison)
//   - Commit must be equal (case-sensitive string comparison)
//   - Annotated must be equal (boolean comparison)
//   - Message must be equal (case-sensitive string comparison)
//
// This method is particularly useful in table-driven tests, assertion libraries,
// deduplication logic, and comparison operations where a method-based approach
// is more idiomatic than manual field-by-field comparison.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently. The comparison is fast, deterministic,
// and performs no allocations beyond the standard string and hash comparisons.
//
// Example:
//
//	tag1 := git.Tag{Name: "v1.2.3", Object: hash1, Commit: hash1}
//	tag2 := git.Tag{Name: "v1.2.3", Object: hash1, Commit: hash1}
//	tag3 := git.Tag{Name: "v1.2.4", Object: hash2, Commit: hash2}
//	fmt.Println(tag1.Equal(tag2)) // Output: true
//	fmt.Println(tag1.Equal(tag3)) // Output: false (different Name and hashes)
func (t Tag) Equal(other Tag) bool {
	return t.Name.Equal(other.Name) &&
		t.Object.Equal(other.Object) &&
		t.Commit.Equal(other.Commit) &&
		t.Annotated == other.Annotated &&
		t.Message == other.Message
}

// Validate checks whether this Tag satisfies all model contracts and
// invariants. This method implements the model.Validatable interface's
// Validate requirement, enforcing data integrity for Git tag records.
//
// Validate returns nil if the Tag conforms to all of the following requirements:
//
// Name validation:
//   - Name MUST NOT be zero/empty (delegates to Name.Validate())
//
// Object validation:
//   - Object MUST NOT be zero/empty (delegates to Object.Validate())
//   - Object MUST be a valid Git hash (40 or 64 hex characters)
//
// Commit validation:
//   - Commit MUST NOT be zero/empty (delegates to Commit.Validate())
//   - Commit MUST be a valid Git hash (40 or 64 hex characters)
//
// Annotated/Message consistency:
//   - If Annotated == false (lightweight tag): Message MUST be empty
//   - If Annotated == true (annotated tag): Message MAY be empty or non-empty
//   - If Message is non-empty: length MUST NOT exceed TagMessageMaxLen (64KB)
//
// Object/Commit consistency:
//   - If Annotated == false (lightweight tag): Object SHOULD equal Commit
//     (this is not enforced as a hard error for flexibility)
//   - If Annotated == true (annotated tag): Object and Commit MAY differ
//
// This method MUST be fast, deterministic, and idempotent. It MUST NOT mutate
// the receiver, MUST NOT have side effects, and MUST be safe to call concurrently.
// Validation does not perform I/O or allocate memory except when constructing
// error messages for invalid values.
//
// Callers SHOULD invoke Validate after creating Tag instances from external
// sources (JSON, YAML, Git commands, user input) to ensure data integrity.
// The marshal/unmarshal methods automatically call Validate to enforce this
// contract.
//
// Example:
//
//	tag := git.Tag{Name: "v1.2.3", Object: hash, Commit: hash, Annotated: false}
//	if err := tag.Validate(); err != nil {
//	    log.Error("invalid tag", "error", err)
//	}
//
//	// Invalid: lightweight tag with message
//	tag = git.Tag{Name: "v1.2.3", Object: hash, Commit: hash, Annotated: false, Message: "oops"}
//	err := tag.Validate()
//	// err: "Tag Message must be empty for lightweight tags"
func (t Tag) Validate() error {
	// Validate Name
	if t.Name.IsZero() {
		return fmt.Errorf("%s Name must not be empty", t.TypeName())
	}
	if err := t.Name.Validate(); err != nil {
		return fmt.Errorf("invalid %s Name: %w", t.TypeName(), err)
	}

	// Validate Object
	if t.Object.IsZero() {
		return fmt.Errorf("%s Object must not be empty", t.TypeName())
	}
	if err := t.Object.Validate(); err != nil {
		return fmt.Errorf("invalid %s Object: %w", t.TypeName(), err)
	}

	// Validate Commit
	if t.Commit.IsZero() {
		return fmt.Errorf("%s Commit must not be empty", t.TypeName())
	}
	if err := t.Commit.Validate(); err != nil {
		return fmt.Errorf("invalid %s Commit: %w", t.TypeName(), err)
	}

	// Validate Message consistency with Annotated flag
	if !t.Annotated && t.Message != "" {
		return fmt.Errorf("%s Message must be empty for lightweight tags (got %d bytes)",
			t.TypeName(), len(t.Message))
	}

	// Validate Message length
	if len(t.Message) > TagMessageMaxLen {
		return fmt.Errorf("%s Message exceeds maximum length of %d bytes (got %d)",
			t.TypeName(), TagMessageMaxLen, len(t.Message))
	}

	return nil
}

// MarshalJSON implements json.Marshaler, serializing the Tag to JSON object
// format. This method satisfies part of the model.Serializable interface
// requirement.
//
// MarshalJSON first validates that the Tag conforms to all constraints by
// calling Validate. If validation fails, marshaling fails with the validation
// error, preventing invalid data from being serialized. If validation succeeds,
// the Tag is serialized to a JSON object with fields:
//
//	{
//	  "name": "v1.2.3",
//	  "object": "a1b2c3d4e5f67890abcdef1234567890abcdef12",
//	  "commit": "a1b2c3d4e5f67890abcdef1234567890abcdef12",
//	  "annotated": false,
//	  "message": ""  // omitted if empty via omitempty tag
//	}
//
// The "message" field is omitted when empty (via `omitempty` JSON tag),
// reducing JSON payload size for lightweight tags and annotated tags without
// messages.
//
// A type alias is used internally to avoid infinite recursion during the
// standard library json.Marshal call.
//
// This method MUST NOT mutate the receiver except as required by the
// json.Marshaler interface contract. It MUST be safe to call concurrently
// on immutable receivers.
//
// Example:
//
//	tag := git.Tag{Name: "v1.2.3", Object: hash, Commit: hash, Annotated: false}
//	data, _ := json.Marshal(tag)
//	fmt.Println(string(data))
//	// Output: {"name":"v1.2.3","object":"a1b2c3d...","commit":"a1b2c3d...","annotated":false}
func (t Tag) MarshalJSON() ([]byte, error) {
	if err := t.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", t.TypeName(), err)
	}

	// Use type alias to avoid infinite recursion
	type tag Tag
	return json.Marshal(tag(t))
}

// UnmarshalJSON implements json.Unmarshaler, deserializing a JSON object
// into a Tag value. This method satisfies part of the model.Serializable
// interface requirement.
//
// UnmarshalJSON accepts JSON objects with the structure defined in MarshalJSON.
// After unmarshaling the JSON data, Validate is called to ensure the resulting
// Tag conforms to all constraints. If validation fails (for example, invalid
// name, invalid hash, or Message/Annotated inconsistency), unmarshaling fails
// with an error describing the validation failure. This fail-fast behavior
// prevents invalid data from entering the system through external inputs.
//
// The method mutates the receiver to store the unmarshaled value. It is not
// safe for concurrent use during unmarshaling, though the resulting Tag value
// is safe for concurrent reads after unmarshaling completes.
//
// Example:
//
//	var tag git.Tag
//	json.Unmarshal([]byte(`{"name":"v1.2.3","object":"...","commit":"...","annotated":false}`), &tag)
//	fmt.Println(tag.Name)
//	// Output: v1.2.3
func (t *Tag) UnmarshalJSON(data []byte) error {
	type tag Tag
	if err := json.Unmarshal(data, (*tag)(t)); err != nil {
		return fmt.Errorf("cannot unmarshal JSON into %s: %w", t.TypeName(), err)
	}

	if err := t.Validate(); err != nil {
		return fmt.Errorf("unmarshaled %s is invalid: %w", t.TypeName(), err)
	}

	return nil
}

// MarshalYAML implements yaml.Marshaler, serializing the Tag to YAML object
// format. This method satisfies part of the model.Serializable interface
// requirement.
//
// MarshalYAML first validates that the Tag conforms to all constraints by
// calling Validate. If validation fails, marshaling fails with the validation
// error, preventing invalid data from being serialized. If validation succeeds,
// the Tag is serialized to a YAML object:
//
//	name: v1.2.3
//	object: a1b2c3d4e5f67890abcdef1234567890abcdef12
//	commit: a1b2c3d4e5f67890abcdef1234567890abcdef12
//	annotated: false
//	message: ""  # omitted if empty via omitempty tag
//
// The "message" field is omitted when empty (via `omitempty` YAML tag),
// improving YAML readability for lightweight tags.
//
// A type alias is used internally to avoid infinite recursion during marshaling.
//
// This method MUST NOT mutate the receiver except as required by the
// yaml.Marshaler interface contract. It MUST be safe to call concurrently
// on immutable receivers.
//
// Example:
//
//	tag := git.Tag{Name: "v1.2.3", Object: hash, Commit: hash, Annotated: false}
//	data, _ := yaml.Marshal(tag)
//	fmt.Println(string(data))
//	// Output:
//	// name: v1.2.3
//	// object: a1b2c3d4e5f67890abcdef1234567890abcdef12
//	// commit: a1b2c3d4e5f67890abcdef1234567890abcdef12
//	// annotated: false
func (t Tag) MarshalYAML() (interface{}, error) {
	if err := t.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", t.TypeName(), err)
	}

	// Use type alias to avoid infinite recursion
	type tag Tag
	return tag(t), nil
}

// UnmarshalYAML implements yaml.Unmarshaler, deserializing a YAML object
// into a Tag value. This method satisfies part of the model.Serializable
// interface requirement.
//
// UnmarshalYAML accepts YAML objects with the structure defined in MarshalYAML.
// After unmarshaling the YAML data, Validate is called to ensure the resulting
// Tag conforms to all constraints. If validation fails, unmarshaling fails
// with an error describing the validation failure. This fail-fast behavior
// prevents invalid configuration data from corrupting system state.
//
// The method mutates the receiver to store the unmarshaled value. It is not
// safe for concurrent use during unmarshaling, though the resulting Tag value
// is safe for concurrent reads after unmarshaling completes.
//
// Example:
//
//	var tag git.Tag
//	yaml.Unmarshal([]byte("name: v1.2.3\nobject: ...\ncommit: ...\nannotated: false"), &tag)
//	fmt.Println(tag.Name)
//	// Output: v1.2.3
func (t *Tag) UnmarshalYAML(node *yaml.Node) error {
	type tag Tag
	if err := node.Decode((*tag)(t)); err != nil {
		return fmt.Errorf("cannot unmarshal YAML into %s: %w", t.TypeName(), err)
	}

	if err := t.Validate(); err != nil {
		return fmt.Errorf("unmarshaled %s is invalid: %w", t.TypeName(), err)
	}

	return nil
}
