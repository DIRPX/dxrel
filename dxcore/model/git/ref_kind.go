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
	"strings"

	"dirpx.dev/dxrel/dxcore/model"
	"gopkg.in/yaml.v3"
)

// RefKind describes the coarse category of a Git reference or revision
// expression, classifying it by its structural form and namespace within
// the Git repository.
//
// This type implements the model.Model interface, providing validation,
// serialization to JSON and YAML, safe logging, type identification, and
// zero-value detection. The zero value of RefKind (RefKindUnknown) is valid
// and represents "kind not determined" or "kind not applicable", indicating
// that a reference has not been classified or the classification is unknown.
//
// RefKind is typically determined by examining the structure of a RefName
// value, such as checking for the "refs/heads/" prefix for branches or
// validating SHA-1/SHA-256 hash format for commit object ids. The classification
// is coarse and structural; semantic information (such as whether a branch is
// active or a tag is signed) is not captured by RefKind.
//
// JSON and YAML serialization uses string representations ("branch", "tag",
// etc.) rather than numeric values to ensure human readability and forward
// compatibility when new kinds are added in future versions of dxrel.
//
// Example values:
//   - RefKindBranch: refs/heads/main, refs/heads/feature/new-thing
//   - RefKindRemoteBranch: refs/remotes/origin/main, refs/remotes/upstream/develop
//   - RefKindTag: refs/tags/v1.0.0, refs/tags/release-2023-01-01
//   - RefKindHead: HEAD, FETCH_HEAD, ORIG_HEAD, MERGE_HEAD
//   - RefKindHash: a1b2c3d4e5f67890abcdef1234567890abcdef12 (40 or 64 hex chars)
//   - RefKindUnknown: unclassified or ambiguous reference
type RefKind uint8

const (
	// RefKindUnknown represents an unknown, unclassified, or ambiguous
	// reference kind.
	//
	// This is the zero value for RefKind. It is valid and MAY be used in
	// data structures where the kind of a ref has not yet been determined
	// or is not applicable. For example, a RefName that does not match any
	// known structural pattern would be classified as RefKindUnknown.
	RefKindUnknown RefKind = iota

	// RefKindBranch represents a local branch reference in the refs/heads/
	// namespace.
	//
	// Example: refs/heads/main, refs/heads/feature/new-ui
	RefKindBranch

	// RefKindRemoteBranch represents a remote-tracking branch reference in
	// the refs/remotes/ namespace.
	//
	// Example: refs/remotes/origin/main, refs/remotes/upstream/develop
	RefKindRemoteBranch

	// RefKindTag represents a tag reference in the refs/tags/ namespace.
	//
	// Example: refs/tags/v1.0.0, refs/tags/release-2023-01-01
	RefKindTag

	// RefKindHead represents a special symbolic reference such as HEAD,
	// FETCH_HEAD, ORIG_HEAD, or MERGE_HEAD.
	//
	// These refs point to commits or other refs and are used by Git to
	// track the current branch, fetch results, merge state, and other
	// transient repository information.
	//
	// Example: HEAD, FETCH_HEAD, ORIG_HEAD, MERGE_HEAD
	RefKindHead

	// RefKindHash represents an explicit full commit object id (SHA-1 or
	// SHA-256 hash).
	//
	// A RefName is classified as RefKindHash if it consists of exactly 40
	// lowercase hexadecimal characters (SHA-1) or exactly 64 lowercase
	// hexadecimal characters (SHA-256). Abbreviated hashes are NOT
	// classified as RefKindHash; they remain RefKindUnknown because they
	// require resolution via Git commands.
	//
	// Example: a1b2c3d4e5f67890abcdef1234567890abcdef12
	RefKindHash
)

const (
	// RefKindUnknownStr is the string representation of RefKindUnknown.
	RefKindUnknownStr = "unknown"

	// RefKindBranchStr is the string representation of RefKindBranch.
	RefKindBranchStr = "branch"

	// RefKindRemoteBranchStr is the canonical string representation of
	// RefKindRemoteBranch.
	//
	// Alternative formats "remote_branch" and "remotebranch" are also
	// accepted during parsing for compatibility, but this canonical form
	// with hyphen is used for serialization.
	RefKindRemoteBranchStr = "remote-branch"

	// RefKindTagStr is the string representation of RefKindTag.
	RefKindTagStr = "tag"

	// RefKindHeadStr is the string representation of RefKindHead.
	RefKindHeadStr = "head"

	// RefKindHashStr is the string representation of RefKindHash.
	RefKindHashStr = "hash"
)

