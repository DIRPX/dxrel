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

// CommitRange represents a resolved, ref-backed range of commits that defines
// a span of Git history between two boundary points. This type is used to
// specify and analyze commit ranges in release planning, changelog generation,
// and version management workflows.
//
// CommitRange follows Git's standard range notation semantics as implemented
// by "git log A..B", where the range includes all commits reachable from B
// but not reachable from A. This means:
//   - From is the exclusive lower bound (commits reachable from From are excluded)
//   - To is the inclusive upper bound (commits reachable from To are included)
//
// The zero value of CommitRange (both From and To are zero Refs) is valid at
// the Go type level but will fail validation if Validate() is called, as a
// valid commit range requires at least a non-zero To boundary.
//
// A CommitRange with a zero From (From.Hash == "") represents a range "from
// the beginning of history", which is semantically equivalent to "git log B"
// or "git log ..B". This is useful for initial releases or when analyzing all
// commits leading up to a specific point.
//
// Both From and To are Ref values that combine symbolic names (branch names,
// tag names) with concrete commit hashes. This dual representation preserves
// both how the boundary was originally specified (metadata) and what specific
// commit it resolved to (for deterministic range computation).
//
// The range is defined primarily by the underlying Hash values in From.Hash
// and To.Hash. The Name and Kind fields on each Ref are preserved as metadata
// describing how the boundary was originally specified (branch, tag, direct
// hash, etc.), which is useful for reporting and display purposes.
//
// Example commit ranges:
//
//	// Range from tag v1.0.0 (exclusive) to HEAD (inclusive)
//	// Represents all commits since v1.0.0 release
//	CommitRange{
//	    From: Ref{Name: "v1.0.0", Kind: RefKindTag, Hash: "abc123..."},
//	    To:   Ref{Name: "HEAD", Kind: RefKindBranch, Hash: "def456..."},
//	}
//
//	// Range from beginning of history to tag v1.0.0 (inclusive)
//	// Represents all commits in the first release
//	CommitRange{
//	    From: Ref{},  // Zero value = from beginning
//	    To:   Ref{Name: "v1.0.0", Kind: RefKindTag, Hash: "abc123..."},
//	}
//
//	// Range between two tags for a specific release span
//	// Represents all commits from v1.0.0 to v2.0.0
//	CommitRange{
//	    From: Ref{Name: "v1.0.0", Kind: RefKindTag, Hash: "abc123..."},
//	    To:   Ref{Name: "v2.0.0", Kind: RefKindTag, Hash: "xyz789..."},
//	}
//
//	// Range from specific commit hash to branch tip
//	// Useful for partial history analysis
//	CommitRange{
//	    From: Ref{Name: "abc123...", Kind: RefKindHash, Hash: "abc123..."},
//	    To:   Ref{Name: "main", Kind: RefKindBranch, Hash: "def456..."},
//	}
//
// This type implements the model.Model interface, providing validation,
// serialization, logging, and equality operations. CommitRange values are
// safe for concurrent use as all fields are value types or immutable.
type CommitRange struct {
	// From is the exclusive lower bound of the commit range.
	//
	// Commits reachable from From.Hash are excluded from the range. This
	// represents the "starting point" that defines where the range begins,
	// but that starting point itself is not included in the result.
	//
	// A zero From (From.Hash == "") has special semantics: it means "from
	// the beginning of history", effectively removing the lower bound and
	// including all commits reachable from To. This is useful for initial
	// releases or when analyzing complete history up to a specific point.
	//
	// The From.Name and From.Kind fields preserve metadata about how this
	// boundary was originally specified (e.g., as a tag name "v1.0.0" or
	// branch name "release/1.x"), which is useful for display and reporting
	// purposes even though the actual range computation uses From.Hash.
	//
	// Examples:
	//   - From.Name = "v1.2.3", From.Kind = RefKindTag, From.Hash = <tag commit hash>
	//     -> Range excludes v1.2.3 and all its ancestors
	//   - From.Name = "", From.Kind = RefKindUnknown, From.Hash = ""
	//     -> Range includes all history (no lower bound)
	//   - From.Name = "abc123...", From.Kind = RefKindHash, From.Hash = "abc123..."
	//     -> Range excludes specific commit and its ancestors
	From Ref `json:"from" yaml:"from"`

	// To is the inclusive upper bound of the commit range.
	//
	// Commits reachable from To.Hash are included in the range. This represents
	// the "end point" that defines what commits to include, and that end point
	// itself is part of the result.
	//
	// Unlike From, To MUST be a non-zero Ref for a valid CommitRange. The
	// To.Hash field must contain a valid commit hash that was resolved from
	// whatever the user specified (HEAD, branch name, tag, explicit hash).
	// A CommitRange with a zero To is invalid and will fail validation.
	//
	// The To.Name and To.Kind fields preserve metadata about how this boundary
	// was originally specified, which is useful for display purposes. For
	// example, knowing that To came from "HEAD" vs "main" vs "v2.0.0" helps
	// with user communication and reporting.
	//
	// Examples:
	//   - To.Name = "HEAD", To.Kind = RefKindBranch, To.Hash = <current HEAD hash>
	//     -> Range includes up to current HEAD
	//   - To.Name = "main", To.Kind = RefKindBranch, To.Hash = <refs/heads/main hash>
	//     -> Range includes up to main branch tip
	//   - To.Name = "v2.0.0", To.Kind = RefKindTag, To.Hash = <tag commit hash>
	//     -> Range includes up to v2.0.0 release
	To Ref `json:"to" yaml:"to"`
}

