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

// CommitRangeSpec describes a user-facing, unresolved range specification in
// terms of symbolic ref names (branches, tags, etc.) that need to be resolved
// into concrete commit hashes before use. This type represents the user's
// intent for a commit range before Git resolution occurs.
//
// CommitRangeSpec is the "specification" or "request" form of a commit range,
// containing only symbolic names (RefName values) that the user provides. These
// names must be resolved through Git operations (git rev-parse, git show-ref,
// etc.) to obtain concrete commit hashes, which are then used to construct a
// fully-resolved CommitRange.
//
// The relationship between CommitRangeSpec and CommitRange is:
//   - CommitRangeSpec: User input with symbolic names -> "what the user wants"
//   - CommitRange: Resolved with concrete hashes -> "what Git will process"
//
// CommitRangeSpec follows the same Git range semantics as CommitRange:
//   - From is the exclusive lower bound (symbolic name)
//   - To is the inclusive upper bound (symbolic name)
//   - Zero From (empty RefName) means "from the beginning of history"
//
// The zero value of CommitRangeSpec (both From and To are empty) is valid at
// the Go type level but will fail validation if Validate() is called, as a
// valid range specification requires at least a non-zero To ref name.
//
// Example commit range specifications:
//
//	// Range from tag v1.0.0 to tag v2.0.0
//	// User specifies: "from v1.0.0 to v2.0.0"
//	CommitRangeSpec{
//	    From: "v1.0.0",
//	    To:   "v2.0.0",
//	}
//
//	// Range from beginning of history to HEAD
//	// User specifies: "everything up to HEAD"
//	CommitRangeSpec{
//	    From: "",      // Empty = from beginning
//	    To:   "HEAD",
//	}
//
//	// Range from main branch to develop branch
//	// User specifies: "main..develop"
//	CommitRangeSpec{
//	    From: "main",
//	    To:   "develop",
//	}
//
//	// Range from last tag to current branch
//	// User specifies: "v1.0.0..HEAD"
//	CommitRangeSpec{
//	    From: "v1.0.0",
//	    To:   "HEAD",
//	}
//
// Typical workflow:
//  1. User provides a CommitRangeSpec (symbolic names)
//  2. Application resolves From and To names to concrete Ref values with hashes
//  3. Application constructs a CommitRange from the resolved Refs
//  4. Application uses CommitRange for Git operations (log, diff, etc.)
//
// This type implements the model.Model interface, providing validation,
// serialization, logging, and equality operations. CommitRangeSpec values are
// safe for concurrent use as all fields are value types or immutable.
type CommitRangeSpec struct {
	// From is the symbolic name of the exclusive lower bound.
	//
	// This RefName represents the user's specification of where the range
	// should start (exclusive). Before this range can be used with Git, the
	// From name must be resolved to a concrete commit hash through Git commands
	// like "git rev-parse From" or "git show-ref From".
	//
	// A zero From (empty RefName "") has special semantics: it means "from
	// the beginning of history", effectively removing the lower bound. When
	// resolved, this becomes a zero Ref in the resulting CommitRange.
	//
	// Examples:
	//   - From = "v1.0.0"     -> Will resolve to tag v1.0.0's commit hash
	//   - From = "main"       -> Will resolve to main branch's current commit
	//   - From = ""           -> No resolution needed, means "from beginning"
	//   - From = "HEAD~10"    -> Will resolve to 10 commits before HEAD
	From RefName `json:"from" yaml:"from"`

	// To is the symbolic name of the inclusive upper bound.
	//
	// This RefName represents the user's specification of where the range
	// should end (inclusive). Unlike From, To MUST be non-zero for a valid
	// CommitRangeSpec. Before use, the To name must be resolved to a concrete
	// commit hash through Git commands.
	//
	// Examples:
	//   - To = "HEAD"         -> Will resolve to current HEAD commit
	//   - To = "v2.0.0"       -> Will resolve to tag v2.0.0's commit hash
	//   - To = "develop"      -> Will resolve to develop branch's current commit
	//   - To = "abc123..."    -> Direct hash, will validate and use as-is
	To RefName `json:"to" yaml:"to"`
}