// ParseRefKind parses a string into a validated RefKind value.
//
// ParseRefKind applies normalization and validation to the input string.
// The normalization process trims leading and trailing whitespace via
// strings.TrimSpace and converts the string to lowercase via strings.ToLower.
// After normalization, the string is matched against the known kind names:
// "unknown", "branch", "remote-branch", "tag", "head", or "hash".
//
// If the normalized input matches a known kind name, the corresponding
// RefKind constant is returned. If the input does not match any known
// name, ParseRefKind returns RefKindUnknown and an error describing the
// parsing failure.
//
// Example usage:
//
//	kind, err := git.ParseRefKind("branch")
//	// kind = RefKindBranch, err = nil
//
//	kind, err := git.ParseRefKind("  TAG  ")
//	// kind = RefKindTag, err = nil
//
//	kind, err := git.ParseRefKind("unknown")
//	// kind = RefKindUnknown, err = nil
//
//	kind, err := git.ParseRefKind("invalid")
//	// kind = RefKindUnknown, err = error
func ParseRefKind(s string) (RefKind, error) {
	// Normalize: trim whitespace and convert to lowercase
	normalized := strings.ToLower(strings.TrimSpace(s))

	switch normalized {
	case RefKindUnknownStr:
		return RefKindUnknown, nil
	case RefKindBranchStr:
		return RefKindBranch, nil
	case RefKindRemoteBranchStr, "remote_branch", "remotebranch":
		return RefKindRemoteBranch, nil
	case RefKindTagStr:
		return RefKindTag, nil
	case RefKindHeadStr:
		return RefKindHead, nil
	case RefKindHashStr:
		return RefKindHash, nil
	default:
		return RefKindUnknown, fmt.Errorf("unknown RefKind name %q (valid: %s, %s, %s, %s, %s, %s)", s,
			RefKindUnknownStr, RefKindBranchStr, RefKindRemoteBranchStr, RefKindTagStr, RefKindHeadStr, RefKindHashStr)
	}
}

// Compile-time assertion that RefKind implements model.Model.
var _ model.Model = (*RefKind)(nil)

// String returns the RefKind value as a string for display or logging.
//
// This method implements the fmt.Stringer interface and the model.Loggable
// contract. For a RefKind, String returns a human-readable lowercase name
// describing the kind of reference: "unknown", "branch", "remote-branch",
// "tag", "head", or "hash".
//
// Example output:
//   - RefKindUnknown.String() -> "unknown"
//   - RefKindBranch.String() -> "branch"
//   - RefKindRemoteBranch.String() -> "remote-branch"
//   - RefKindTag.String() -> "tag"
//   - RefKindHead.String() -> "head"
//   - RefKindHash.String() -> "hash"
func (rk RefKind) String() string {
	switch rk {
	case RefKindUnknown:
		return RefKindUnknownStr
	case RefKindBranch:
		return RefKindBranchStr
	case RefKindRemoteBranch:
		return RefKindRemoteBranchStr
	case RefKindTag:
		return RefKindTagStr
	case RefKindHead:
		return RefKindHeadStr
	case RefKindHash:
		return RefKindHashStr
	default:
		return fmt.Sprintf("RefKind(%d)", uint8(rk))
	}
}

// Redacted returns a redacted form of the RefKind suitable for logging
// in security-sensitive contexts.
//
// This method implements the model.Loggable contract. For RefKind, the
// redacted form is identical to the full value, as reference kind
// classifications are not considered sensitive information. The String()
// method is called to produce the human-readable name.
func (rk RefKind) Redacted() string {
	return rk.String()
}

// TypeName returns the name of this type for error messages, logging,
// and debugging.
//
// This method implements the model.Identifiable contract. It always
// returns "RefKind", which is used by higher-level error handling and
// serialization code to construct clear diagnostic messages.
func (rk RefKind) TypeName() string {
	return "RefKind"
}

// IsZero reports whether this RefKind is the zero value.
//
// This method implements the model.ZeroCheckable contract. A RefKind is
// considered zero when it equals RefKindUnknown, representing "kind not
// determined" or "kind not applicable". The zero value is valid and MAY
// appear in data structures where a ref kind has not been classified.
//
// Example usage:
//
//	var kind git.RefKind
//	if kind.IsZero() {
//	    // Handle unclassified ref kind
//	}
func (rk RefKind) IsZero() bool {
	return rk == RefKindUnknown
}

// Equal reports whether this RefKind is equal to another RefKind value.
//
// Two RefKind values are equal if and only if they have the same numeric
// value. This method uses direct integer comparison on the underlying uint8
// representation.
//
// Example usage:
//
//	kind1 := git.RefKindBranch
//	kind2 := git.RefKindBranch
//	kind3 := git.RefKindTag
//	kind1.Equal(kind2) // returns true
//	kind1.Equal(kind3) // returns false
func (rk RefKind) Equal(other RefKind) bool {
	return rk == other
}

