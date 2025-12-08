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
	"net/mail"
	"strings"
	"time"

	"dirpx.dev/dxrel/dxcore/errors"
	"dirpx.dev/dxrel/dxcore/model"
	"gopkg.in/yaml.v3"
)

const (
	// SignatureNameMaxLength is the maximum allowed length for a signature name
	// (author or committer name), measured in bytes.
	//
	// This limit prevents abuse and ensures names fit in typical display contexts
	// such as git log output, web interfaces, and terminal displays. A limit of
	// 256 bytes provides sufficient space for most real-world names, including
	// multi-byte UTF-8 characters for international names.
	//
	// While Git itself does not enforce strict name length limits, practical
	// considerations including terminal width, log readability, and tooling
	// compatibility make 256 bytes a sensible upper bound. Names exceeding this
	// limit MUST be rejected during validation.
	//
	// Note that this is a byte limit, not a Unicode code point (rune) limit.
	// Multi-byte UTF-8 characters in names count toward this limit based on
	// their encoded byte length, not the number of visible characters.
	SignatureNameMaxLength = 256

	// SignatureEmailMaxLength is the maximum allowed length for a signature email
	// address, measured in bytes.
	//
	// This limit is derived from RFC 5321, which specifies a maximum email address
	// length of 254 characters (not including angle brackets). This limit ensures
	// compatibility with standard email systems while preventing abuse through
	// excessively long email addresses.
	//
	// Git does not strictly validate email format or length, but dxrel enforces
	// both for data integrity. Email addresses exceeding 254 bytes MUST be rejected
	// during validation.
	//
	// The email format is validated using Go's standard library mail.ParseAddress,
	// which implements RFC 5322 parsing. This ensures that only well-formed email
	// addresses are accepted.
	SignatureEmailMaxLength = 254 // RFC 5321 maximum
)

// Signature represents a Git identity (author or committer) with associated
// timestamp information.
//
// This type implements the model.Model interface, providing validation,
// serialization to JSON and YAML, safe logging, type identification, and
// zero-value detection.
//
// A Signature combines three key pieces of information:
//   - Name: The human-readable name of the person
//   - Email: The email address associated with the identity
//   - When: The timestamp when the action occurred
//
// This representation is used for tracking Git commit authors and committers,
// preserving the identity information exactly as it appears in Git history.
//
// The zero value of Signature (all fields zero) is valid and represents
// "no signature specified", indicating that identity information has not
// been provided. Zero-value Signatures will fail validation if Validate()
// is called.
//
// Example values:
//
//	Signature{
//	    Name:  "Jane Doe",
//	    Email: "jane@example.com",
//	    When:  time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
//	}
//
//	Signature{
//	    Name:  "John Smith",
//	    Email: "john.smith@company.org",
//	    When:  time.Now(),
//	}
type Signature struct {
	// Name is the human-readable name as recorded in the commit.
	//
	// This is the full name of the person who created the commit or
	// authored the change. Git does not enforce any particular format,
	// but conventional practice is "First Last" or "First Middle Last".
	//
	// Name MUST be non-empty for a valid Signature and SHOULD be limited
	// to 256 characters. The name is used in git log output, commit
	// displays, and attribution.
	//
	// Examples:
	//   - "Jane Doe"
	//   - "John Smith"
	//   - "李明" (Unicode names are supported)
	Name string `json:"name" yaml:"name"`

	// Email is the email address as recorded in the commit.
	//
	// This is the contact email for the person. Git uses this for
	// identity tracking and does not validate the email format strictly,
	// but dxrel enforces basic validation (localpart@domain.tld).
	//
	// Email MUST be non-empty for a valid Signature and MUST follow
	// basic email format rules. Maximum length is 254 characters per
	// RFC 5321.
	//
	// Examples:
	//   - "jane@example.com"
	//   - "john.smith@company.org"
	//   - "developer+git@domain.co.uk"
	Email string `json:"email" yaml:"email"`

	// When is the timestamp associated with the signature.
	//
	// For commit authors, this is the author date (when the change was
	// originally created). For committers, this is the committer date
	// (when the commit was applied to the repository).
	//
	// When MUST be non-zero for a valid Signature. The timestamp is
	// serialized in RFC3339 format for JSON/YAML to ensure portability
	// and precision.
	//
	// Git stores timestamps with timezone information, and dxrel
	// preserves this via time.Time's Location field.
	When time.Time `json:"when" yaml:"when"`
}