// NewCommitRangeSpec constructs and validates a new CommitRangeSpec with the
// specified symbolic ref names for the range boundaries. This function provides
// a convenient and safe way to create CommitRangeSpec values with automatic
// validation of both boundary ref names.
//
// NewCommitRangeSpec accepts two RefName values representing the symbolic names
// for the exclusive lower bound (from) and inclusive upper bound (to) of the
// commit range. The from parameter MAY be an empty RefName to indicate "from
// the beginning of history", but the to parameter MUST be a non-empty, valid
// RefName.
//
// This constructor automatically validates the resulting CommitRangeSpec after
// construction, ensuring that all structural and semantic constraints are
// satisfied. If either boundary ref name is invalid, or if the constructed
// CommitRangeSpec fails validation, an error is returned.
//
// Parameters:
//   - from: The symbolic name for the exclusive lower bound. May be empty for
//     "from beginning".
//   - to: The symbolic name for the inclusive upper bound. Must be non-empty
//     and valid.
//
// Returns the validated CommitRangeSpec on success, or an error if validation
// fails.
//
// Example usage:
//
//	spec, err := git.NewCommitRangeSpec("v1.0.0", "v2.0.0")
//	if err != nil {
//	    return fmt.Errorf("invalid range spec: %w", err)
//	}
//	// spec represents the user's request for range v1.0.0..v2.0.0
//	// Next step: resolve to concrete hashes via Git
//
//	// From beginning to HEAD
//	spec, err := git.NewCommitRangeSpec("", "HEAD")
//	// spec represents ..HEAD (all history up to HEAD)
func NewCommitRangeSpec(from, to RefName) (CommitRangeSpec, error) {
	spec := CommitRangeSpec{
		From: from,
		To:   to,
	}

	if err := spec.Validate(); err != nil {
		return CommitRangeSpec{}, fmt.Errorf("invalid CommitRangeSpec: %w", err)
	}

	return spec, nil
}

// Compile-time assertion that CommitRangeSpec implements model.Model.
var _ model.Model = (*CommitRangeSpec)(nil)

// String returns a human-readable string representation of the CommitRangeSpec
// that displays the range boundaries in Git's standard notation suitable for
// logging, debugging, and user display. This method implements the
// model.Loggable contract through model.Model.
//
// The returned string uses Git's standard range notation "A..B" where A is
// the From boundary (exclusive) and B is the To boundary (inclusive). When
// From is empty (representing "from the beginning"), it uses the notation "..B"
// to indicate an unbounded lower range.
//
// Since CommitRangeSpec contains only symbolic names (not resolved hashes),
// the string representation shows only the ref names without hash information.
// This differs from CommitRange.String() which includes both names and hashes.
//
// This method MUST NOT mutate the CommitRangeSpec receiver and is safe for
// concurrent use from multiple goroutines.
//
// Example return values:
//
//	"v1.0.0..v2.0.0"    // Range between two tags
//	"..HEAD"            // From beginning to HEAD
//	"main..develop"     // Branch to branch range
//	"v1.0.0..HEAD"      // Tag to current commit
//	"(empty)..(empty)"  // Zero value (invalid)
func (crs CommitRangeSpec) String() string {
	fromStr := "(empty)"
	if !crs.From.IsZero() {
		fromStr = string(crs.From)
	}

	toStr := "(empty)"
	if !crs.To.IsZero() {
		toStr = string(crs.To)
	}

	return fmt.Sprintf("%s..%s", fromStr, toStr)
}

// Redacted returns a redacted string representation of the CommitRangeSpec
// suitable for logging and display where sensitive information should be
// obscured. This method implements the model.Loggable contract through
// model.Model.
//
// The returned string uses the same format as String() but delegates to the
// Redacted() methods of the From and To RefName fields to obscure any sensitive
// information. CommitRangeSpec itself does not contain directly sensitive
// information (ref names are typically public), but this method ensures
// consistent privacy handling across the entire object graph.
//
// This method MUST NOT mutate the CommitRangeSpec receiver and is safe for
// concurrent use from multiple goroutines.
//
// Example usage:
//
//	spec := git.CommitRangeSpec{From: "v1.0.0", To: "v2.0.0"}
//	redacted := spec.Redacted()
//	log.Info("Processing range spec", "spec", redacted)
//	// Output: "v1.0.0..v2.0.0" (ref names are typically not sensitive)
func (crs CommitRangeSpec) Redacted() string {
	fromStr := "(empty)"
	if !crs.From.IsZero() {
		fromStr = crs.From.Redacted()
	}

	toStr := "(empty)"
	if !crs.To.IsZero() {
		toStr = crs.To.Redacted()
	}

	return fmt.Sprintf("%s..%s", fromStr, toStr)
}