// NewCommitRange constructs and validates a new CommitRange with the specified
// boundary references. This function provides a convenient and safe way to
// create CommitRange values with automatic validation of both boundary refs.
//
// NewCommitRange accepts two Ref values representing the exclusive lower bound
// (from) and inclusive upper bound (to) of the commit range. The from parameter
// MAY be a zero Ref to indicate "from the beginning of history", but the to
// parameter MUST be a non-zero, valid Ref. Both refs are validated through
// their respective Validate() methods before constructing the CommitRange.
//
// This constructor automatically validates the resulting CommitRange after
// construction, ensuring that all structural and semantic constraints are
// satisfied. If either boundary ref is invalid, or if the constructed
// CommitRange fails validation, an error is returned.
//
// Parameters:
//   - from: The exclusive lower bound ref. May be zero for "from beginning".
//   - to: The inclusive upper bound ref. Must be non-zero and valid.
//
// Returns the validated CommitRange on success, or an error if validation fails.
//
// Example usage:
//
//	fromRef := git.Ref{Name: "v1.0.0", Kind: git.RefKindTag, Hash: "abc123..."}
//	toRef := git.Ref{Name: "v2.0.0", Kind: git.RefKindTag, Hash: "def456..."}
//	cr, err := git.NewCommitRange(fromRef, toRef)
//	if err != nil {
//	    return fmt.Errorf("invalid commit range: %w", err)
//	}
//	// cr is now a valid CommitRange from v1.0.0 to v2.0.0
//
//	// From beginning of history to HEAD
//	headRef := git.Ref{Name: "HEAD", Kind: git.RefKindBranch, Hash: "xyz789..."}
//	cr, err := git.NewCommitRange(git.Ref{}, headRef)
//	// cr represents complete history up to HEAD
func NewCommitRange(from, to Ref) (CommitRange, error) {
	cr := CommitRange{
		From: from,
		To:   to,
	}

	if err := cr.Validate(); err != nil {
		return CommitRange{}, fmt.Errorf("invalid CommitRange: %w", err)
	}

	return cr, nil
}

// Compile-time assertion that CommitRange implements model.Model.
var _ model.Model = (*CommitRange)(nil)

