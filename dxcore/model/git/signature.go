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

	"dirpx.dev/dxrel/dxcore/model"
	"gopkg.in/yaml.v3"
)

const (
	// SignatureNameMaxLength is the maximum length for a signature name.
	// This limit prevents abuse and ensures names fit in typical display contexts.
	SignatureNameMaxLength = 256

	// SignatureEmailMaxLength is the maximum length for a signature email.
	// This limit accommodates most valid email addresses while preventing abuse.
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

// NewSignature creates a new Signature with the given Name, Email, and When.
//
// This is a convenience constructor that creates and validates a Signature
// in one step. If any of the components are invalid, NewSignature returns
// a zero Signature and an error.
//
// Example usage:
//
//	sig, err := git.NewSignature("Jane Doe", "jane@example.com", time.Now())
//	if err != nil {
//	    // handle error
//	}
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

// redactEmail redacts an email address for safe logging.
// Pattern: first char + *** + @ + domain
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
// invariants.
//
// This method implements the model.Validatable contract. Validation ensures:
//   - Name is non-empty and within length limits (1-256 characters)
//   - Email is non-empty, follows basic format rules, and within length limits (1-254 characters)
//   - When is non-zero (not the zero time)
//
// Email validation uses Go's standard library mail.ParseAddress which
// validates according to RFC 5322. This catches most common errors while
// accepting the wide variety of email formats used in Git history.
//
// Zero-value Signatures (all fields zero) will fail validation.
//
// Returns nil if the Signature is valid, or a descriptive error if validation fails.
//
// Examples:
//
//	Signature{Name: "Jane", Email: "jane@example.com", When: time.Now()}.Validate()
//	// Returns: nil (valid)
//
//	Signature{}.Validate()
//	// Returns: error "Signature Name must not be empty"
//
//	Signature{Name: "Jane", Email: "invalid", When: time.Now()}.Validate()
//	// Returns: error about invalid email format
func (s Signature) Validate() error {
	// Validate Name
	if s.Name == "" {
		return fmt.Errorf("%s Name must not be empty", s.TypeName())
	}
	if len(s.Name) > SignatureNameMaxLength {
		return fmt.Errorf("%s Name exceeds maximum length of %d characters (got %d)",
			s.TypeName(), SignatureNameMaxLength, len(s.Name))
	}

	// Validate Email
	if s.Email == "" {
		return fmt.Errorf("%s Email must not be empty", s.TypeName())
	}
	if len(s.Email) > SignatureEmailMaxLength {
		return fmt.Errorf("%s Email exceeds maximum length of %d characters (got %d)",
			s.TypeName(), SignatureEmailMaxLength, len(s.Email))
	}
	// Use standard library to validate email format per RFC 5322
	if _, err := mail.ParseAddress(s.Email); err != nil {
		return fmt.Errorf("%s Email has invalid format: %q (%w)", s.TypeName(), s.Email, err)
	}

	// Validate When
	if s.When.IsZero() {
		return fmt.Errorf("%s When must not be zero", s.TypeName())
	}

	return nil
}

// MarshalJSON serializes this Signature to JSON.
//
// This method implements the json.Marshaler interface and the
// model.Serializable contract.
//
// The JSON format is an object with three fields:
//
//	{
//	  "name": "Jane Doe",
//	  "email": "jane@example.com",
//	  "when": "2025-01-15T10:30:00Z"
//	}
//
// The When field is serialized in RFC3339 format for portability and
// precision. All three fields are always present in the output.
//
// Before encoding, MarshalJSON calls Validate. If the Signature is invalid,
// the validation error is returned and no JSON is produced.
func (s Signature) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", s.TypeName(), err)
	}

	// Use type alias to avoid infinite recursion
	type signature Signature
	return json.Marshal(signature(s))
}

// UnmarshalJSON deserializes a Signature from JSON.
//
// This method implements the json.Unmarshaler interface and the
// model.Serializable contract.
//
// The expected JSON format is an object with three fields:
//
//	{
//	  "name": "Jane Doe",
//	  "email": "jane@example.com",
//	  "when": "2025-01-15T10:30:00Z"
//	}
//
// After unmarshaling, Validate is called to ensure the deserialized Signature
// satisfies all invariants. If validation fails, the error is returned
// and the Signature MUST NOT be used.
func (s *Signature) UnmarshalJSON(data []byte) error {
	type signature Signature
	if err := json.Unmarshal(data, (*signature)(s)); err != nil {
		return fmt.Errorf("cannot unmarshal JSON into %s: %w", s.TypeName(), err)
	}

	if err := s.Validate(); err != nil {
		return fmt.Errorf("unmarshaled %s is invalid: %w", s.TypeName(), err)
	}

	return nil
}

// MarshalYAML serializes this Signature to YAML.
//
// This method implements the yaml.Marshaler interface and the
// model.Serializable contract.
//
// The YAML format is an object with three fields:
//
//	name: Jane Doe
//	email: jane@example.com
//	when: 2025-01-15T10:30:00Z
//
// The When field is serialized in RFC3339 format. All three fields are
// always present in the output.
//
// Before encoding, MarshalYAML calls Validate. If the Signature is invalid,
// the validation error is returned and no YAML is produced.
func (s Signature) MarshalYAML() (interface{}, error) {
	if err := s.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", s.TypeName(), err)
	}

	// Use type alias to avoid infinite recursion
	type signature Signature
	return signature(s), nil
}

// UnmarshalYAML deserializes a Signature from YAML.
//
// This method implements the yaml.Unmarshaler interface and the
// model.Serializable contract.
//
// The expected YAML format is an object with three fields:
//
//	name: Jane Doe
//	email: jane@example.com
//	when: 2025-01-15T10:30:00Z
//
// After unmarshaling, Validate is called to ensure the deserialized Signature
// satisfies all invariants. If validation fails, the error is returned
// and the Signature MUST NOT be used.
func (s *Signature) UnmarshalYAML(node *yaml.Node) error {
	type signature Signature
	if err := node.Decode((*signature)(s)); err != nil {
		return fmt.Errorf("cannot unmarshal YAML into %s: %w", s.TypeName(), err)
	}

	if err := s.Validate(); err != nil {
		return fmt.Errorf("unmarshaled %s is invalid: %w", s.TypeName(), err)
	}

	return nil
}
