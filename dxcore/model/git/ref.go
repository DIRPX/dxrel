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

	"dirpx.dev/dxrel/dxcore/model"
	"gopkg.in/yaml.v3"
)

// Ref represents a resolved Git reference, tying together the human-facing
// name, its kind (branch, tag, etc.), and the target commit hash.
//
// This type implements the model.Model interface, providing validation,
// serialization to JSON and YAML, safe logging, type identification, and
// zero-value detection.
//
// A Ref combines three key pieces of information:
//   - Name: The symbolic reference name (e.g., "main", "refs/heads/main", "v1.0.0")
//   - Kind: The category of reference (branch, tag, remote branch, etc.)
//   - Hash: The commit SHA that this reference currently points to
//
// This representation is useful for tracking resolved Git references in
// operations like release planning, commit analysis, and version tracking,
// where both the symbolic name and the concrete commit id are needed.
//
// The zero value of Ref (all fields zero) is valid and represents "no ref
// specified", indicating that a Git reference has not been provided or
// resolved. Zero-value Refs will fail validation if Validate() is called.
//
// Example values:
//
//	Ref{
//	    Name: "refs/heads/main",
//	    Kind: RefKindBranch,
//	    Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
//	}
//
//	Ref{
//	    Name: "v1.2.3",
//	    Kind: RefKindTag,
//	    Hash: "1234567890abcdef1234567890abcdef12345678",
//	}
type Ref struct {
	// Name is the original ref name (symbolic), as requested or listed.
	//
	// This preserves the exact form of the reference as it appears in Git,
	// whether that's a short name like "main", a full ref like
	// "refs/heads/main", or a tag name like "v1.0.0".
	//
	// Examples:
	//   - "main"
	//   - "refs/heads/main"
	//   - "v1.2.3"
	//   - "refs/tags/v1.2.3"
	//   - "refs/remotes/origin/develop"
	//
	// For direct hash refs (RefKindHash), Name MAY equal the hex Hash string,
	// or it MAY be a symbolic name that resolved to that hash.
	Name RefName

	// Kind is the coarse category of the ref (branch, tag, remote, etc.).
	//
	// This categorization helps callers understand the nature of the
	// reference without having to parse the Name string. It indicates
	// whether this is a local branch, remote tracking branch, annotated
	// tag, lightweight tag, or direct commit hash.
	//
	// The Kind is typically determined by examining the Name prefix:
	//   - refs/heads/* -> RefKindBranch
	//   - refs/tags/* -> RefKindTag
	//   - refs/remotes/* -> RefKindRemoteBranch
	//   - 40 or 64 hex chars -> RefKindHash
	Kind RefKind

	// Hash is the resolved object id this ref points to.
	//
	// For branches and lightweight tags, this is the commit hash directly.
	// For annotated tags, this is typically the commit hash after peeling
	// the tag object, though higher-level code can decide whether to peel.
	//
	// The Hash MUST be a fully resolved Git object id in canonical form
	// (40-character SHA-1 or 64-character SHA-256 in lowercase hex).
	// Abbreviated hashes are not valid here; callers MUST expand them
	// via git rev-parse or similar before creating a Ref.
	Hash Hash
}

// Compile-time check that Ref implements model.Model
var _ model.Model = (*Ref)(nil)

// NewRef creates a new Ref with the given Name, Kind, and Hash.
//
// This is a convenience constructor that creates and validates a Ref in
// one step. If any of the components are invalid, NewRef returns a zero
// Ref and an error.
//
// Example usage:
//
//	ref, err := git.NewRef("refs/heads/main", git.RefKindBranch, "a1b2c3d...")
//	if err != nil {
//	    // handle error
//	}
func NewRef(name RefName, kind RefKind, hash Hash) (Ref, error) {
	ref := Ref{
		Name: name,
		Kind: kind,
		Hash: hash,
	}

	if err := ref.Validate(); err != nil {
		return Ref{}, err
	}

	return ref, nil
}

