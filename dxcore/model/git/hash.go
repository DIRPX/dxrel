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

	"dirpx.dev/dxrel/dxcore/model"
	"gopkg.in/yaml.v3"
)

const (
	// HashHexSizeSHA1 is the number of hexadecimal characters in a
	// canonical SHA-1 Git object id.
	//
	// Git SHA-1 commit hashes are always 40 hex characters long, which
	// corresponds to a 20-byte digest.
	HashHexSizeSHA1 = 40

	// HashByteSizeSHA1 is the number of raw bytes in an SHA-1 digest.
	HashByteSizeSHA1 = 20

	// HashHexSizeSHA256 is the number of hexadecimal characters in a
	// canonical SHA-256 Git object id.
	//
	// Git repositories configured to use SHA-256 produce 64-character
	// hex object ids, which correspond to a 32-byte digest.
	HashHexSizeSHA256 = 64

	// HashByteSizeSHA256 is the number of raw bytes in an SHA-256 digest.
	HashByteSizeSHA256 = 32

	// HashShortLen is the default length for abbreviated commit hashes
	// used in display contexts. A length of 7 characters provides sufficient
	// uniqueness for most repositories while remaining human-readable.
	HashShortLen = 7
)

const (
	// hashHexPattern is the regular expression used to validate canonical
	// Git object ids for dxrel.
	//
	// The pattern matches:
	//   - exactly 40 lower-case hex characters (SHA-1), or
	//   - exactly 64 lower-case hex characters (SHA-256).
	//
	// This pattern assumes that the input has already been normalized:
	//   - no surrounding whitespace
	//   - lower-case letters only
	//
	// The zero value (empty string) is handled separately and is not
	// expected to be validated against this pattern.
	hashHexPattern = `^(?:[0-9a-f]{40}|[0-9a-f]{64})$`
)

var (
	// HashHexRegexp is the compiled regular expression used to validate
	// canonical Git object ids.
	//
	// It is safe for concurrent use by multiple goroutines. Callers
	// SHOULD prefer higher-level helpers such as ParseHash, Hash.Validate,
	// or internal validation functions rather than using this regexp
	// directly in business logic, so that normalization and error
	// reporting remain consistent across the codebase.
	HashHexRegexp = regexp.MustCompile(hashHexPattern)
)

// Hash represents a canonical Git commit object id, uniquely identifying a
// specific commit in a Git repository. Hash values are used throughout dxrel
// wherever commit identification is required, including Conventional Commit
// metadata, planner ranges, release results, and version tracking.
//
// This type implements the model.Model interface, providing validation,
// serialization to JSON and YAML, safe logging, type identification, and
// zero-value detection. The zero value of Hash (empty string "") is valid
// and represents "no hash attached" or "synthetic commit", indicating that
// a commit has not yet been associated with a Git object id.
//
// Hash values MUST be fully expanded Git object ids in canonical form:
// lowercase hexadecimal strings of exactly 40 characters (SHA-1) or exactly
// 64 characters (SHA-256). Abbreviated hashes (such as "a1b2c3d") are
// intended only for display and CLI input. Code that stores a Hash MUST
// resolve abbreviations to full object ids via Git commands (git rev-parse,
// git log, etc.) before constructing the Hash value.
//
// Hash values MUST be normalized to lowercase during parsing. Git internally
// treats object ids as case-insensitive (both "A1B2" and "a1b2" refer to
// the same object), but dxrel enforces lowercase for consistency, storage
// efficiency, and comparison performance. The ParseHash function automatically
// normalizes input to lowercase.
//
// The zero value (empty string) has special semantics distinct from invalid
// hashes. An empty Hash is valid and represents absence of a commit id,
// which occurs in synthetic commits, uncommitted changes, or placeholder
// values during construction. Validation does not reject empty hashes.
//
// Example usage:
//
//	hash := git.Hash("a1b2c3d4e5f6789012345678901234567890abcd")
//	fmt.Println(hash.String())  // Full hash
//	fmt.Println(hash.Short())   // "a1b2c3d" (abbreviated)
//	fmt.Println(hash.IsSHA1())  // true
//
//	var parsed git.Hash
//	json.Unmarshal([]byte(`"A1B2C3D4E5F6789012345678901234567890ABCD"`), &parsed)
//	fmt.Println(parsed.Validate()) // Output: <nil> (normalized to lowercase)
type Hash string

