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

	"dirpx.dev/dxrel/dxcore/errors"
	"dirpx.dev/dxrel/dxcore/model"
	"gopkg.in/yaml.v3"
)

const (
	// CommitMessageMaxLen is the maximum allowed length for a commit message,
	// measured in bytes.
	//
	// This limit prevents abuse and ensures that commit messages remain
	// reasonable in size. A limit of 1MB (1048576 bytes) provides ample space
	// for detailed commit messages, including extended descriptions, issue
	// references, and other metadata, while preventing excessively large
	// messages that could impact performance or storage.
	//
	// Most Git commit messages are much shorter than this limit. The Linux
	// kernel recommends 50 characters for the subject line and 72 characters
	// for body lines, but dxrel does not enforce these conventions at the
	// model layer, leaving such formatting decisions to higher layers.
	CommitMessageMaxLen = 1048576 // 1MB

	// CommitSummaryMaxLen is the maximum allowed length for a commit summary
	// (the first line of the commit message), measured in bytes.
	//
	// This limit ensures that commit summaries remain concise and readable.
	// A limit of 512 bytes accommodates most common summary formats including
	// Conventional Commits headers with long scopes and subjects, while
	// preventing abuse.
	//
	// The Git community generally recommends keeping summaries under 50-72
	// characters for readability in git log output, but dxrel allows longer
	// summaries to support various commit message conventions used in
	// different projects.
	CommitSummaryMaxLen = 512

	// CommitParentsMaxCount is the maximum number of parent commits allowed
	// for a single commit.
	//
	// While Git theoretically supports unlimited parents for octopus merges,
	// practical considerations make very large numbers of parents suspicious
	// and potentially indicative of malformed data or abuse. A limit of 64
	// parents accommodates even the most complex octopus merges while
	// preventing pathological cases.
	//
	// Most commits have 1 parent (normal commit) or 2 parents (merge commit).
	// Octopus merges with 3+ parents are rare in practice.
	CommitParentsMaxCount = 64

	// CommitChangesMaxCount is the maximum number of file changes allowed
	// for a single commit.
	//
	// While some commits may legitimately touch many files (e.g., mass
	// refactorings, dependency updates, code generation), extremely large
	// numbers of changes can impact performance and may indicate data issues.
	// A limit of 10,000 files accommodates even very large commits while
	// preventing pathological cases.
	//
	// Most commits touch fewer than 10 files. Commits touching hundreds or
	// thousands of files are uncommon but valid in certain scenarios (e.g.,
	// merging vendored dependencies, reformatting entire codebase).
	CommitChangesMaxCount = 10000
)