// TypeName returns the string "CommitRangeSpec", which identifies this type in
// logs, error messages, and serialized output. This method implements the
// model.Identifiable contract through model.Model.
//
// The returned type name is a simple, unqualified identifier that describes
// the semantic purpose of this type without package prefixes. It is used by
// logging frameworks, error messages, and introspection tools to identify the
// type of a model value.
//
// This method MUST NOT mutate the CommitRangeSpec receiver and is safe for
// concurrent use from multiple goroutines.
//
// Example usage:
//
//	var spec git.CommitRangeSpec
//	fmt.Println(spec.TypeName())  // Output: CommitRangeSpec
//
//	// In error messages
//	return fmt.Errorf("invalid %s: %w", spec.TypeName(), err)
//	// Output: invalid CommitRangeSpec: ...
func (crs CommitRangeSpec) TypeName() string {
	return "CommitRangeSpec"
}

// IsZero reports whether the CommitRangeSpec is the zero value, with both From
// and To ref names being empty. This method implements the model.ZeroCheckable
// contract through model.Model.
//
// A zero CommitRangeSpec has both From.IsZero() and To.IsZero() returning true,
// indicating that no range specification has been provided. This is distinct
// from a CommitRangeSpec with only From being empty (which represents "from
// the beginning of history to To"), where IsZero() returns false because To
// is non-empty.
//
// Zero-value CommitRangeSpecs are valid at the Go type level but will fail
// validation if Validate() is called, as a valid range specification requires
// at least a non-empty To ref name.
//
// This method MUST NOT mutate the CommitRangeSpec receiver and is safe for
// concurrent use from multiple goroutines.
//
// Example usage:
//
//	var spec git.CommitRangeSpec
//	if spec.IsZero() {
//	    fmt.Println("No range spec provided")
//	}
//
//	spec = git.CommitRangeSpec{
//	    From: "",      // Empty
//	    To:   "HEAD",
//	}
//	if !spec.IsZero() {
//	    fmt.Println("Range spec from beginning to HEAD")
//	}
func (crs CommitRangeSpec) IsZero() bool {
	return crs.From.IsZero() && crs.To.IsZero()
}

// Equal reports whether the CommitRangeSpec is semantically equal to another
// CommitRangeSpec by comparing the From and To ref names for equality using
// their respective Equal methods. This method implements the model.Model
// interface contract.
//
// Two CommitRangeSpecs are considered equal if and only if their From ref
// names are equal (via RefName.Equal) and their To ref names are equal (via
// RefName.Equal). This is a strict string comparison of the symbolic names.
//
// The other parameter must be a CommitRangeSpec or *CommitRangeSpec. If other
// is a different type, Equal returns false without panicking.
//
// This method MUST NOT mutate the CommitRangeSpec receiver and is safe for
// concurrent use from multiple goroutines.
//
// Example usage:
//
//	spec1 := git.CommitRangeSpec{From: "v1.0.0", To: "v2.0.0"}
//	spec2 := git.CommitRangeSpec{From: "v1.0.0", To: "v2.0.0"}
//	if spec1.Equal(spec2) {
//	    fmt.Println("Specs are equal")
//	}
//
//	spec3 := git.CommitRangeSpec{From: "v1.0.0", To: "HEAD"}
//	if !spec1.Equal(spec3) {
//	    fmt.Println("Specs differ in To boundary")
//	}
func (crs CommitRangeSpec) Equal(other any) bool {
	switch o := other.(type) {
	case CommitRangeSpec:
		return crs.From.Equal(o.From) && crs.To.Equal(o.To)
	case *CommitRangeSpec:
		if o == nil {
			return false
		}
		return crs.From.Equal(o.From) && crs.To.Equal(o.To)
	default:
		return false
	}
}