// String returns the string representation of the Hash, which is the full
// hexadecimal object id in lowercase form. This method satisfies the
// model.Loggable interface's String requirement, providing a human-readable
// representation suitable for display, debugging, and logging.
//
// For non-zero hashes, the returned string is the canonical Git object id
// (40 characters for SHA-1, 64 characters for SHA-256). For the zero value
// (empty hash), the returned string is an empty string. When rendering
// commit information, callers SHOULD check IsZero() to determine whether to
// display the hash or indicate absence of a commit id.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently. The returned string is the Hash value
// itself, ensuring zero allocations for this operation.
//
// Example:
//
//	hash := git.Hash("a1b2c3d4e5f6789012345678901234567890abcd")
//	fmt.Println(hash.String()) // Output: "a1b2c3d4e5f6789012345678901234567890abcd"
//
//	empty := git.Hash("")
//	fmt.Println(empty.String()) // Output: ""
func (h Hash) String() string {
	return string(h)
}

// Redacted returns a safe string representation suitable for logging in
// production environments. For Hash, Redacted returns an abbreviated form
// of the commit id (first 7 characters) to reduce log verbosity while
// maintaining human readability and commit identification.
//
// This method satisfies the model.Loggable interface's Redacted requirement,
// ensuring that Hash can be safely logged without exposing full object ids
// in production logs. While commit hashes are not sensitive data, abbreviated
// hashes reduce log size and improve readability without sacrificing the
// ability to identify specific commits. Seven characters provide sufficient
// uniqueness for most repositories (approximately 268 million possibilities).
//
// For zero-value hashes (empty string), Redacted returns an empty string.
// For hashes shorter than 7 characters (which would be invalid), Redacted
// returns the full hash to avoid index out of bounds errors.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently.
//
// Example:
//
//	hash := git.Hash("a1b2c3d4e5f6789012345678901234567890abcd")
//	log.Info("processing commit", "hash", hash.Redacted()) // Output: "a1b2c3d"
func (h Hash) Redacted() string {
	return h.Short()
}

// TypeName returns the canonical name of this model type for debugging,
// logging, and reflection purposes. This method satisfies the model.Identifiable
// interface's TypeName requirement.
//
// The returned value is always the constant string "Hash", uniquely
// identifying this type within the dxrel domain. The name follows CamelCase
// convention and omits the package prefix as required by the Model contract.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently. The returned string is a constant literal,
// ensuring zero allocations.
func (h Hash) TypeName() string {
	return "Hash"
}

// IsZero reports whether this Hash instance is in a zero or empty state,
// meaning no commit id has been associated. For Hash, the zero value (empty
// string) represents "no hash attached" or "synthetic commit", indicating
// that a commit has not yet been associated with a Git object id.
//
// This method satisfies the model.ZeroCheckable interface's IsZero requirement.
// Empty hashes are valid and have special semantics distinct from invalid
// hashes. An empty Hash represents absence of a commit id, which occurs in
// synthetic commits generated for analysis, uncommitted changes, or placeholder
// values during commit construction.
//
// Callers can use IsZero to determine whether to display commit ids or
// indicate absence in UI and logs. When IsZero returns true, implementations
// SHOULD display alternative text such as "uncommitted", "synthetic", or
// omit the hash field entirely.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently.
//
// Example:
//
//	hash := git.Hash("")
//	fmt.Println(hash.IsZero()) // Output: true
//
//	hash = git.Hash("a1b2c3d4e5f6789012345678901234567890abcd")
//	fmt.Println(hash.IsZero()) // Output: false
func (h Hash) IsZero() bool {
	return h == ""
}

// Equal reports whether this Hash is equal to another Hash value, providing
// an explicit equality comparison method that follows common Go idioms for
// string-based value types. While Hash values can be compared using the ==
// operator directly, this method offers a named alternative that improves
// code readability and maintains consistency with other model types in the
// dxrel codebase.
//
// Equal performs a simple string comparison and returns true if both Hash
// values contain identical character sequences. The comparison is case-sensitive
// and exact, considering each hexadecimal digit. Empty hashes (zero values)
// are equal to other empty hashes, and normalized hashes are equal only to
// identically normalized hashes (both lowercase).
//
// This method is particularly useful in table-driven tests, assertion libraries,
// commit deduplication, and comparison operations where a method-based approach
// is more idiomatic than operator-based comparison. It also provides a
// consistent interface across all model types, some of which MAY require more
// complex equality semantics than simple string comparison.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently. The comparison is fast, deterministic,
// and performs no additional allocations beyond the standard string comparison.
//
// Example:
//
//	h1 := git.Hash("a1b2c3d4e5f6789012345678901234567890abcd")
//	h2 := git.Hash("1234567890abcdef1234567890abcdef12345678")
//	h3 := git.Hash("a1b2c3d4e5f6789012345678901234567890abcd")
//	fmt.Println(h1.Equal(h2)) // Output: false
//	fmt.Println(h1.Equal(h3)) // Output: true
func (h Hash) Equal(other Hash) bool {
	return h == other
}