// Commit represents a Git commit object as seen by dxrel's git layer,
// containing all metadata needed for commit analysis, classification, and
// reporting.
//
// This type implements the model.Model interface, providing validation,
// serialization to JSON and YAML, safe logging, type identification, and
// zero-value detection.
//
// A Commit provides enough information for higher layers to:
//   - Classify commits by Conventional Commits rules (using Message/Summary)
//   - Map commits to modules by changed paths (using Changes)
//   - Render summaries and ranges in logs and reports (using all fields)
//   - Track authorship and timing information (using Author/Committer)
//   - Understand commit graph structure (using Hash/Parents)
//
// The zero value of Commit (all fields zero) is valid at the Go type level
// but represents "no commit specified" and will fail validation if Validate()
// is called.
//
// Example values:
//
//	// Normal commit
//	Commit{
//	    Hash:      "a1b2c3d4e5f67890abcdef1234567890abcdef12",
//	    Parents:   []Hash{"parent1234567890abcdef1234567890abcdef12"},
//	    Author:    Signature{Name: "Jane", Email: "jane@example.com", When: time.Now()},
//	    Committer: Signature{Name: "Jane", Email: "jane@example.com", When: time.Now()},
//	    Message:   "feat: add new feature\n\nDetailed description here.",
//	    Summary:   "feat: add new feature",
//	    Changes:   []FileChange{{Path: "src/feature.go", Kind: FileChangeAdded}},
//	}
//
//	// Merge commit
//	Commit{
//	    Hash:    "merge123456789abcdef1234567890abcdef1234",
//	    Parents: []Hash{"parent1...", "parent2..."},
//	    // ... other fields
//	}
type Commit struct {
	// Hash is the commit object id.
	//
	// This uniquely identifies the commit within the Git repository. The
	// hash MUST be a fully resolved Git object id in canonical form
	// (40-character SHA-1 or 64-character SHA-256 in lowercase hex).
	//
	// The Hash field MUST NOT be empty for a valid Commit.
	Hash Hash `json:"hash" yaml:"hash"`

	// Parents lists the parent commits in order.
	//
	// For a normal commit this typically has 1 element (the previous commit
	// in the branch history). For merge commits it has 2 or more parents
	// (the merged branches). For the initial commit in a repository, this
	// slice is empty.
	//
	// The order of parents matters: the first parent is the branch being
	// merged into, subsequent parents are the branches being merged.
	//
	// Each parent hash MUST be a valid Hash (validated by Hash.Validate()).
	// Maximum number of parents is CommitParentsMaxCount (64).
	Parents []Hash `json:"parents" yaml:"parents"`

	// Author is the author signature recorded in the commit.
	//
	// This represents the person who originally wrote the code, as recorded
	// in the "author" field of the Git commit object. The author signature
	// includes the author's name, email, and timestamp.
	//
	// The Author field MUST NOT be zero for a valid Commit.
	Author Signature `json:"author" yaml:"author"`

	// Committer is the committer signature recorded in the commit.
	//
	// This represents the person who created the commit object, as recorded
	// in the "committer" field of the Git commit object. Often the committer
	// is the same as the author, but they may differ when commits are applied
	// by someone else (e.g., maintainers applying patches, rebasing, or
	// cherry-picking).
	//
	// The Committer field MUST NOT be zero for a valid Commit.
	Committer Signature `json:"committer" yaml:"committer"`

	// Message is the full raw commit message as stored in Git.
	//
	// The message MUST use '\n' (LF) line endings. Parsers are expected to
	// normalize CRLF or lone '\r' sequences at read time. The message MAY
	// contain multiple paragraphs separated by blank lines, following common
	// Git commit message conventions.
	//
	// The first line of Message is the commit summary/subject, which SHOULD
	// be a concise description of the change. Subsequent lines (separated
	// from the summary by a blank line) form the commit body, which MAY
	// contain detailed explanations, issue references, and other metadata.
	//
	// The Message field MUST NOT be empty for a valid Commit. Maximum
	// length is CommitMessageMaxLen (1MB).
	//
	// Example:
	//
	//	"feat(auth): add OAuth2 support\n\nImplements OAuth2 authentication flow.\n\nFixes #123"
	Message string `json:"message" yaml:"message"`

	// Summary is the first line of the commit message, trimmed of trailing
	// newline characters.
	//
	// This is provided as a convenience field for displaying commit
	// summaries in logs, UIs, and reports without needing to parse Message.
	// The Summary is always derivable from Message by extracting the first
	// line and trimming trailing whitespace.
	//
	// For valid commits, Summary MUST equal the first line of Message
	// (trimmed). The Summary field MUST NOT be empty for a valid Commit.
	// Maximum length is CommitSummaryMaxLen (512 bytes).
	//
	// Example:
	//
	//	Message:  "feat: add feature\n\nBody text here"
	//	Summary:  "feat: add feature"
	Summary string `json:"summary" yaml:"summary"`

	// Changes lists the file changes affected by this commit.
	//
	// Each FileChange describes a file that was added, modified, deleted,
	// renamed, copied, or type-changed in this commit. Paths are relative
	// to the repository root.
	//
	// The list SHOULD NOT contain duplicates (same path with same change kind).
	// For renames and copies, the FileChange includes both OldPath and Path.
	//
	// dxrel uses this information to map commits to modules by matching
	// module directories against changed paths. For example, a commit that
	// changes "moduleA/src/file.go" would be associated with "moduleA".
	//
	// The Changes slice MAY be empty for commits that only change Git
	// metadata (e.g., empty commits created with --allow-empty). Maximum
	// number of changes is CommitChangesMaxCount (10,000).
	Changes []FileChange `json:"changes" yaml:"changes"`
}