// String returns the human-readable representation of the Ref.
//
// This method implements the fmt.Stringer interface and satisfies the
// model.Loggable contract's String() requirement. The output includes
// all three components (Name, Kind, and Hash) for complete debugging
// visibility.
//
// Format: "Ref{Name:<name>, Kind:<kind>, Hash:<hash>}"
//
// Examples:
//
//	Ref{Name: "main", Kind: RefKindBranch, Hash: "a1b2c3d..."}.String()
//	// Output: "Ref{Name:main, Kind:branch, Hash:a1b2c3d4e5f67890abcdef1234567890abcdef12}"
//
//	Ref{Name: "v1.0.0", Kind: RefKindTag, Hash: "1234567..."}.String()
//	// Output: "Ref{Name:v1.0.0, Kind:tag, Hash:1234567890abcdef1234567890abcdef12345678}"
//
//	Ref{}.String()
//	// Output: "Ref{Name:, Kind:unknown, Hash:}"
func (r Ref) String() string {
	return fmt.Sprintf("Ref{Name:%s, Kind:%s, Hash:%s}",
		r.Name.String(),
		r.Kind.String(),
		r.Hash.String())
}

// Redacted returns a safe, concise representation of the Ref suitable for
// production logging.
//
// This method implements the model.Loggable contract's Redacted() requirement.
// For Ref, all components (Name, Kind, Hash) are safe to log in production
// as they do not contain sensitive information like credentials or secrets.
// The Name is a public Git reference, Kind is a category enum, and Hash is
// a public commit id.
//
// The implementation delegates to the Redacted() methods of the component
// types (RefName, RefKind, Hash), which may apply their own formatting for
// brevity. For example, Hash.Redacted() abbreviates hashes to 7 characters.
//
// Format: "Ref{Name:<name>, Kind:<kind>, Hash:<hash-short>}"
//
// Examples:
//
//	Ref{Name: "main", Kind: RefKindBranch, Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef12"}.Redacted()
//	// Output: "Ref{Name:main, Kind:branch, Hash:a1b2c3d}"
//
//	Ref{Name: "v1.0.0", Kind: RefKindTag, Hash: "1234567890abcdef1234567890abcdef12345678"}.Redacted()
//	// Output: "Ref{Name:v1.0.0, Kind:tag, Hash:1234567}"
func (r Ref) Redacted() string {
	return fmt.Sprintf("Ref{Name:%s, Kind:%s, Hash:%s}",
		r.Name.Redacted(),
		r.Kind.Redacted(),
		r.Hash.Redacted())
}

// TypeName returns the name of this type for error messages and debugging.
//
// This method implements the model.Identifiable contract.
func (r Ref) TypeName() string {
	return "Ref"
}

// IsZero reports whether this Ref is the zero value.
//
// This method implements the model.ZeroCheckable contract. A Ref is
// considered zero if all three components (Name, Kind, Hash) are zero.
//
// Zero Refs are semantically invalid for most operations and will fail
// validation if Validate() is called. However, the zero value is useful
// as a sentinel for "no ref specified" in optional fields or when
// initializing data structures.
//
// Examples:
//
//	Ref{}.IsZero()  // true
//	Ref{Name: "main"}.IsZero()  // false (has Name)
//	Ref{Hash: "a1b2c3d..."}.IsZero()  // false (has Hash)
func (r Ref) IsZero() bool {
	return r.Name.IsZero() && r.Kind.IsZero() && r.Hash.IsZero()
}

// Equal reports whether this Ref is equal to another Ref.
//
// Two Refs are equal if all three components match:
//   - Name must be equal (case-sensitive string comparison)
//   - Kind must be equal (same RefKind value)
//   - Hash must be equal (case-sensitive string comparison)
//
// This method is used for testing, assertions, and deduplication logic.
//
// Examples:
//
//	r1 := Ref{Name: "main", Kind: RefKindBranch, Hash: "abc123..."}
//	r2 := Ref{Name: "main", Kind: RefKindBranch, Hash: "abc123..."}
//	r1.Equal(r2)  // true
//
//	r3 := Ref{Name: "main", Kind: RefKindBranch, Hash: "def456..."}
//	r1.Equal(r3)  // false (different Hash)
func (r Ref) Equal(other Ref) bool {
	return r.Name.Equal(other.Name) &&
		r.Kind.Equal(other.Kind) &&
		r.Hash.Equal(other.Hash)
}