// Compile-time check that Signature implements model.Model
var _ model.Model = (*Signature)(nil)

// NewSignature creates a new Signature with the given Name, Email, and When,
// validating the result before returning.
//
// This is a convenience constructor that creates and validates a Signature in
// one step, ensuring that all components conform to the validation rules defined
// by Signature.Validate. If any of the components are invalid (empty name, empty
// email, invalid email format, exceeds length limits, or zero timestamp), NewSignature
// returns a zero Signature and an error describing the validation failure.
//
// This function is particularly useful when constructing Signatures from user
// input, configuration files, or Git command output, as it guarantees that the
// resulting Signature is valid and ready for use.
//
// This function is pure and has no side effects. It is safe to call concurrently
// from multiple goroutines.
//
// Example usage:
//
//	sig, err := git.NewSignature("Jane Doe", "jane@example.com", time.Now())
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(sig.Name) // Output: Jane Doe
//
//	// Invalid email format
//	sig, err = git.NewSignature("John", "invalid-email", time.Now())
//	// err: "Signature Email has invalid format: ..."
func NewSignature(name string, email string, when time.Time) (Signature, error) {
	sig := Signature{
		Name:  name,
		Email: email,
		When:  when,
	}

	if err := sig.Validate(); err != nil {
		return Signature{}, err
	}

	return sig, nil
}

// String returns the human-readable representation of the Signature.
//
// This method implements the fmt.Stringer interface and satisfies the
// model.Loggable contract's String() requirement. The output includes
// all three components (Name, Email, When) for complete debugging
// visibility.
//
// Format: "Signature{Name:<name>, Email:<email>, When:<timestamp>}"
//
// Examples:
//
//	Signature{Name: "Jane Doe", Email: "jane@example.com", When: <time>}.String()
//	// Output: "Signature{Name:Jane Doe, Email:jane@example.com, When:2025-01-15T10:30:00Z}"
//
//	Signature{}.String()
//	// Output: "Signature{Name:, Email:, When:0001-01-01T00:00:00Z}"
func (s Signature) String() string {
	return fmt.Sprintf("Signature{Name:%s, Email:%s, When:%s}",
		s.Name,
		s.Email,
		s.When.Format(time.RFC3339))
}

// Redacted returns a safe representation of the Signature suitable for
// production logging.
//
// This method implements the model.Loggable contract's Redacted() requirement.
// The email address is partially redacted to protect privacy in production
// logs while maintaining enough information for debugging.
//
// Email redaction pattern:
//   - Shows first character + *** + @ + domain
//   - Example: "jane@example.com" -> "j***@example.com"
//   - Empty or invalid emails shown as "[invalid]"
//
// Name and When are not redacted as they are not considered sensitive.
//
// Format: "Signature{Name:<name>, Email:<redacted-email>, When:<timestamp>}"
//
// Examples:
//
//	Signature{Name: "Jane Doe", Email: "jane@example.com", When: <time>}.Redacted()
//	// Output: "Signature{Name:Jane Doe, Email:j***@example.com, When:2025-01-15T10:30:00Z}"
//
//	Signature{Name: "John", Email: "a@b.c", When: <time>}.Redacted()
//	// Output: "Signature{Name:John, Email:a***@b.c, When:2025-01-15T10:30:00Z}"
func (s Signature) Redacted() string {
	redactedEmail := redactEmail(s.Email)
	return fmt.Sprintf("Signature{Name:%s, Email:%s, When:%s}",
		s.Name,
		redactedEmail,
		s.When.Format(time.RFC3339))
}