// Compile-time assertion that Commit implements model.Model.
var _ model.Model = (*Commit)(nil)

// NewCommit creates a new Commit with the given components, validating the
// result before returning.
//
// This is a convenience constructor that creates and validates a Commit in
// one step, ensuring that all components conform to the validation rules
// defined by Commit.Validate. If any of the components are invalid or the
// combination violates Commit invariants, NewCommit returns a zero Commit
// and an error describing the validation failure.
//
// The summary parameter should be the first line of the message (without
// trailing newline). If summary is empty, it will be automatically extracted
// from message. If summary is provided, it MUST match the first line of
// message (after trimming).
//
// This function is pure and has no side effects. It is safe to call
// concurrently from multiple goroutines.
//
// Example usage:
//
//	// Simple commit
//	commit, err := git.NewCommit(
//	    hash,
//	    []git.Hash{parentHash},
//	    author,
//	    committer,
//	    "feat: add feature\n\nDetailed description.",
//	    "feat: add feature",
//	    []git.FileChange{{Path: "src/feature.go", Kind: git.FileChangeAdded}},
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Merge commit
//	commit, err := git.NewCommit(
//	    mergeHash,
//	    []git.Hash{parent1, parent2},
//	    author,
//	    committer,
//	    "Merge branch 'feature' into main",
//	    "Merge branch 'feature' into main",
//	    changes,
//	)
func NewCommit(hash Hash, parents []Hash, author, committer Signature, message, summary string, changes []FileChange) (Commit, error) {
	// Auto-extract summary if not provided
	if summary == "" && message != "" {
		lines := strings.Split(message, "\n")
		if len(lines) > 0 {
			summary = strings.TrimSpace(lines[0])
		}
	}

	commit := Commit{
		Hash:      hash,
		Parents:   parents,
		Author:    author,
		Committer: committer,
		Message:   message,
		Summary:   summary,
		Changes:   changes,
	}

	if err := commit.Validate(); err != nil {
		return Commit{}, err
	}

	return commit, nil
}

// String returns the human-readable representation of the Commit for display
// and debugging purposes. This method implements the fmt.Stringer interface
// and satisfies the model.Loggable contract's String() requirement.
//
// The output format includes key commit metadata:
//
//	Commit{Hash:<hash>, Parents:<count>, Author:<name>, Summary:<summary>}
//
// The Message, Committer, and Changes fields are omitted from String() output
// for brevity, as they can be very long. Callers needing these fields should
// access them directly.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently.
//
// Example:
//
//	commit := git.Commit{Hash: hash, Summary: "feat: add feature", ...}
//	fmt.Println(commit.String())
//	// Output: Commit{Hash:a1b2c3d, Parents:1, Author:Jane, Summary:feat: add feature}
func (c Commit) String() string {
	return fmt.Sprintf("Commit{Hash:%s, Parents:%d, Author:%s, Summary:%s}",
		c.Hash.String(),
		len(c.Parents),
		c.Author.Name,
		c.Summary)
}