// Validate checks whether this Ref satisfies all model contracts and
// invariants.
//
// This method implements the model.Validatable contract. Validation ensures:
//   - Name is valid (delegates to RefName.Validate)
//   - Kind is valid (delegates to RefKind.Validate)
//   - Hash is valid (delegates to Hash.Validate)
//   - At least one of Name or Hash is non-zero (a Ref must identify something)
//   - Kind and Name are consistent if both are non-zero and Kind is not RefKindUnknown
//
// The consistency check verifies that the Kind matches what would be inferred
// from the Name structure:
//   - refs/heads/* must have Kind = RefKindBranch
//   - refs/tags/* must have Kind = RefKindTag
//   - refs/remotes/* must have Kind = RefKindRemoteBranch
//   - HEAD, FETCH_HEAD, etc. must have Kind = RefKindHead
//   - 40/64 hex chars must have Kind = RefKindHash
//
// Zero-value Refs (all fields zero) will fail validation. Partial Refs
// where some but not all fields are set may be valid depending on context,
// but typically all three fields should be populated for a fully resolved
// Git reference.
//
// Validation is performed recursively by calling Validate on each component
// type. Component-specific validation rules (e.g., Hash format, RefName
// character restrictions) are enforced by those types.
//
// Returns nil if the Ref is valid, or a descriptive error if validation fails.
//
// Examples:
//
//	Ref{Name: "refs/heads/main", Kind: RefKindBranch, Hash: "a1b2c3d..."}.Validate()
//	// Returns: nil (valid)
//
//	Ref{}.Validate()
//	// Returns: error "Ref must have at least Name or Hash"
//
//	Ref{Name: "refs/heads/main", Kind: RefKindTag, Hash: "a1b2c3d..."}.Validate()
//	// Returns: error "Kind mismatch: Name implies branch but got tag"
func (r Ref) Validate() error {
	// At least Name or Hash must be non-zero for a meaningful Ref
	if r.Name.IsZero() && r.Hash.IsZero() {
		return fmt.Errorf("%s must have at least Name or Hash", r.TypeName())
	}

	// Validate Name if present
	if !r.Name.IsZero() {
		if err := r.Name.Validate(); err != nil {
			return fmt.Errorf("invalid %s Name: %w", r.TypeName(), err)
		}
	}

	// Validate Kind if present
	if !r.Kind.IsZero() {
		if err := r.Kind.Validate(); err != nil {
			return fmt.Errorf("invalid %s Kind: %w", r.TypeName(), err)
		}
	}

	// Validate Hash if present
	if !r.Hash.IsZero() {
		if err := r.Hash.Validate(); err != nil {
			return fmt.Errorf("invalid %s Hash: %w", r.TypeName(), err)
		}
	}

	// Cross-validate Kind and Name consistency if both are present and non-zero
	if !r.Name.IsZero() && !r.Kind.IsZero() && r.Kind != RefKindUnknown {
		expectedKind := inferRefKindFromName(r.Name)
		if expectedKind != RefKindUnknown && expectedKind != r.Kind {
			return fmt.Errorf("%s Kind mismatch: Name %q implies Kind %q but got %q",
				r.TypeName(), r.Name, expectedKind.String(), r.Kind.String())
		}
	}

	return nil
}

// inferRefKindFromName determines the expected RefKind based on the structure
// of a RefName.
//
// This function examines the RefName string to identify structural patterns
// that indicate the kind of reference:
//   - refs/heads/* -> RefKindBranch
//   - refs/tags/* -> RefKindTag
//   - refs/remotes/* -> RefKindRemoteBranch
//   - HEAD, FETCH_HEAD, ORIG_HEAD, MERGE_HEAD -> RefKindHead
//   - 40 or 64 lowercase hex characters -> RefKindHash
//
// If the name does not match any known pattern, RefKindUnknown is returned.
//
// This function is used internally by Validate to check consistency between
// the Name and Kind fields of a Ref.
func inferRefKindFromName(name RefName) RefKind {
	s := string(name)

	// Check for standard ref prefixes
	if len(s) >= 11 && s[:11] == "refs/heads/" {
		return RefKindBranch
	}
	if len(s) >= 10 && s[:10] == "refs/tags/" {
		return RefKindTag
	}
	if len(s) >= 13 && s[:13] == "refs/remotes/" {
		return RefKindRemoteBranch
	}

	// Check for special HEAD refs
	if s == "HEAD" || s == "FETCH_HEAD" || s == "ORIG_HEAD" || s == "MERGE_HEAD" {
		return RefKindHead
	}

	// Check for full hash (40 or 64 hex chars)
	if len(s) == HashHexSizeSHA1 || len(s) == HashHexSizeSHA256 {
		// Quick hex check
		allHex := true
		for i := 0; i < len(s); i++ {
			c := s[i]
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				allHex = false
				break
			}
		}
		if allHex {
			return RefKindHash
		}
	}

	return RefKindUnknown
}