// redactEmail redacts an email address for safe logging in production environments,
// applying a privacy-preserving transformation that obscures the local part while
// preserving the domain for debugging purposes.
//
// Redaction pattern: first_char + "***" + "@" + domain
//
// This function implements a consistent redaction strategy:
//   - If email is empty, returns "[empty]"
//   - If email has no "@" or "@" is at position 0, returns "[invalid]"
//   - Otherwise, returns: first character of local part + "***" + "@" + domain
//
// Examples:
//   - "jane@example.com" -> "j***@example.com"
//   - "a@b.c" -> "a***@b.c"
//   - "" -> "[empty]"
//   - "invalid" -> "[invalid]"
//   - "@example.com" -> "[invalid]"
//
// This redaction strikes a balance between privacy and debuggability. The domain
// is preserved to help identify organizational affiliations or email providers
// without exposing the full user identity.
//
// This function is pure, has no side effects, and is safe for concurrent use.
func redactEmail(email string) string {
	if email == "" {
		return "[empty]"
	}

	atIndex := strings.Index(email, "@")
	if atIndex <= 0 {
		return "[invalid]"
	}

	localPart := email[:atIndex]
	domain := email[atIndex:]

	if len(localPart) == 0 {
		return "[invalid]"
	}

	// Show first character + *** + @domain
	return string(localPart[0]) + "***" + domain
}

// TypeName returns the name of this type for error messages and debugging.
//
// This method implements the model.Identifiable contract.
func (s Signature) TypeName() string {
	return "Signature"
}

// IsZero reports whether this Signature is the zero value.
//
// This method implements the model.ZeroCheckable contract. A Signature is
// considered zero if all three components (Name, Email, When) are zero.
//
// Zero Signatures are semantically invalid for most operations and will fail
// validation if Validate() is called. However, the zero value is useful
// as a sentinel for "no signature specified" in optional fields or when
// initializing data structures.
//
// Examples:
//
//	Signature{}.IsZero()  // true
//	Signature{Name: "Jane"}.IsZero()  // false (has Name)
//	Signature{When: time.Now()}.IsZero()  // false (has When)
func (s Signature) IsZero() bool {
	return s.Name == "" && s.Email == "" && s.When.IsZero()
}

// Equal reports whether this Signature is equal to another Signature.
//
// Two Signatures are equal if all three components match:
//   - Name must be equal (case-sensitive string comparison)
//   - Email must be equal (case-sensitive string comparison)
//   - When must be equal (time.Time.Equal comparison, accounts for timezone)
//
// This method is used for testing, assertions, and deduplication logic.
//
// Examples:
//
//	s1 := Signature{Name: "Jane", Email: "jane@example.com", When: t1}
//	s2 := Signature{Name: "Jane", Email: "jane@example.com", When: t1}
//	s1.Equal(s2)  // true
//
//	s3 := Signature{Name: "Jane", Email: "jane@example.com", When: t2}
//	s1.Equal(s3)  // false (different When)
func (s Signature) Equal(other Signature) bool {
	return s.Name == other.Name &&
		s.Email == other.Email &&
		s.When.Equal(other.When)
}