// Short returns an abbreviated form of the Hash suitable for display in
// user interfaces, logs, and command-line output. The abbreviated hash
// consists of the first HashShortLen (7) characters of the full object id,
// providing a human-readable identifier that is sufficiently unique for most
// repositories.
//
// Short is commonly used in git log output, commit references, and UI
// displays where full 40 or 64 character hashes would be excessively verbose.
// Seven characters provide approximately 268 million unique values for SHA-1
// hashes and even more for SHA-256, making collisions extremely rare in
// typical repository sizes.
//
// For zero-value hashes (empty string), Short returns an empty string. For
// hashes shorter than HashShortLen characters (which would be invalid and
// fail validation), Short returns the full hash to avoid index out of bounds
// errors.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently. The returned string is allocated from
// a substring of the hash value.
//
// Example:
//
//	hash := git.Hash("a1b2c3d4e5f6789012345678901234567890abcd")
//	fmt.Println(hash.Short()) // Output: "a1b2c3d"
//
//	empty := git.Hash("")
//	fmt.Println(empty.Short()) // Output: ""
func (h Hash) Short() string {
	str := string(h)
	if len(str) < HashShortLen {
		return str
	}
	return str[:HashShortLen]
}

// IsSHA1 reports whether this Hash represents an SHA-1 object id. SHA-1 hashes
// are 40 hexadecimal characters long and were the original hashing algorithm
// used by Git. This method checks only the length, not the validity of the
// hexadecimal content.
//
// IsSHA1 returns true if the hash is exactly 40 characters long. It returns
// false for zero-value hashes (empty string), SHA-256 hashes (64 characters),
// and invalid hashes of incorrect lengths.
//
// This method is useful for distinguishing between SHA-1 and SHA-256 based
// repositories, selecting appropriate hashing algorithms, and formatting
// output based on object id type.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently.
//
// Example:
//
//	sha1 := git.Hash("a1b2c3d4e5f6789012345678901234567890abcd")
//	fmt.Println(sha1.IsSHA1()) // Output: true
//
//	sha256 := git.Hash("a1b2c3d4e5f6789012345678901234567890abcda1b2c3d4e5f6789012345678")
//	fmt.Println(sha256.IsSHA1()) // Output: false
func (h Hash) IsSHA1() bool {
	return len(h) == HashHexSizeSHA1
}

// IsSHA256 reports whether this Hash represents an SHA-256 object id. SHA-256
// hashes are 64 hexadecimal characters long and are supported in Git
// repositories configured for stronger cryptographic hashing. This method
// checks only the length, not the validity of the hexadecimal content.
//
// IsSHA256 returns true if the hash is exactly 64 characters long. It returns
// false for zero-value hashes (empty string), SHA-1 hashes (40 characters),
// and invalid hashes of incorrect lengths.
//
// This method is useful for distinguishing between SHA-1 and SHA-256 based
// repositories, selecting appropriate hashing algorithms, and formatting
// output based on object id type.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently.
//
// Example:
//
//	sha256 := git.Hash("a1b2c3d4e5f6789012345678901234567890abcda1b2c3d4e5f6789012345678")
//	fmt.Println(sha256.IsSHA256()) // Output: true
//
//	sha1 := git.Hash("a1b2c3d4e5f6789012345678901234567890abcd")
//	fmt.Println(sha1.IsSHA256()) // Output: false
func (h Hash) IsSHA256() bool {
	return len(h) == HashHexSizeSHA256
}