// Validate checks that the CommitRangeSpec satisfies all structural and
// semantic constraints, returning nil if valid or an error describing the
// first validation failure encountered. This method implements the
// model.Validatable contract through model.Model.
//
// Validation rules for CommitRangeSpec:
//   - The zero value (both From and To are empty) is INVALID. A valid range
//     spec requires at least a non-empty To ref name.
//   - To MUST be non-empty (To.IsZero() == false). The upper bound is required.
//   - From MAY be empty, which means "from the beginning of history".
//   - If From is non-empty, it MUST pass RefName.Validate().
//   - To MUST pass RefName.Validate() (required non-empty RefName).
//
// Validate does NOT check whether the ref names exist in a Git repository or
// whether they can be resolved to commits. That resolution happens later when
// converting CommitRangeSpec to CommitRange. This validation only checks that
// the ref names are structurally valid (correct format, length, characters).
//
// This method MUST NOT mutate the CommitRangeSpec receiver and is safe for
// concurrent use from multiple goroutines.
//
// Example usage:
//
//	spec := git.CommitRangeSpec{From: "v1.0.0", To: "v2.0.0"}
//	if err := spec.Validate(); err != nil {
//	    return fmt.Errorf("invalid range spec: %w", err)
//	}
//	// spec is now known to be structurally valid
//
//	// Empty To is invalid
//	spec = git.CommitRangeSpec{From: "v1.0.0", To: ""}
//	err := spec.Validate()  // Returns error about empty To
func (crs CommitRangeSpec) Validate() error {
	// Zero value is invalid - at minimum need a To ref name
	if crs.IsZero() {
		return fmt.Errorf("CommitRangeSpec is zero (both From and To are empty)")
	}

	// To MUST be non-empty
	if crs.To.IsZero() {
		return fmt.Errorf("CommitRangeSpec To is empty (To ref name is required)")
	}

	// Validate To (required)
	if err := crs.To.Validate(); err != nil {
		return fmt.Errorf("invalid CommitRangeSpec To: %w", err)
	}

	// Validate From if non-empty (empty From is allowed = from beginning)
	if !crs.From.IsZero() {
		if err := crs.From.Validate(); err != nil {
			return fmt.Errorf("invalid CommitRangeSpec From: %w", err)
		}
	}

	return nil
}

// MarshalJSON serializes the CommitRangeSpec to JSON format after validating
// that it represents a valid range specification. This method implements the
// json.Marshaler interface and is part of the model.Serializable contract
// through model.Model.
//
// MarshalJSON validates the CommitRangeSpec before marshaling to ensure that
// only valid range specifications can be serialized. If validation fails, an
// error is returned and no JSON is produced. This prevents invalid data from
// being written to storage or transmitted over APIs.
//
// The JSON structure uses a type alias (commitRangeSpecJSON) to delegate to
// the standard JSON encoder while avoiding infinite recursion. The resulting
// JSON has the structure:
//
//	{
//	  "from": "v1.0.0",
//	  "to": "v2.0.0"
//	}
//
// This method MUST NOT mutate the CommitRangeSpec receiver and is safe for
// concurrent use from multiple goroutines.
//
// Example usage:
//
//	spec := git.CommitRangeSpec{From: "v1.0.0", To: "v2.0.0"}
//	data, err := json.Marshal(spec)
//	if err != nil {
//	    return fmt.Errorf("marshal failed: %w", err)
//	}
//	// data contains: {"from":"v1.0.0","to":"v2.0.0"}
//
// Returns an error if validation fails or if JSON encoding fails.
func (crs CommitRangeSpec) MarshalJSON() ([]byte, error) {
	// Validate before marshaling
	if err := crs.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid CommitRangeSpec: %w", err)
	}

	// Use type alias to avoid infinite recursion
	type commitRangeSpecJSON CommitRangeSpec
	return json.Marshal(commitRangeSpecJSON(crs))
}