// String returns a human-readable string representation of the CommitRange
// that displays the range boundaries in a compact format suitable for logging,
// debugging, and user display. This method implements the model.Loggable
// contract through model.Model.
//
// The returned string uses Git's standard range notation "A..B" where A is
// the From boundary (exclusive) and B is the To boundary (inclusive). When
// From is zero (representing "from the beginning"), it uses the notation "..B"
// to indicate an unbounded lower range.
//
// For each boundary that has a non-empty Name, the string includes both the
// symbolic name and the short form of the hash (first 7 characters). When a
// boundary has no symbolic name (direct hash reference), only the short hash
// is shown.
//
// This method MUST NOT mutate the CommitRange receiver and is safe for
// concurrent use from multiple goroutines.
//
// Example return values:
//
//	"v1.0.0(abc1234)..v2.0.0(def5678)"  // Named range between two tags
//	"..HEAD(abc1234)"                    // From beginning to HEAD
//	"abc1234..def5678"                   // Direct hash range (no names)
//	"main(abc1234)..develop(def5678)"    // Branch to branch range
//	"(zero)..main(abc1234)"              // Zero from to main
func (cr CommitRange) String() string {
	fromStr := "(zero)"
	if !cr.From.IsZero() {
		if cr.From.Name != "" {
			fromStr = fmt.Sprintf("%s(%s)", cr.From.Name, shortHash(cr.From.Hash))
		} else {
			fromStr = string(shortHash(cr.From.Hash))
		}
	}

	toStr := "(zero)"
	if !cr.To.IsZero() {
		if cr.To.Name != "" {
			toStr = fmt.Sprintf("%s(%s)", cr.To.Name, shortHash(cr.To.Hash))
		} else {
			toStr = string(shortHash(cr.To.Hash))
		}
	}

	return fmt.Sprintf("%s..%s", fromStr, toStr)
}

// shortHash returns the first 7 characters of a hash for compact display.
func shortHash(h Hash) Hash {
	if len(h) > 7 {
		return h[:7]
	}
	return h
}

// Redacted returns a redacted string representation of the CommitRange suitable
// for logging and display where sensitive information should be obscured. This
// method implements the model.Loggable contract through model.Model.
//
// The returned string uses the same format as String() but delegates to the
// Redacted() methods of the From and To Ref fields to obscure any sensitive
// information within those refs. CommitRange itself does not contain directly
// sensitive information (commit hashes and ref names are typically public), but
// this method ensures consistent privacy handling across the entire object graph.
//
// The format matches String(): "A..B" where A is the From boundary (exclusive)
// and B is the To boundary (inclusive), with each boundary potentially redacted
// according to Ref.Redacted() behavior.
//
// This method MUST NOT mutate the CommitRange receiver and is safe for
// concurrent use from multiple goroutines.
//
// Example usage:
//
//	cr := git.CommitRange{From: fromRef, To: toRef}
//	redacted := cr.Redacted()
//	log.Info("Processing range", "range", redacted)
//	// Output might be: "v1.0.0(abc1234)..v2.0.0(def5678)"
func (cr CommitRange) Redacted() string {
	fromStr := "(zero)"
	if !cr.From.IsZero() {
		fromStr = cr.From.Redacted()
	}

	toStr := "(zero)"
	if !cr.To.IsZero() {
		toStr = cr.To.Redacted()
	}

	return fmt.Sprintf("%s..%s", fromStr, toStr)
}

// TypeName returns the string "CommitRange", which identifies this type in
// logs, error messages, and serialized output. This method implements the
// model.Identifiable contract through model.Model.
//
// The returned type name is a simple, unqualified identifier that describes
// the semantic purpose of this type without package prefixes. It is used by
// logging frameworks, error messages, and introspection tools to identify the
// type of a model value.
//
// This method MUST NOT mutate the CommitRange receiver and is safe for
// concurrent use from multiple goroutines.
//
// Example usage:
//
//	var cr git.CommitRange
//	fmt.Println(cr.TypeName())  // Output: CommitRange
//
//	// In error messages
//	return fmt.Errorf("invalid %s: %w", cr.TypeName(), err)
//	// Output: invalid CommitRange: ...
func (cr CommitRange) TypeName() string {
	return "CommitRange"
}