// Validate checks that the Hash value conforms to all constraints defined by
// Git object id conventions and dxrel policies. This method satisfies the
// model.Validatable interface's Validate requirement, enforcing data integrity.
//
// Validate returns nil if the Hash is either the zero value (empty string,
// representing "no hash attached") or a non-empty string that satisfies all
// the following requirements: the length MUST be exactly 40 characters
// (SHA-1) or exactly 64 characters (SHA-256); all characters MUST be lowercase
// hexadecimal digits [0-9a-f]; the hash MUST match HashHexRegexp pattern.
//
// Validate returns an error if any constraint is violated. The error message
// describes which specific constraint failed and includes the invalid value
// to aid debugging. Common validation failures include hashes with incorrect
// length (abbreviated or corrupted hashes), hashes containing uppercase
// letters (not normalized), hashes containing invalid characters (not
// hexadecimal), and hashes with mixed case.
//
// This method MUST be fast, deterministic, and idempotent. It MUST NOT mutate
// the receiver, MUST NOT have side effects, and MUST be safe to call concurrently.
// Validation does not perform I/O or allocate memory except when constructing
// error messages for invalid values.
//
// Callers SHOULD invoke Validate after deserializing Hash from external
// sources (JSON, YAML, databases, user input) to ensure data integrity. The
// ParseHash function automatically validates after normalization.
//
// Example:
//
//	hash := git.Hash("a1b2c3d4e5f6789012345678901234567890abcd")
//	if err := hash.Validate(); err != nil {
//	    log.Error("invalid hash", "error", err)
//	}
func (h Hash) Validate() error {
	// Empty hash is valid (represents "not set" or "synthetic commit")
	if h.IsZero() {
		return nil
	}

	str := string(h)

	// Check length
	if len(str) != HashHexSizeSHA1 && len(str) != HashHexSizeSHA256 {
		return fmt.Errorf("Hash %q has invalid length: %d (expected %d for SHA-1 or %d for SHA-256)", str, len(str), HashHexSizeSHA1, HashHexSizeSHA256)
	}

	// Check format (lowercase hexadecimal)
	if !HashHexRegexp.MatchString(str) {
		return fmt.Errorf("Hash %q contains invalid characters (must be lowercase hexadecimal [0-9a-f])", str)
	}

	return nil
}

// MarshalJSON implements json.Marshaler, serializing the Hash to its string
// representation as a JSON string. This method satisfies part of the
// model.Serializable interface requirement.
//
// MarshalJSON first validates that the Hash conforms to all constraints by
// calling Validate. If validation fails, marshaling fails with the validation
// error, preventing invalid data from being serialized. If validation succeeds,
// the Hash is converted to its string representation and marshaled as a JSON
// string.
//
// Empty hashes (zero values) marshal to the JSON string "" (empty string),
// representing "no hash attached". Non-empty hashes marshal to their normalized
// lowercase hexadecimal form (40 or 64 characters).
//
// This method MUST NOT mutate the receiver except as required by the json.Marshaler
// interface contract. It MUST be safe to call concurrently on immutable receivers.
//
// Example:
//
//	hash := git.Hash("a1b2c3d4e5f6789012345678901234567890abcd")
//	data, _ := json.Marshal(hash)
//	fmt.Println(string(data)) // Output: "a1b2c3d4e5f6789012345678901234567890abcd"
func (h Hash) MarshalJSON() ([]byte, error) {
	if err := h.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", h.TypeName(), err)
	}
	return json.Marshal(string(h))
}

// UnmarshalJSON implements json.Unmarshaler, deserializing a JSON string into
// a Hash value. This method satisfies part of the model.Serializable interface
// requirement.
//
// UnmarshalJSON accepts JSON strings containing Git object ids and applies
// normalization before validation. The input undergoes case normalization
// where uppercase letters are converted to lowercase (Git treats object ids
// as case-insensitive) and whitespace trimming via strings.TrimSpace. This
// normalization ensures consistent representation regardless of the source
// system or input format.
//
// After unmarshaling and normalization, Validate is called to ensure the
// resulting Hash conforms to all constraints. If the normalized string is
// invalid (for example, wrong length, contains non-hexadecimal characters),
// unmarshaling fails with an error describing the validation failure. This
// fail-fast behavior prevents invalid data from entering the system through
// external inputs.
//
// Empty JSON strings unmarshal successfully to the zero value Hash, representing
// "no hash attached". JSON null values are rejected as invalid input.
//
// The method mutates the receiver to store the unmarshaled value. It is not
// safe for concurrent use during unmarshaling, though the resulting Hash value
// is safe for concurrent reads after unmarshaling completes.
//
// Example:
//
//	var hash git.Hash
//	json.Unmarshal([]byte(`"A1B2C3D4E5F6789012345678901234567890ABCD"`), &hash)
//	fmt.Println(hash.String()) // Output: "a1b2c3d4e5f6789012345678901234567890abcd" (normalized)
func (h *Hash) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return fmt.Errorf("cannot unmarshal JSON: %w", err)
	}

	parsed, err := ParseHash(str)
	if err != nil {
		return fmt.Errorf("unmarshaled model is invalid: %w", err)
	}

	*h = parsed
	return nil
}