// Redacted returns a safe string representation suitable for logging in
// production environments. This method satisfies the model.Loggable
// interface's Redacted requirement.
//
// For Commit, the redacted output includes Hash (abbreviated), parent count,
// author name (not email, to preserve some privacy), and summary. The
// implementation delegates to the Redacted() methods of component types
// (Hash, which abbreviates to 7 characters).
//
// Email addresses from Author and Committer are omitted to protect privacy.
// The full Message is omitted as it may contain sensitive information.
// Changes are omitted for brevity.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently.
//
// Example:
//
//	commit := git.Commit{Hash: hash, Summary: "feat: add feature", ...}
//	log.Info("processing commit", "commit", commit.Redacted())
//	// Output (in logs): Commit{Hash:a1b2c3d, Parents:1, Author:Jane, Summary:feat: add feature}
func (c Commit) Redacted() string {
	return fmt.Sprintf("Commit{Hash:%s, Parents:%d, Author:%s, Summary:%s}",
		c.Hash.Redacted(),
		len(c.Parents),
		c.Author.Name,
		c.Summary)
}

// TypeName returns the canonical name of this model type for debugging,
// logging, and reflection purposes. This method satisfies the model.Identifiable
// interface's TypeName requirement.
//
// The returned value is always the constant string "Commit", uniquely
// identifying this type within the dxrel domain. The name follows CamelCase
// convention and omits the package prefix as required by the Model contract.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently. The returned string is a constant literal,
// ensuring zero allocations.
func (c Commit) TypeName() string {
	return "Commit"
}

// IsZero reports whether this Commit instance is in a zero or empty state,
// meaning no commit has been specified. For Commit, the zero value represents
// "no commit specified" or "commit not initialized".
//
// This method satisfies the model.ZeroCheckable interface's IsZero requirement.
// A Commit is considered zero if all fields are their zero values: empty Hash,
// nil/empty Parents, zero Author and Committer, empty Message and Summary, and
// nil/empty Changes.
//
// Zero-value Commits are semantically invalid for most operations and will
// fail validation if Validate() is called. However, the zero value is useful
// as a sentinel for "no commit specified" in optional fields or when
// initializing data structures.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently.
//
// Example:
//
//	var commit git.Commit // Zero value
//	fmt.Println(commit.IsZero()) // Output: true
//
//	commit = git.Commit{Hash: hash, Summary: "feat: add", ...}
//	fmt.Println(commit.IsZero()) // Output: false
func (c Commit) IsZero() bool {
	return c.Hash.IsZero() &&
		len(c.Parents) == 0 &&
		c.Author.IsZero() &&
		c.Committer.IsZero() &&
		c.Message == "" &&
		c.Summary == "" &&
		len(c.Changes) == 0
}

// Equal reports whether this Commit is equal to another Commit value.
//
// Two Commits are equal if and only if all components match:
//   - Hash must be equal (case-sensitive string comparison)
//   - Parents must have the same length and all hashes equal (order matters)
//   - Author must be equal (delegates to Signature.Equal)
//   - Committer must be equal (delegates to Signature.Equal)
//   - Message must be equal (case-sensitive string comparison)
//   - Summary must be equal (case-sensitive string comparison)
//   - Changes must have the same length and all changes equal (order matters)
//
// This method is particularly useful in table-driven tests, assertion libraries,
// deduplication logic, and comparison operations where a method-based approach
// is more idiomatic than manual field-by-field comparison.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently. The comparison is fast, deterministic,
// and performs no allocations beyond the standard comparisons.
//
// Example:
//
//	commit1 := git.Commit{Hash: hash1, Summary: "feat: add", ...}
//	commit2 := git.Commit{Hash: hash1, Summary: "feat: add", ...}
//	commit3 := git.Commit{Hash: hash2, Summary: "fix: bug", ...}
//	fmt.Println(commit1.Equal(commit2)) // Output: true
//	fmt.Println(commit1.Equal(commit3)) // Output: false
func (c Commit) Equal(other Commit) bool {
	// Compare scalar fields
	if !c.Hash.Equal(other.Hash) ||
		!c.Author.Equal(other.Author) ||
		!c.Committer.Equal(other.Committer) ||
		c.Message != other.Message ||
		c.Summary != other.Summary {
		return false
	}

	// Compare Parents slice
	if len(c.Parents) != len(other.Parents) {
		return false
	}
	for i := range c.Parents {
		if !c.Parents[i].Equal(other.Parents[i]) {
			return false
		}
	}

	// Compare Changes slice
	if len(c.Changes) != len(other.Changes) {
		return false
	}
	for i := range c.Changes {
		if !c.Changes[i].Equal(other.Changes[i]) {
			return false
		}
	}

	return true
}