// IsZero reports whether the CommitRange is the zero value, with both From
// and To boundaries being zero Refs. This method implements the
// model.ZeroCheckable contract through model.Model.
//
// A zero CommitRange has both From.IsZero() and To.IsZero() returning true,
// indicating that no commit range has been specified. This is distinct from
// a CommitRange with only From being zero (which represents "from the beginning
// of history to To"), where IsZero() returns false because To is non-zero.
//
// Zero-value CommitRanges are valid at the Go type level but will fail
// validation if Validate() is called, as a valid commit range requires at
// least a non-zero To boundary.
//
// This method MUST NOT mutate the CommitRange receiver and is safe for
// concurrent use from multiple goroutines.
//
// Example usage:
//
//	var cr git.CommitRange
//	if cr.IsZero() {
//	    fmt.Println("No commit range specified")
//	}
//
//	cr = git.CommitRange{
//	    From: git.Ref{},  // Zero
//	    To:   git.Ref{Name: "HEAD", Hash: "abc123..."},
//	}
//	if !cr.IsZero() {
//	    fmt.Println("Commit range from beginning to HEAD")
//	}
func (cr CommitRange) IsZero() bool {
	return cr.From.IsZero() && cr.To.IsZero()
}

// Equal reports whether the CommitRange is semantically equal to another
// CommitRange by comparing the From and To boundary refs for equality using
// their respective Equal methods. This method implements the model.Model
// interface contract.
//
// Two CommitRanges are considered equal if and only if their From refs are
// equal (via Ref.Equal) and their To refs are equal (via Ref.Equal). This
// comparison includes all fields of each Ref (Name, Kind, Hash), ensuring
// that both the symbolic names and concrete hashes match exactly.
//
// Equal is strict about representation: two CommitRanges with the same
// underlying hash boundaries but different symbolic names (e.g., "v1.0.0" vs
// "refs/tags/v1.0.0") are NOT considered equal. This preserves the metadata
// about how the range was originally specified.
//
// The other parameter must be a CommitRange or *CommitRange. If other is a
// different type, Equal returns false without panicking.
//
// This method MUST NOT mutate the CommitRange receiver and is safe for
// concurrent use from multiple goroutines.
//
// Example usage:
//
//	cr1 := git.CommitRange{From: fromRef, To: toRef}
//	cr2 := git.CommitRange{From: fromRef, To: toRef}
//	if cr1.Equal(cr2) {
//	    fmt.Println("Ranges are equal")
//	}
//
//	cr3 := git.CommitRange{From: fromRef, To: differentToRef}
//	if !cr1.Equal(cr3) {
//	    fmt.Println("Ranges differ in To boundary")
//	}
func (cr CommitRange) Equal(other any) bool {
	switch o := other.(type) {
	case CommitRange:
		return cr.From.Equal(o.From) && cr.To.Equal(o.To)
	case *CommitRange:
		if o == nil {
			return false
		}
		return cr.From.Equal(o.From) && cr.To.Equal(o.To)
	default:
		return false
	}
}