// Validate checks whether this Signature satisfies all model contracts and
// invariants. This method implements the model.Validatable interface's Validate
// requirement, enforcing data integrity for Git identity information.
//
// Validate returns nil if the Signature conforms to all of the following
// requirements:
//
// Name validation:
//   - Name MUST NOT be empty (empty names are invalid for commit identities)
//   - Name length MUST NOT exceed SignatureNameMaxLength (256 bytes)
//
// Email validation:
//   - Email MUST NOT be empty (every commit identity requires an email)
//   - Email length MUST NOT exceed SignatureEmailMaxLength (254 bytes per RFC 5321)
//   - Email MUST conform to RFC 5322 format (validated using mail.ParseAddress)
//
// When validation:
//   - When MUST NOT be zero (time.IsZero() == false)
//   - Timestamps are required for all commit identities
//
// Email validation uses Go's standard library mail.ParseAddress, which implements
// RFC 5322 parsing. This catches most common errors (missing @, invalid characters,
// malformed addresses) while accepting the wide variety of email formats used in
// Git history, including addresses without TLDs for local networks.
//
// Zero-value Signatures (all fields zero) will fail validation, as commit identities
// MUST have meaningful values for all three components.
//
// This method MUST be fast, deterministic, and idempotent. It MUST NOT mutate
// the receiver, MUST NOT have side effects, and MUST be safe to call concurrently.
// Validation does not perform I/O or allocate memory except when constructing
// error messages for invalid values.
//
// Callers SHOULD invoke Validate after creating Signature instances from external
// sources (JSON, YAML, Git commands, user input) to ensure data integrity. The
// marshal/unmarshal methods automatically call Validate to enforce this contract.
//
// Example:
//
//	sig := git.Signature{Name: "Jane", Email: "jane@example.com", When: time.Now()}
//	if err := sig.Validate(); err != nil {
//	    log.Error("invalid signature", "error", err)
//	}
//
//	// Invalid: zero value
//	sig = git.Signature{}
//	err := sig.Validate()
//	// err: "Signature Name must not be empty"
func (s Signature) Validate() error {
	// Validate Name
	if s.Name == "" {
		return &errors.ValidationError{
			Type:   s.TypeName(),
			Field:  "Name",
			Reason: "must not be empty",
		}
	}
	if len(s.Name) > SignatureNameMaxLength {
		return &errors.ValidationError{
			Type:   s.TypeName(),
			Field:  "Name",
			Reason: fmt.Sprintf("exceeds maximum length of %d characters (got %d)", SignatureNameMaxLength, len(s.Name)),
		}
	}

	// Validate Email
	if s.Email == "" {
		return &errors.ValidationError{
			Type:   s.TypeName(),
			Field:  "Email",
			Reason: "must not be empty",
		}
	}
	if len(s.Email) > SignatureEmailMaxLength {
		return &errors.ValidationError{
			Type:   s.TypeName(),
			Field:  "Email",
			Reason: fmt.Sprintf("exceeds maximum length of %d characters (got %d)", SignatureEmailMaxLength, len(s.Email)),
		}
	}
	// Use standard library to validate email format per RFC 5322
	if _, err := mail.ParseAddress(s.Email); err != nil {
		return &errors.ValidationError{
			Type:   s.TypeName(),
			Field:  "Email",
			Reason: fmt.Sprintf("has invalid format: %q (%v)", s.Email, err),
		}
	}

	// Validate When
	if s.When.IsZero() {
		return &errors.ValidationError{
			Type:   s.TypeName(),
			Field:  "When",
			Reason: "must not be zero",
		}
	}

	return nil
}

// MarshalJSON implements json.Marshaler, serializing the Signature to JSON
// object format. This method satisfies part of the model.Serializable interface
// requirement.
//
// MarshalJSON first validates that the Signature conforms to all constraints
// by calling Validate. If validation fails, marshaling fails with the validation
// error, preventing invalid data from being serialized. If validation succeeds,
// the Signature is serialized to a JSON object with three fields:
//
//	{
//	  "name": "Jane Doe",
//	  "email": "jane@example.com",
//	  "when": "2025-01-15T10:30:00Z"
//	}
//
// The When field is serialized in RFC3339 format (time.RFC3339) for portability,
// precision, and timezone preservation. All three fields are always present in
// the output, as all are required for a valid Signature.
//
// A type alias is used internally to avoid infinite recursion during the
// standard library json.Marshal call.
//
// This method MUST NOT mutate the receiver except as required by the json.Marshaler
// interface contract. It MUST be safe to call concurrently on immutable receivers.
//
// Example:
//
//	sig := git.Signature{Name: "Jane", Email: "jane@example.com", When: time.Now()}
//	data, _ := json.Marshal(sig)
//	fmt.Println(string(data))
//	// Output: {"name":"Jane","email":"jane@example.com","when":"2025-01-15T10:30:00Z"}
func (s Signature) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", s.TypeName(), err)
	}

	// Use type alias to avoid infinite recursion
	type signature Signature
	return json.Marshal(signature(s))
}