// Validate checks whether this Commit satisfies all model contracts and
// invariants. This method implements the model.Validatable interface's
// Validate requirement, enforcing data integrity for Git commit records.
//
// Validate returns nil if the Commit conforms to all of the following
// requirements:
//
// Hash validation:
//   - Hash MUST NOT be zero/empty
//   - Hash MUST be a valid Git hash (delegates to Hash.Validate())
//
// Parents validation:
//   - Parents MAY be empty (for initial commits)
//   - Each parent hash MUST be valid (delegates to Hash.Validate())
//   - Number of parents MUST NOT exceed CommitParentsMaxCount (64)
//
// Author validation:
//   - Author MUST NOT be zero
//   - Author MUST be valid (delegates to Signature.Validate())
//
// Committer validation:
//   - Committer MUST NOT be zero
//   - Committer MUST be valid (delegates to Signature.Validate())
//
// Message validation:
//   - Message MUST NOT be empty
//   - Message length MUST NOT exceed CommitMessageMaxLen (1MB)
//   - Message MUST use LF line endings (checked by looking for CRLF or lone CR)
//
// Summary validation:
//   - Summary MUST NOT be empty
//   - Summary length MUST NOT exceed CommitSummaryMaxLen (512 bytes)
//   - Summary MUST match first line of Message (trimmed)
//   - Summary MUST NOT contain newlines
//
// Changes validation:
//   - Changes MAY be empty (for commits with no file changes)
//   - Each change MUST be valid (delegates to FileChange.Validate())
//   - Number of changes MUST NOT exceed CommitChangesMaxCount (10,000)
//
// This method MUST be fast, deterministic, and idempotent. It MUST NOT mutate
// the receiver, MUST NOT have side effects, and MUST be safe to call concurrently.
// Validation does not perform I/O or allocate memory except when constructing
// error messages for invalid values.
//
// Callers SHOULD invoke Validate after creating Commit instances from external
// sources (JSON, YAML, Git commands, user input) to ensure data integrity.
// The marshal/unmarshal methods automatically call Validate to enforce this
// contract.
//
// Example:
//
//	commit := git.Commit{Hash: hash, Summary: "feat: add", ...}
//	if err := commit.Validate(); err != nil {
//	    log.Error("invalid commit", "error", err)
//	}
func (c Commit) Validate() error {
	// Validate Hash
	if c.Hash.IsZero() {
		return &errors.ValidationError{
			Type:   c.TypeName(),
			Field:  "Hash",
			Reason: "must not be empty",
		}
	}
	if err := c.Hash.Validate(); err != nil {
		return &errors.ValidationError{
			Type:   c.TypeName(),
			Field:  "Hash",
			Reason: fmt.Sprintf("invalid: %v", err),
		}
	}

	// Validate Parents
	if len(c.Parents) > CommitParentsMaxCount {
		return &errors.ValidationError{
			Type:   c.TypeName(),
			Field:  "Parents",
			Reason: fmt.Sprintf("has too many parents: %d (maximum %d)", len(c.Parents), CommitParentsMaxCount),
		}
	}
	for i, parent := range c.Parents {
		if parent.IsZero() {
			return &errors.ValidationError{
				Type:   c.TypeName(),
				Field:  fmt.Sprintf("Parents[%d]", i),
				Reason: "must not be empty",
			}
		}
		if err := parent.Validate(); err != nil {
			return &errors.ValidationError{
				Type:   c.TypeName(),
				Field:  fmt.Sprintf("Parents[%d]", i),
				Reason: fmt.Sprintf("invalid: %v", err),
			}
		}
	}

	// Validate Author
	if c.Author.IsZero() {
		return &errors.ValidationError{
			Type:   c.TypeName(),
			Field:  "Author",
			Reason: "must not be empty",
		}
	}
	if err := c.Author.Validate(); err != nil {
		return &errors.ValidationError{
			Type:   c.TypeName(),
			Field:  "Author",
			Reason: fmt.Sprintf("invalid: %v", err),
		}
	}

	// Validate Committer
	if c.Committer.IsZero() {
		return &errors.ValidationError{
			Type:   c.TypeName(),
			Field:  "Committer",
			Reason: "must not be empty",
		}
	}
	if err := c.Committer.Validate(); err != nil {
		return &errors.ValidationError{
			Type:   c.TypeName(),
			Field:  "Committer",
			Reason: fmt.Sprintf("invalid: %v", err),
		}
	}

	// Validate Message
	if c.Message == "" {
		return &errors.ValidationError{
			Type:   c.TypeName(),
			Field:  "Message",
			Reason: "must not be empty",
		}
	}
	if len(c.Message) > CommitMessageMaxLen {
		return &errors.ValidationError{
			Type:   c.TypeName(),
			Field:  "Message",
			Reason: fmt.Sprintf("exceeds maximum length of %d bytes (got %d)", CommitMessageMaxLen, len(c.Message)),
		}
	}
	// Check for CRLF or lone CR (should be normalized to LF)
	if strings.Contains(c.Message, "\r\n") || strings.Contains(c.Message, "\r") {
		return &errors.ValidationError{
			Type:   c.TypeName(),
			Field:  "Message",
			Reason: "contains CRLF or CR line endings (must use LF)",
		}
	}

	// Validate Summary
	if c.Summary == "" {
		return &errors.ValidationError{
			Type:   c.TypeName(),
			Field:  "Summary",
			Reason: "must not be empty",
		}
	}
	if len(c.Summary) > CommitSummaryMaxLen {
		return &errors.ValidationError{
			Type:   c.TypeName(),
			Field:  "Summary",
			Reason: fmt.Sprintf("exceeds maximum length of %d bytes (got %d)", CommitSummaryMaxLen, len(c.Summary)),
		}
	}
	if strings.Contains(c.Summary, "\n") || strings.Contains(c.Summary, "\r") {
		return &errors.ValidationError{
			Type:   c.TypeName(),
			Field:  "Summary",
			Reason: "must not contain newlines",
		}
	}

	// Validate Summary matches first line of Message
	lines := strings.Split(c.Message, "\n")
	if len(lines) > 0 {
		expectedSummary := strings.TrimSpace(lines[0])
		if c.Summary != expectedSummary {
			return &errors.ValidationError{
				Type:   c.TypeName(),
				Field:  "Summary",
				Reason: fmt.Sprintf("%q does not match first line of Message %q", c.Summary, expectedSummary),
			}
		}
	}

	// Validate Changes
	if len(c.Changes) > CommitChangesMaxCount {
		return &errors.ValidationError{
			Type:   c.TypeName(),
			Field:  "Changes",
			Reason: fmt.Sprintf("has too many changes: %d (maximum %d)", len(c.Changes), CommitChangesMaxCount),
		}
	}
	for i, change := range c.Changes {
		if err := change.Validate(); err != nil {
			return &errors.ValidationError{
				Type:   c.TypeName(),
				Field:  fmt.Sprintf("Changes[%d]", i),
				Reason: fmt.Sprintf("invalid: %v", err),
			}
		}
	}

	return nil
}