// Validate checks that the CommitRange satisfies all structural and semantic
// constraints, returning nil if valid or an error describing the first
// validation failure encountered. This method implements the model.Validatable
// contract through model.Model.
//
// Validation rules for CommitRange:
//   - The zero value (both From and To are zero) is INVALID. A valid commit
//     range requires at least a non-zero To boundary.
//   - To MUST be non-zero (To.Hash != ""). The upper bound is required.
//   - From MAY be zero (From.Hash == ""), which means "from the beginning".
//   - If From is non-zero, it MUST pass Ref.Validate().
//   - To MUST pass Ref.Validate() (required non-zero Ref).
//
// Validate does NOT check whether the From hash is an ancestor of the To hash,
// as that would require Git repository access. Callers that need ancestry
// validation SHOULD use Git commands like "git merge-base --is-ancestor" or
// equivalent Git operations.
//
// This method MUST NOT mutate the CommitRange receiver and is safe for
// concurrent use from multiple goroutines.
//
// Example usage:
//
//	cr := git.CommitRange{From: fromRef, To: toRef}
//	if err := cr.Validate(); err != nil {
//	    return fmt.Errorf("invalid commit range: %w", err)
//	}
//	// cr is now known to be structurally valid
//
//	// Zero To is invalid
//	cr = git.CommitRange{From: fromRef, To: git.Ref{}}
//	err := cr.Validate()  // Returns error about zero To
func (cr CommitRange) Validate() error {
	// Zero value is invalid - at minimum need a To boundary
	if cr.IsZero() {
		return fmt.Errorf("CommitRange is zero (both From and To are zero)")
	}

	// To MUST be non-zero
	if cr.To.IsZero() {
		return fmt.Errorf("CommitRange To is zero (To boundary is required)")
	}

	// Validate To (required)
	if err := cr.To.Validate(); err != nil {
		return fmt.Errorf("invalid CommitRange To: %w", err)
	}

	// Validate From if non-zero (zero From is allowed = from beginning)
	if !cr.From.IsZero() {
		if err := cr.From.Validate(); err != nil {
			return fmt.Errorf("invalid CommitRange From: %w", err)
		}
	}

	return nil
}

// MarshalJSON serializes the CommitRange to JSON format after validating that
// it represents a valid commit range. This method implements the json.Marshaler
// interface and is part of the model.Serializable contract through model.Model.
//
// MarshalJSON validates the CommitRange before marshaling to ensure that only
// valid commit ranges can be serialized. If validation fails, an error is
// returned and no JSON is produced. This prevents invalid data from being
// written to storage or transmitted over APIs.
//
// The JSON structure uses a type alias (commitRangeJSON) to delegate to the
// standard JSON encoder while avoiding infinite recursion. The resulting JSON
// has the structure:
//
//	{
//	  "from": {<Ref JSON>},
//	  "to": {<Ref JSON>}
//	}
//
// This method MUST NOT mutate the CommitRange receiver and is safe for
// concurrent use from multiple goroutines.
//
// Example usage:
//
//	cr := git.CommitRange{From: fromRef, To: toRef}
//	data, err := json.Marshal(cr)
//	if err != nil {
//	    return fmt.Errorf("marshal failed: %w", err)
//	}
//	// data contains: {"from":{...},"to":{...}}
//
// Returns an error if validation fails or if JSON encoding fails.
func (cr CommitRange) MarshalJSON() ([]byte, error) {
	// Validate before marshaling
	if err := cr.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid CommitRange: %w", err)
	}

	// Use type alias to avoid infinite recursion
	type commitRangeJSON CommitRange
	return json.Marshal(commitRangeJSON(cr))
}

// UnmarshalJSON deserializes a CommitRange from JSON format and validates the
// result to ensure structural and semantic correctness. This method implements
// the json.Unmarshaler interface and is part of the model.Serializable contract
// through model.Model.
//
// UnmarshalJSON first decodes the JSON into the CommitRange structure using
// a type alias (commitRangeJSON) to avoid infinite recursion, then validates
// the decoded value using Validate(). If the JSON is malformed or if the
// decoded CommitRange is invalid, an error is returned.
//
// This two-phase approach (decode then validate) ensures that only valid
// CommitRange values can be constructed from JSON, preventing invalid data
// from entering the system through deserialization.
//
// This method mutates the CommitRange receiver by decoding into it. It is NOT
// safe for concurrent use with other operations on the same CommitRange value.
//
// Example usage:
//
//	var cr git.CommitRange
//	err := json.Unmarshal(data, &cr)
//	if err != nil {
//	    return fmt.Errorf("unmarshal failed: %w", err)
//	}
//	// cr is now a valid CommitRange decoded from JSON
//
// Returns an error if JSON decoding fails or if validation fails.
func (cr *CommitRange) UnmarshalJSON(data []byte) error {
	// Use type alias to avoid infinite recursion
	type commitRangeJSON CommitRange
	var temp commitRangeJSON

	if err := json.Unmarshal(data, &temp); err != nil {
		return fmt.Errorf("failed to unmarshal CommitRange: %w", err)
	}

	*cr = CommitRange(temp)

	// Validate after unmarshaling
	if err := cr.Validate(); err != nil {
		return fmt.Errorf("invalid CommitRange after unmarshal: %w", err)
	}

	return nil
}