// MarshalJSON serializes this Ref to JSON.
//
// This method implements the json.Marshaler interface and the
// model.Serializable contract.
//
// The JSON format is an object with three fields:
//
//	{
//	  "name": "refs/heads/main",
//	  "kind": "branch",
//	  "hash": "a1b2c3d4e5f67890abcdef1234567890abcdef12"
//	}
//
// All three fields are always present in the output, even if some are
// zero values. This ensures consistent JSON structure and makes parsing
// straightforward.
//
// Before encoding, MarshalJSON calls Validate. If the Ref is invalid,
// the validation error is returned and no JSON is produced.
func (r Ref) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", r.TypeName(), err)
	}

	// Use type alias to avoid infinite recursion
	type ref Ref
	return json.Marshal(ref(r))
}

// UnmarshalJSON deserializes a Ref from JSON.
//
// This method implements the json.Unmarshaler interface and the
// model.Serializable contract.
//
// The expected JSON format is an object with three fields:
//
//	{
//	  "name": "refs/heads/main",
//	  "kind": "branch",
//	  "hash": "a1b2c3d4e5f67890abcdef1234567890abcdef12"
//	}
//
// After unmarshaling, Validate is called to ensure the deserialized Ref
// satisfies all invariants. If validation fails, the error is returned
// and the Ref MUST NOT be used.
func (r *Ref) UnmarshalJSON(data []byte) error {
	type ref Ref
	if err := json.Unmarshal(data, (*ref)(r)); err != nil {
		return fmt.Errorf("cannot unmarshal JSON into %s: %w", r.TypeName(), err)
	}

	if err := r.Validate(); err != nil {
		return fmt.Errorf("unmarshaled %s is invalid: %w", r.TypeName(), err)
	}

	return nil
}

// MarshalYAML serializes this Ref to YAML.
//
// This method implements the yaml.Marshaler interface and the
// model.Serializable contract.
//
// The YAML format is an object with three fields:
//
//	name: refs/heads/main
//	kind: branch
//	hash: a1b2c3d4e5f67890abcdef1234567890abcdef12
//
// All three fields are always present in the output, even if some are
// zero values. This ensures consistent YAML structure.
//
// Before encoding, MarshalYAML calls Validate. If the Ref is invalid,
// the validation error is returned and no YAML is produced.
func (r Ref) MarshalYAML() (interface{}, error) {
	if err := r.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", r.TypeName(), err)
	}

	// Use type alias to avoid infinite recursion
	type ref Ref
	return ref(r), nil
}

// UnmarshalYAML deserializes a Ref from YAML.
//
// This method implements the yaml.Unmarshaler interface and the
// model.Serializable contract.
//
// The expected YAML format is an object with three fields:
//
//	name: refs/heads/main
//	kind: branch
//	hash: a1b2c3d4e5f67890abcdef1234567890abcdef12
//
// After unmarshaling, Validate is called to ensure the deserialized Ref
// satisfies all invariants. If validation fails, the error is returned
// and the Ref MUST NOT be used.
func (r *Ref) UnmarshalYAML(node *yaml.Node) error {
	type ref Ref
	if err := node.Decode((*ref)(r)); err != nil {
		return fmt.Errorf("cannot unmarshal YAML into %s: %w", r.TypeName(), err)
	}

	if err := r.Validate(); err != nil {
		return fmt.Errorf("unmarshaled %s is invalid: %w", r.TypeName(), err)
	}

	return nil
}