// MarshalJSON implements json.Marshaler, serializing the Commit to JSON
// object format. This method satisfies part of the model.Serializable
// interface requirement.
//
// MarshalJSON first validates that the Commit conforms to all constraints
// by calling Validate. If validation fails, marshaling fails with the
// validation error, preventing invalid data from being serialized. If
// validation succeeds, the Commit is serialized to a JSON object with all
// fields.
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
//	commit := git.Commit{Hash: hash, Summary: "feat: add", ...}
//	data, _ := json.Marshal(commit)
func (c Commit) MarshalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", c.TypeName(), err)
	}

	// Use type alias to avoid infinite recursion
	type commit Commit
	return json.Marshal(commit(c))
}

// UnmarshalJSON implements json.Unmarshaler, deserializing a JSON object
// into a Commit value. This method satisfies part of the model.Serializable
// interface requirement.
//
// UnmarshalJSON accepts JSON objects with the structure defined in MarshalJSON.
// After unmarshaling the JSON data, Validate is called to ensure the resulting
// Commit conforms to all constraints. If validation fails, unmarshaling fails
// with an error describing the validation failure. This fail-fast behavior
// prevents invalid data from entering the system through external inputs.
//
// The method mutates the receiver to store the unmarshaled value. It is not
// safe for concurrent use during unmarshaling, though the resulting Commit
// value is safe for concurrent reads after unmarshaling completes.
//
// Example:
//
//	var commit git.Commit
//	json.Unmarshal(data, &commit)
//	fmt.Println(commit.Hash)
func (c *Commit) UnmarshalJSON(data []byte) error {
	type commit Commit
	if err := json.Unmarshal(data, (*commit)(c)); err != nil {
		return &errors.UnmarshalError{
			Type:   c.TypeName(),
			Data:   data,
			Reason: err.Error(),
		}
	}

	if err := c.Validate(); err != nil {
		return &errors.UnmarshalError{
			Type:   c.TypeName(),
			Data:   data,
			Reason: fmt.Sprintf("validation failed: %v", err),
		}
	}

	return nil
}