// MarshalYAML serializes the CommitRange to YAML format after validating that
// it represents a valid commit range. This method implements the yaml.Marshaler
// interface and is part of the model.Serializable contract through model.Model.
//
// MarshalYAML validates the CommitRange before marshaling to ensure that only
// valid commit ranges can be serialized. If validation fails, an error is
// returned and no YAML is produced. This prevents invalid data from being
// written to configuration files or other YAML-based storage.
//
// The YAML structure uses a type alias (commitRangeYAML) to delegate to the
// standard YAML encoder while avoiding infinite recursion. The resulting YAML
// has the structure:
//
//	from:
//	  name: v1.0.0
//	  kind: tag
//	  hash: abc123...
//	to:
//	  name: v2.0.0
//	  kind: tag
//	  hash: def456...
//
// This method MUST NOT mutate the CommitRange receiver and is safe for
// concurrent use from multiple goroutines.
//
// Example usage:
//
//	cr := git.CommitRange{From: fromRef, To: toRef}
//	data, err := yaml.Marshal(cr)
//	if err != nil {
//	    return fmt.Errorf("marshal failed: %w", err)
//	}
//
// Returns an error if validation fails or if YAML encoding fails.
func (cr CommitRange) MarshalYAML() (interface{}, error) {
	// Validate before marshaling
	if err := cr.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid CommitRange: %w", err)
	}

	// Use type alias to avoid infinite recursion
	type commitRangeYAML CommitRange
	return commitRangeYAML(cr), nil
}

// UnmarshalYAML deserializes a CommitRange from YAML format and validates the
// result to ensure structural and semantic correctness. This method implements
// the yaml.Unmarshaler interface and is part of the model.Serializable contract
// through model.Model.
//
// UnmarshalYAML first decodes the YAML into the CommitRange structure using
// a type alias (commitRangeYAML) to avoid infinite recursion, then validates
// the decoded value using Validate(). If the YAML is malformed or if the
// decoded CommitRange is invalid, an error is returned.
//
// This two-phase approach (decode then validate) ensures that only valid
// CommitRange values can be constructed from YAML, preventing invalid data
// from entering the system through configuration files or other YAML sources.
//
// This method mutates the CommitRange receiver by decoding into it. It is NOT
// safe for concurrent use with other operations on the same CommitRange value.
//
// Example usage:
//
//	var cr git.CommitRange
//	err := yaml.Unmarshal(data, &cr)
//	if err != nil {
//	    return fmt.Errorf("unmarshal failed: %w", err)
//	}
//	// cr is now a valid CommitRange decoded from YAML
//
// Returns an error if YAML decoding fails or if validation fails.
func (cr *CommitRange) UnmarshalYAML(node *yaml.Node) error {
	// Use type alias to avoid infinite recursion
	type commitRangeYAML CommitRange
	var temp commitRangeYAML

	if err := node.Decode(&temp); err != nil {
		return fmt.Errorf("failed to unmarshal CommitRange: %w", err)
	}

	*cr = CommitRange(temp)

	// Validate after unmarshaling
	if err := cr.Validate(); err != nil {
		return fmt.Errorf("invalid CommitRange after unmarshal: %w", err)
	}

	return nil
}