// UnmarshalJSON implements json.Unmarshaler, deserializing a JSON object into
// a Signature value. This method satisfies part of the model.Serializable
// interface requirement.
//
// UnmarshalJSON accepts JSON objects with the following structure:
//
//	{
//	  "name": "Jane Doe",
//	  "email": "jane@example.com",
//	  "when": "2025-01-15T10:30:00Z"
//	}
//
// All three fields are required. The "when" field MUST be in RFC3339 format
// (or any format accepted by Go's time.Time JSON unmarshaling).
//
// After unmarshaling the JSON data, Validate is called to ensure the resulting
// Signature conforms to all constraints. If validation fails (for example, invalid
// name, invalid email format, or zero timestamp), unmarshaling fails with an error
// describing the validation failure. This fail-fast behavior prevents invalid
// data from entering the system through external inputs.
//
// The method mutates the receiver to store the unmarshaled value. It is not
// safe for concurrent use during unmarshaling, though the resulting Signature
// value is safe for concurrent reads after unmarshaling completes.
//
// Example:
//
//	var sig git.Signature
//	json.Unmarshal([]byte(`{"name":"Jane","email":"jane@example.com","when":"2025-01-15T10:30:00Z"}`), &sig)
//	fmt.Println(sig.Name)
//	// Output: Jane
func (s *Signature) UnmarshalJSON(data []byte) error {
	type signature Signature
	if err := json.Unmarshal(data, (*signature)(s)); err != nil {
		return &errors.UnmarshalError{
			Type:   s.TypeName(),
			Data:   data,
			Reason: err.Error(),
		}
	}

	if err := s.Validate(); err != nil {
		return &errors.UnmarshalError{
			Type:   s.TypeName(),
			Data:   data,
			Reason: fmt.Sprintf("validation failed: %v", err),
		}
	}

	return nil
}

// MarshalYAML implements yaml.Marshaler, serializing the Signature to YAML
// object format. This method satisfies part of the model.Serializable interface
// requirement.
//
// MarshalYAML first validates that the Signature conforms to all constraints
// by calling Validate. If validation fails, marshaling fails with the validation
// error, preventing invalid data from being serialized. If validation succeeds,
// the Signature is serialized to a YAML object:
//
//	name: Jane Doe
//	email: jane@example.com
//	when: 2025-01-15T10:30:00Z
//
// The When field is serialized in RFC3339 format for portability and timezone
// preservation. All three fields are always present in the output.
//
// A type alias is used internally to avoid infinite recursion during marshaling.
//
// This method MUST NOT mutate the receiver except as required by the yaml.Marshaler
// interface contract. It MUST be safe to call concurrently on immutable receivers.
//
// Example:
//
//	sig := git.Signature{Name: "John", Email: "john@example.com", When: time.Now()}
//	data, _ := yaml.Marshal(sig)
//	fmt.Println(string(data))
//	// Output:
//	// name: John
//	// email: john@example.com
//	// when: 2025-01-15T10:30:00Z
func (s Signature) MarshalYAML() (interface{}, error) {
	if err := s.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", s.TypeName(), err)
	}

	// Use type alias to avoid infinite recursion
	type signature Signature
	return signature(s), nil
}

// UnmarshalYAML implements yaml.Unmarshaler, deserializing a YAML object into
// a Signature value. This method satisfies part of the model.Serializable
// interface requirement.
//
// UnmarshalYAML accepts YAML objects with the following structure:
//
//	name: Jane Doe
//	email: jane@example.com
//	when: 2025-01-15T10:30:00Z
//
// All three fields are required. The "when" field MUST be in RFC3339 format
// (or any format accepted by Go's time.Time YAML unmarshaling).
//
// After unmarshaling the YAML data, Validate is called to ensure the resulting
// Signature conforms to all constraints. If validation fails, unmarshaling fails
// with an error describing the validation failure. This fail-fast behavior prevents
// invalid configuration data from corrupting system state.
//
// The method mutates the receiver to store the unmarshaled value. It is not
// safe for concurrent use during unmarshaling, though the resulting Signature
// value is safe for concurrent reads after unmarshaling completes.
//
// Example:
//
//	var sig git.Signature
//	yaml.Unmarshal([]byte("name: Jane\nemail: jane@example.com\nwhen: 2025-01-15T10:30:00Z"), &sig)
//	fmt.Println(sig.Name)
//	// Output: Jane
func (s *Signature) UnmarshalYAML(node *yaml.Node) error {
	type signature Signature
	if err := node.Decode((*signature)(s)); err != nil {
		return &errors.UnmarshalError{
			Type:   s.TypeName(),
			Data:   []byte(fmt.Sprintf("%v", node)),
			Reason: err.Error(),
		}
	}

	if err := s.Validate(); err != nil {
		return &errors.UnmarshalError{
			Type:   s.TypeName(),
			Data:   []byte(fmt.Sprintf("%v", node)),
			Reason: fmt.Sprintf("validation failed: %v", err),
		}
	}

	return nil
}