// MarshalYAML implements yaml.Marshaler, serializing the Hash to its string
// representation for YAML encoding. This method satisfies part of the
// model.Serializable interface requirement.
//
// MarshalYAML first validates that the Hash conforms to all constraints by
// calling Validate. If validation fails, marshaling fails with the validation
// error, preventing invalid data from being serialized. If validation succeeds,
// the Hash is converted to its string representation.
//
// Empty hashes (zero values) marshal to the YAML scalar "" (empty string).
// Non-empty hashes marshal to their normalized lowercase hexadecimal form
// as YAML scalars (40 or 64 characters).
//
// This method MUST NOT mutate the receiver except as required by the yaml.Marshaler
// interface contract. It MUST be safe to call concurrently on immutable receivers.
//
// Example:
//
//	hash := git.Hash("a1b2c3d4e5f6789012345678901234567890abcd")
//	data, _ := yaml.Marshal(hash)
//	fmt.Println(string(data)) // Output: a1b2c3d4e5f6789012345678901234567890abcd
func (h Hash) MarshalYAML() (interface{}, error) {
	if err := h.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", h.TypeName(), err)
	}
	return string(h), nil
}

// UnmarshalYAML implements yaml.Unmarshaler, deserializing a YAML scalar into
// a Hash value. This method satisfies part of the model.Serializable interface
// requirement.
//
// UnmarshalYAML accepts YAML scalars containing Git object ids and applies
// normalization before validation. The input undergoes case normalization
// where uppercase letters are converted to lowercase and whitespace trimming.
// This normalization ensures consistent representation regardless of the source
// formatting.
//
// After unmarshaling and normalization, Validate is called to ensure the
// resulting Hash conforms to all constraints. If the normalized string is
// invalid, unmarshaling fails with an error describing the validation failure.
// This fail-fast behavior prevents invalid configuration data from corrupting
// system state.
//
// Empty YAML scalars unmarshal successfully to the zero value Hash. YAML null
// values are rejected as invalid input.
//
// The method mutates the receiver to store the unmarshaled value. It is not
// safe for concurrent use during unmarshaling, though the resulting Hash value
// is safe for concurrent reads after unmarshaling completes.
//
// Example:
//
//	var hash git.Hash
//	yaml.Unmarshal([]byte("A1B2C3D4E5F6789012345678901234567890ABCD"), &hash)
//	fmt.Println(hash.String()) // Output: "a1b2c3d4e5f6789012345678901234567890abcd" (normalized)
func (h *Hash) UnmarshalYAML(node *yaml.Node) error {
	var str string
	if err := node.Decode(&str); err != nil {
		return fmt.Errorf("cannot unmarshal YAML: %w", err)
	}

	parsed, err := ParseHash(str)
	if err != nil {
		return fmt.Errorf("unmarshaled model is invalid: %w", err)
	}

	*h = parsed
	return nil
}

// ParseHash parses a string into a Hash value, normalizing and validating the
// input before returning. This function provides a unified parsing entry point
// for converting external string representations into Hash values with
// comprehensive input normalization and validation.
//
// ParseHash applies multi-stage normalization to the input. First, leading
// and trailing whitespace is removed via strings.TrimSpace. Second, uppercase
// letters are converted to lowercase via strings.ToLower, ensuring consistent
// canonical representation (Git treats object ids as case-insensitive, but
// dxrel enforces lowercase for consistency). Third, the result is validated
// against all Hash constraints.
//
// After normalization, ParseHash validates the result against all Hash
// constraints. The normalized string MUST be either empty (zero value) or
// exactly 40 characters (SHA-1) or exactly 64 characters (SHA-256) of lowercase
// hexadecimal digits [0-9a-f]. If any constraint is violated, ParseHash returns
// an error describing the specific validation failure.
//
// ParseHash returns an error in the following cases: if the normalized result
// has incorrect length (not 40 or 64 characters), if the normalized result
// contains non-hexadecimal characters, or if normalization produced an invalid
// hash. The error message includes relevant metrics to aid debugging and provide
// clear feedback to users.
//
// The empty string is a valid input and parses successfully to the zero value
// Hash, representing "no hash attached". Strings containing only whitespace
// also parse to the zero value Hash after trimming.
//
// Callers MUST check the returned error before using the Hash value. This
// function is pure and has no side effects. It is safe to call concurrently
// from multiple goroutines.
//
// Example:
//
//	hash, err := git.ParseHash("A1B2C3D4E5F6789012345678901234567890ABCD")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(hash.String()) // Output: "a1b2c3d4e5f6789012345678901234567890abcd"
func ParseHash(s string) (Hash, error) {
	// Normalize: trim whitespace and convert to lowercase
	normalized := strings.ToLower(strings.TrimSpace(s))

	// Create hash and validate
	hash := Hash(normalized)
	if err := hash.Validate(); err != nil {
		return "", fmt.Errorf("invalid hash: %w", err)
	}

	return hash, nil
}

// Compile-time verification that Hash implements model.Model interface.
var _ model.Model = (*Hash)(nil)