// MarshalYAML implements yaml.Marshaler, serializing the Commit to YAML
// object format. This method satisfies part of the model.Serializable
// interface requirement.
//
// MarshalYAML first validates that the Commit conforms to all constraints
// by calling Validate. If validation fails, marshaling fails with the
// validation error, preventing invalid data from being serialized. If
// validation succeeds, the Commit is serialized to a YAML object with all
// fields.
//
// A type alias is used internally to avoid infinite recursion during marshaling.
//
// This method MUST NOT mutate the receiver except as required by the
// yaml.Marshaler interface contract. It MUST be safe to call concurrently
// on immutable receivers.
//
// Example:
//
//	commit := git.Commit{Hash: hash, Summary: "feat: add", ...}
//	data, _ := yaml.Marshal(commit)
func (c Commit) MarshalYAML() (interface{}, error) {
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", c.TypeName(), err)
	}

	// Use type alias to avoid infinite recursion
	type commit Commit
	return commit(c), nil
}

// UnmarshalYAML implements yaml.Unmarshaler, deserializing a YAML object
// into a Commit value. This method satisfies part of the model.Serializable
// interface requirement.
//
// UnmarshalYAML accepts YAML objects with the structure defined in MarshalYAML.
// After unmarshaling the YAML data, Validate is called to ensure the resulting
// Commit conforms to all constraints. If validation fails, unmarshaling fails
// with an error describing the validation failure. This fail-fast behavior
// prevents invalid configuration data from corrupting system state.
//
// The method mutates the receiver to store the unmarshaled value. It is not
// safe for concurrent use during unmarshaling, though the resulting Commit
// value is safe for concurrent reads after unmarshaling completes.
//
// Example:
//
//	var commit git.Commit
//	yaml.Unmarshal(data, &commit)
//	fmt.Println(commit.Hash)
func (c *Commit) UnmarshalYAML(node *yaml.Node) error {
	type commit Commit
	if err := node.Decode((*commit)(c)); err != nil {
		return &errors.UnmarshalError{
			Type:   c.TypeName(),
			Data:   []byte(fmt.Sprintf("%v", node)),
			Reason: err.Error(),
		}
	}

	if err := c.Validate(); err != nil {
		return &errors.UnmarshalError{
			Type:   c.TypeName(),
			Data:   []byte(fmt.Sprintf("%v", node)),
			Reason: fmt.Sprintf("validation failed: %v", err),
		}
	}

	return nil
}