// UnmarshalJSON deserializes a CommitRangeSpec from JSON format and validates
// the result to ensure structural and semantic correctness. This method
// implements the json.Unmarshaler interface and is part of the
// model.Serializable contract through model.Model.
//
// UnmarshalJSON first decodes the JSON into the CommitRangeSpec structure using
// a type alias (commitRangeSpecJSON) to avoid infinite recursion, then validates
// the decoded value using Validate(). If the JSON is malformed or if the
// decoded CommitRangeSpec is invalid, an error is returned.
//
// This two-phase approach (decode then validate) ensures that only valid
// CommitRangeSpec values can be constructed from JSON, preventing invalid data
// from entering the system through deserialization.
//
// This method mutates the CommitRangeSpec receiver by decoding into it. It is
// NOT safe for concurrent use with other operations on the same CommitRangeSpec
// value.
//
// Example usage:
//
//	var spec git.CommitRangeSpec
//	err := json.Unmarshal(data, &spec)
//	if err != nil {
//	    return fmt.Errorf("unmarshal failed: %w", err)
//	}
//	// spec is now a valid CommitRangeSpec decoded from JSON
//
// Returns an error if JSON decoding fails or if validation fails.
func (crs *CommitRangeSpec) UnmarshalJSON(data []byte) error {
	// Use type alias to avoid infinite recursion
	type commitRangeSpecJSON CommitRangeSpec
	var temp commitRangeSpecJSON

	if err := json.Unmarshal(data, &temp); err != nil {
		return fmt.Errorf("failed to unmarshal CommitRangeSpec: %w", err)
	}

	*crs = CommitRangeSpec(temp)

	// Validate after unmarshaling
	if err := crs.Validate(); err != nil {
		return fmt.Errorf("invalid CommitRangeSpec after unmarshal: %w", err)
	}

	return nil
}

// MarshalYAML serializes the CommitRangeSpec to YAML format after validating
// that it represents a valid range specification. This method implements the
// yaml.Marshaler interface and is part of the model.Serializable contract
// through model.Model.
//
// MarshalYAML validates the CommitRangeSpec before marshaling to ensure that
// only valid range specifications can be serialized. If validation fails, an
// error is returned and no YAML is produced. This prevents invalid data from
// being written to configuration files or other YAML-based storage.
//
// The YAML structure uses a type alias (commitRangeSpecYAML) to delegate to
// the standard YAML encoder while avoiding infinite recursion. The resulting
// YAML has the structure:
//
//	from: v1.0.0
//	to: v2.0.0
//
// This method MUST NOT mutate the CommitRangeSpec receiver and is safe for
// concurrent use from multiple goroutines.
//
// Example usage:
//
//	spec := git.CommitRangeSpec{From: "v1.0.0", To: "v2.0.0"}
//	data, err := yaml.Marshal(spec)
//	if err != nil {
//	    return fmt.Errorf("marshal failed: %w", err)
//	}
//
// Returns an error if validation fails or if YAML encoding fails.
func (crs CommitRangeSpec) MarshalYAML() (interface{}, error) {
	// Validate before marshaling
	if err := crs.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid CommitRangeSpec: %w", err)
	}

	// Use type alias to avoid infinite recursion
	type commitRangeSpecYAML CommitRangeSpec
	return commitRangeSpecYAML(crs), nil
}

// UnmarshalYAML deserializes a CommitRangeSpec from YAML format and validates
// the result to ensure structural and semantic correctness. This method
// implements the yaml.Unmarshaler interface and is part of the
// model.Serializable contract through model.Model.
//
// UnmarshalYAML first decodes the YAML into the CommitRangeSpec structure using
// a type alias (commitRangeSpecYAML) to avoid infinite recursion, then validates
// the decoded value using Validate(). If the YAML is malformed or if the
// decoded CommitRangeSpec is invalid, an error is returned.
//
// This two-phase approach (decode then validate) ensures that only valid
// CommitRangeSpec values can be constructed from YAML, preventing invalid data
// from entering the system through configuration files or other YAML sources.
//
// This method mutates the CommitRangeSpec receiver by decoding into it. It is
// NOT safe for concurrent use with other operations on the same CommitRangeSpec
// value.
//
// Example usage:
//
//	var spec git.CommitRangeSpec
//	err := yaml.Unmarshal(data, &spec)
//	if err != nil {
//	    return fmt.Errorf("unmarshal failed: %w", err)
//	}
//	// spec is now a valid CommitRangeSpec decoded from YAML
//
// Returns an error if YAML decoding fails or if validation fails.
func (crs *CommitRangeSpec) UnmarshalYAML(node *yaml.Node) error {
	// Use type alias to avoid infinite recursion
	type commitRangeSpecYAML CommitRangeSpec
	var temp commitRangeSpecYAML

	if err := node.Decode(&temp); err != nil {
		return fmt.Errorf("failed to unmarshal CommitRangeSpec: %w", err)
	}

	*crs = CommitRangeSpec(temp)

	// Validate after unmarshaling
	if err := crs.Validate(); err != nil {
		return fmt.Errorf("invalid CommitRangeSpec after unmarshal: %w", err)
	}

	return nil
}