// Validate checks whether this RefKind is a known, valid kind value.
//
// This method implements the model.Validatable contract. Validate returns
// nil if the RefKind is one of the defined constants (RefKindUnknown,
// RefKindBranch, RefKindRemoteBranch, RefKindTag, RefKindHead, RefKindHash),
// or an error if the numeric value falls outside the known range.
//
// Validation ensures that RefKind values used in dxrel are limited to the
// explicitly defined set, preventing invalid or corrupted data from
// propagating through serialization and business logic.
//
// Example usage:
//
//	kind := git.RefKindBranch
//	if err := kind.Validate(); err != nil {
//	    return fmt.Errorf("invalid ref kind: %w", err)
//	}
func (rk RefKind) Validate() error {
	switch rk {
	case RefKindUnknown, RefKindBranch, RefKindRemoteBranch, RefKindTag, RefKindHead, RefKindHash:
		return nil
	default:
		return fmt.Errorf("RefKind value %d is not a known kind (valid range: 0-%d)", uint8(rk), uint8(RefKindHash))
	}
}

// MarshalJSON serializes this RefKind to JSON.
//
// This method implements the json.Marshaler interface and the
// model.Serializable contract. The RefKind is validated before marshaling
// to ensure only well-formed values are written to JSON. If validation
// fails, MarshalJSON returns an error.
//
// The RefKind is encoded as a JSON string containing the human-readable
// name returned by String(): "unknown", "branch", "remote-branch", "tag",
// "head", or "hash". String encoding (rather than numeric) provides better
// readability and forward compatibility when new kinds are added.
//
// Example output:
//
//	{"kind": "branch"}
//	{"kind": "tag"}
//	{"kind": "unknown"}
func (rk RefKind) MarshalJSON() ([]byte, error) {
	if err := rk.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", rk.TypeName(), err)
	}
	return json.Marshal(rk.String())
}

// UnmarshalJSON deserializes a RefKind from JSON.
//
// This method implements the json.Unmarshaler interface and the
// model.Serializable contract. The input JSON MUST be a string value
// containing one of the valid kind names: "unknown", "branch",
// "remote-branch", "tag", "head", or "hash". UnmarshalJSON applies
// normalization (strings.TrimSpace, strings.ToLower) and then validates
// the result. If parsing or validation fails, the RefKind is not modified
// and an error is returned.
//
// Example input:
//
//	{"kind": "branch"}
//	{"kind": "BRANCH"}  // normalized to "branch"
//	{"kind": "  tag  "}  // normalized to "tag"
//	{"kind": "unknown"}
func (rk *RefKind) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return fmt.Errorf("cannot unmarshal JSON into %s: %w", rk.TypeName(), err)
	}

	parsed, err := ParseRefKind(str)
	if err != nil {
		return fmt.Errorf("unmarshaled %s is invalid: %w", rk.TypeName(), err)
	}

	*rk = parsed
	return nil
}

// MarshalYAML serializes this RefKind to YAML.
//
// This method implements the yaml.Marshaler interface and the
// model.Serializable contract. The RefKind is validated before marshaling
// to ensure only well-formed values are written to YAML. If validation
// fails, MarshalYAML returns an error.
//
// The RefKind is encoded as a YAML string containing the human-readable
// name returned by String(): "unknown", "branch", "remote-branch", "tag",
// "head", or "hash".
//
// A type alias is used internally to prevent infinite recursion during
// marshaling.
//
// Example output:
//
//	kind: branch
//	kind: tag
//	kind: unknown
func (rk RefKind) MarshalYAML() (interface{}, error) {
	if err := rk.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", rk.TypeName(), err)
	}
	return rk.String(), nil
}

// UnmarshalYAML deserializes a RefKind from YAML.
//
// This method implements the yaml.Unmarshaler interface and the
// model.Serializable contract. The input YAML MUST be a string value
// containing one of the valid kind names. UnmarshalYAML applies
// normalization (strings.TrimSpace, strings.ToLower) and then validates
// the result. If parsing or validation fails, the RefKind is not modified
// and an error is returned.
//
// Example input:
//
//	kind: branch
//	kind: "  TAG  "  # normalized to "tag"
//	kind: unknown
func (rk *RefKind) UnmarshalYAML(node *yaml.Node) error {
	var str string
	if err := node.Decode(&str); err != nil {
		return fmt.Errorf("cannot unmarshal YAML into %s: %w", rk.TypeName(), err)
	}

	parsed, err := ParseRefKind(str)
	if err != nil {
		return fmt.Errorf("unmarshaled %s is invalid: %w", rk.TypeName(), err)
	}

	*rk = parsed
	return nil
}
