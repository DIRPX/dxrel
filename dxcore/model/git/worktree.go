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

// WorktreeStatus represents the cleanliness state of a Git working tree,
// summarizing whether there are unstaged changes, staged changes, or untracked
// files present. This type is used to determine if a repository is in a clean
// state before performing operations like releases, builds, or deployments.
//
// WorktreeStatus captures the three key dimensions of working tree cleanliness
// that are reported by "git status":
//   - Unstaged changes: Modified tracked files not yet staged with "git add"
//   - Staged changes: Changes staged for commit but not yet committed
//   - Untracked files: New files not tracked by Git
//
// The zero value of WorktreeStatus (all fields false) represents a completely
// clean working tree with no modifications, staged changes, or untracked files.
// This is the ideal state for performing releases or builds.
//
// A working tree is considered "clean" (Clean() == true) if and only if all
// three flags are false, meaning there are no unstaged changes, no staged
// changes, and no untracked files. Any combination of these flags being true
// indicates a "dirty" working tree that may need attention before proceeding
// with critical operations.
//
// Example working tree states:
//
//	// Completely clean working tree
//	// Nothing to commit, working tree clean
//	WorktreeStatus{
//	    HasUnstaged:   false,
//	    HasStaged:     false,
//	    HasUntracked:  false,
//	}
//
//	// Modified files not staged
//	// Changes not staged for commit: modified: src/main.go
//	WorktreeStatus{
//	    HasUnstaged:   true,
//	    HasStaged:     false,
//	    HasUntracked:  false,
//	}
//
//	// Staged changes ready to commit
//	// Changes to be committed: modified: src/main.go
//	WorktreeStatus{
//	    HasUnstaged:   false,
//	    HasStaged:     true,
//	    HasUntracked:  false,
//	}
//
//	// Untracked files present
//	// Untracked files: temp.txt
//	WorktreeStatus{
//	    HasUnstaged:   false,
//	    HasStaged:     false,
//	    HasUntracked:  true,
//	}
//
//	// Dirty working tree (multiple conditions)
//	// Modified, staged, and untracked files present
//	WorktreeStatus{
//	    HasUnstaged:   true,
//	    HasStaged:     true,
//	    HasUntracked:  true,
//	}
//
// This type implements the model.Model interface, providing validation,
// serialization, logging, and equality operations. WorktreeStatus values are
// safe for concurrent use as all fields are immutable boolean values.
//
// Typical usage:
//  1. Call "git status --porcelain" to get repository status
//  2. Parse output to determine which files are unstaged/staged/untracked
//  3. Construct WorktreeStatus from parsed information
//  4. Use Clean() to determine if operations should proceed
//  5. Log or serialize status for reporting/auditing
type WorktreeStatus struct {
	// HasUnstaged reports whether there are modified tracked files that have
	// not been staged for commit using "git add".
	//
	// This flag is true when "git status" shows modified files in the
	// "Changes not staged for commit" section. These are files that Git is
	// tracking, that have been modified in the working directory, but whose
	// changes have not yet been added to the staging area (index).
	//
	// Examples of unstaged changes:
	//   - Modified files: "modified: src/main.go"
	//   - Deleted files: "deleted: old.txt"
	//   - Files with type changes: "typechange: script.sh"
	//
	// HasUnstaged does NOT include:
	//   - Untracked files (see HasUntracked)
	//   - Already staged changes (see HasStaged)
	//   - Committed changes
	//
	// A true value indicates that "git add" has not been run on all modified
	// tracked files, meaning the working directory differs from the staging area.
	HasUnstaged bool `json:"has_unstaged" yaml:"has_unstaged"`

	// HasStaged reports whether there are staged changes in the index that have
	// not yet been committed.
	//
	// This flag is true when "git status" shows changes in the "Changes to be
	// committed" section. These are changes that have been added to the staging
	// area using "git add" but have not yet been committed with "git commit".
	//
	// Examples of staged changes:
	//   - Staged modifications: "modified: src/main.go"
	//   - Staged additions: "new file: new.go"
	//   - Staged deletions: "deleted: old.txt"
	//   - Staged renames: "renamed: old.go -> new.go"
	//
	// HasStaged does NOT include:
	//   - Unstaged modifications (see HasUnstaged)
	//   - Untracked files (see HasUntracked)
	//   - Already committed changes
	//
	// A true value indicates that "git commit" has not been run to commit the
	// staged changes, meaning the staging area differs from HEAD.
	HasStaged bool `json:"has_staged" yaml:"has_staged"`

	// HasUntracked reports whether there are untracked files present in the
	// working directory.
	//
	// This flag is true when "git status" shows files in the "Untracked files"
	// section. These are files that exist in the working directory but are not
	// tracked by Git (not in the repository) and are not ignored by .gitignore.
	//
	// Examples of untracked files:
	//   - New source files not yet added: "new_feature.go"
	//   - Temporary files not in .gitignore: "debug.log"
	//   - Build artifacts not in .gitignore: "output.bin"
	//
	// HasUntracked does NOT include:
	//   - Modified tracked files (see HasUnstaged or HasStaged)
	//   - Files ignored by .gitignore
	//   - Files in subdirectories if only directory is shown
	//
	// A true value indicates that "git add" has not been run on new files,
	// meaning there are files in the working directory unknown to Git.
	HasUntracked bool `json:"has_untracked" yaml:"has_untracked"`
}

// NewWorktreeStatus constructs a WorktreeStatus with the specified flags
// indicating the presence of unstaged changes, staged changes, and untracked
// files. This function provides a convenient way to create WorktreeStatus
// values with explicit control over each dimension of working tree cleanliness.
//
// NewWorktreeStatus does NOT perform validation because all combinations of
// boolean flags are valid WorktreeStatus states. There are no invalid states
// for this type - even all flags being false (completely clean) or all flags
// being true (maximally dirty) are valid and meaningful.
//
// Parameters:
//   - hasUnstaged: True if there are modified tracked files not yet staged
//   - hasStaged: True if there are staged changes not yet committed
//   - hasUntracked: True if there are untracked files present
//
// Returns a WorktreeStatus representing the specified state.
//
// Example usage:
//
//	// Create status for a clean working tree
//	status := git.NewWorktreeStatus(false, false, false)
//	if status.Clean() {
//	    fmt.Println("Working tree is clean")
//	}
//
//	// Create status for dirty working tree with unstaged changes
//	status := git.NewWorktreeStatus(true, false, false)
//	if !status.Clean() {
//	    fmt.Println("Has unstaged changes")
//	}
//
//	// Create status for completely dirty working tree
//	status := git.NewWorktreeStatus(true, true, true)
//	fmt.Println(status.String())  // Shows all dirty conditions
func NewWorktreeStatus(hasUnstaged, hasStaged, hasUntracked bool) WorktreeStatus {
	return WorktreeStatus{
		HasUnstaged:  hasUnstaged,
		HasStaged:    hasStaged,
		HasUntracked: hasUntracked,
	}
}

// Compile-time assertion that WorktreeStatus implements model.Model.
var _ model.Model = (*WorktreeStatus)(nil)

// Clean reports whether the working tree is completely clean with no
// modifications, staged changes, or untracked files.
//
// Clean returns true if and only if:
//   - HasUnstaged == false (no unstaged changes)
//   - HasStaged == false (no staged changes)
//   - HasUntracked == false (no untracked files)
//
// A clean working tree is in the ideal state for performing operations like:
//   - Creating releases or tags
//   - Running production builds
//   - Switching branches without conflicts
//   - Pulling changes without merge issues
//   - Performing automated deployments
//
// If Clean() returns false, the working tree is "dirty" and may require
// attention before proceeding with critical operations. Use String() to get
// a human-readable description of what makes the tree dirty.
//
// This method MUST NOT mutate the WorktreeStatus receiver and is safe for
// concurrent use from multiple goroutines.
//
// Example usage:
//
//	status := git.WorktreeStatus{HasUnstaged: true}
//	if !status.Clean() {
//	    return fmt.Errorf("cannot release: working tree is dirty (%s)", status.String())
//	}
//
//	status = git.WorktreeStatus{}
//	if status.Clean() {
//	    fmt.Println("Ready to proceed with release")
//	}
func (ws WorktreeStatus) Clean() bool {
	return !ws.HasUnstaged && !ws.HasStaged && !ws.HasUntracked
}

// String returns a human-readable string representation of the WorktreeStatus
// that describes which dimensions of the working tree are dirty, suitable for
// logging, debugging, and user display. This method implements the
// model.Loggable contract through model.Model.
//
// The returned string provides a concise summary:
//   - If Clean(), returns "clean"
//   - Otherwise, returns a comma-separated list of dirty conditions:
//     - "unstaged" if HasUnstaged is true
//     - "staged" if HasStaged is true
//     - "untracked" if HasUntracked is true
//
// This format makes it easy to understand at a glance what makes a working
// tree dirty and what actions might be needed to clean it.
//
// This method MUST NOT mutate the WorktreeStatus receiver and is safe for
// concurrent use from multiple goroutines.
//
// Example return values:
//
//	"clean"                      // All flags false
//	"unstaged"                   // Only HasUnstaged true
//	"staged"                     // Only HasStaged true
//	"untracked"                  // Only HasUntracked true
//	"unstaged, staged"           // HasUnstaged and HasStaged true
//	"unstaged, untracked"        // HasUnstaged and HasUntracked true
//	"staged, untracked"          // HasStaged and HasUntracked true
//	"unstaged, staged, untracked" // All flags true
func (ws WorktreeStatus) String() string {
	if ws.Clean() {
		return "clean"
	}

	var parts []string
	if ws.HasUnstaged {
		parts = append(parts, "unstaged")
	}
	if ws.HasStaged {
		parts = append(parts, "staged")
	}
	if ws.HasUntracked {
		parts = append(parts, "untracked")
	}

	return strings.Join(parts, ", ")
}

// Redacted returns a redacted string representation of the WorktreeStatus,
// which is identical to String() since WorktreeStatus contains no sensitive
// information. This method implements the model.Loggable contract through
// model.Model.
//
// Working tree status (whether files are unstaged, staged, or untracked) is
// not considered sensitive information as it describes the state of the local
// working directory without exposing file contents, names, or other private
// data. Therefore, Redacted() simply delegates to String().
//
// This method MUST NOT mutate the WorktreeStatus receiver and is safe for
// concurrent use from multiple goroutines.
//
// Example usage:
//
//	status := git.WorktreeStatus{HasUnstaged: true, HasStaged: true}
//	log.Info("Repository status", "status", status.Redacted())
//	// Output: Repository status status=unstaged, staged
func (ws WorktreeStatus) Redacted() string {
	return ws.String()
}

// TypeName returns the string "WorktreeStatus", which identifies this type in
// logs, error messages, and serialized output. This method implements the
// model.Identifiable contract through model.Model.
//
// The returned type name is a simple, unqualified identifier that describes
// the semantic purpose of this type without package prefixes. It is used by
// logging frameworks, error messages, and introspection tools to identify the
// type of a model value.
//
// This method MUST NOT mutate the WorktreeStatus receiver and is safe for
// concurrent use from multiple goroutines.
//
// Example usage:
//
//	var status git.WorktreeStatus
//	fmt.Println(status.TypeName())  // Output: WorktreeStatus
//
//	// In error messages
//	return fmt.Errorf("invalid %s: dirty working tree", status.TypeName())
//	// Output: invalid WorktreeStatus: dirty working tree
func (ws WorktreeStatus) TypeName() string {
	return "WorktreeStatus"
}

// IsZero reports whether the WorktreeStatus represents a completely clean
// working tree with all flags false. This method implements the
// model.ZeroCheckable contract through model.Model.
//
// IsZero returns true if and only if all flags are false:
//   - HasUnstaged == false
//   - HasStaged == false
//   - HasUntracked == false
//
// Note that IsZero() and Clean() return the same value - they both check if
// all flags are false. IsZero() is provided to satisfy the model.Model
// interface contract, while Clean() provides domain-specific semantics for
// working tree cleanliness.
//
// This method MUST NOT mutate the WorktreeStatus receiver and is safe for
// concurrent use from multiple goroutines.
//
// Example usage:
//
//	var status git.WorktreeStatus
//	if status.IsZero() {
//	    fmt.Println("Working tree is in zero/clean state")
//	}
//
//	status = git.WorktreeStatus{HasUnstaged: true}
//	if !status.IsZero() {
//	    fmt.Println("Working tree has some changes")
//	}
func (ws WorktreeStatus) IsZero() bool {
	return !ws.HasUnstaged && !ws.HasStaged && !ws.HasUntracked
}

// Equal reports whether the WorktreeStatus is semantically equal to another
// WorktreeStatus by comparing all three boolean flags. This method implements
// the model.Model interface contract.
//
// Two WorktreeStatus values are considered equal if and only if all three
// flags match exactly:
//   - HasUnstaged values are equal
//   - HasStaged values are equal
//   - HasUntracked values are equal
//
// The other parameter must be a WorktreeStatus or *WorktreeStatus. If other
// is a different type, Equal returns false without panicking.
//
// This method MUST NOT mutate the WorktreeStatus receiver and is safe for
// concurrent use from multiple goroutines.
//
// Example usage:
//
//	status1 := git.WorktreeStatus{HasUnstaged: true, HasStaged: false}
//	status2 := git.WorktreeStatus{HasUnstaged: true, HasStaged: false}
//	if status1.Equal(status2) {
//	    fmt.Println("Status values are equal")
//	}
//
//	status3 := git.WorktreeStatus{HasUnstaged: false, HasStaged: true}
//	if !status1.Equal(status3) {
//	    fmt.Println("Status values differ")
//	}
func (ws WorktreeStatus) Equal(other any) bool {
	switch o := other.(type) {
	case WorktreeStatus:
		return ws.HasUnstaged == o.HasUnstaged &&
			ws.HasStaged == o.HasStaged &&
			ws.HasUntracked == o.HasUntracked
	case *WorktreeStatus:
		if o == nil {
			return false
		}
		return ws.HasUnstaged == o.HasUnstaged &&
			ws.HasStaged == o.HasStaged &&
			ws.HasUntracked == o.HasUntracked
	default:
		return false
	}
}

// Validate checks that the WorktreeStatus satisfies all structural and
// semantic constraints. This method implements the model.Validatable contract
// through model.Model.
//
// For WorktreeStatus, ALL combinations of boolean flags are valid. There are
// no invalid states:
//   - All false (clean working tree) is valid
//   - All true (maximally dirty) is valid
//   - Any combination of flags is valid
//
// Therefore, Validate() always returns nil for WorktreeStatus. This method
// exists to satisfy the model.Model interface contract and maintain consistency
// with other model types that do require validation.
//
// This method MUST NOT mutate the WorktreeStatus receiver and is safe for
// concurrent use from multiple goroutines.
//
// Example usage:
//
//	status := git.WorktreeStatus{HasUnstaged: true, HasStaged: true, HasUntracked: true}
//	if err := status.Validate(); err != nil {
//	    // This will never happen - all states are valid
//	    return fmt.Errorf("invalid status: %w", err)
//	}
func (ws WorktreeStatus) Validate() error {
	// All combinations of boolean flags are valid for WorktreeStatus
	return nil
}

// MarshalJSON serializes the WorktreeStatus to JSON format. This method
// implements the json.Marshaler interface and is part of the
// model.Serializable contract through model.Model.
//
// MarshalJSON does NOT validate before marshaling because all WorktreeStatus
// states are valid (Validate() always returns nil). This is a departure from
// other model types that validate before marshaling, but is appropriate here
// since there are no invalid states.
//
// The JSON structure uses a type alias (worktreeStatusJSON) to delegate to
// the standard JSON encoder while avoiding infinite recursion. The resulting
// JSON has the structure:
//
//	{
//	  "has_unstaged": true,
//	  "has_staged": false,
//	  "has_untracked": true
//	}
//
// This method MUST NOT mutate the WorktreeStatus receiver and is safe for
// concurrent use from multiple goroutines.
//
// Example usage:
//
//	status := git.WorktreeStatus{HasUnstaged: true, HasStaged: false, HasUntracked: true}
//	data, err := json.Marshal(status)
//	if err != nil {
//	    return fmt.Errorf("marshal failed: %w", err)
//	}
//	// data contains: {"has_unstaged":true,"has_staged":false,"has_untracked":true}
func (ws WorktreeStatus) MarshalJSON() ([]byte, error) {
	// No validation needed - all states are valid
	type worktreeStatusJSON WorktreeStatus
	return json.Marshal(worktreeStatusJSON(ws))
}

// UnmarshalJSON deserializes a WorktreeStatus from JSON format. This method
// implements the json.Unmarshaler interface and is part of the
// model.Serializable contract through model.Model.
//
// UnmarshalJSON decodes the JSON into the WorktreeStatus structure using a
// type alias (worktreeStatusJSON) to avoid infinite recursion. Since all
// WorktreeStatus states are valid, no validation is performed after decoding.
//
// This method mutates the WorktreeStatus receiver by decoding into it. It is
// NOT safe for concurrent use with other operations on the same WorktreeStatus
// value.
//
// Example usage:
//
//	var status git.WorktreeStatus
//	err := json.Unmarshal(data, &status)
//	if err != nil {
//	    return fmt.Errorf("unmarshal failed: %w", err)
//	}
//	// status now contains the decoded WorktreeStatus
func (ws *WorktreeStatus) UnmarshalJSON(data []byte) error {
	type worktreeStatusJSON WorktreeStatus
	var temp worktreeStatusJSON

	if err := json.Unmarshal(data, &temp); err != nil {
		return &errors.UnmarshalError{
			Type:   ws.TypeName(),
			Data:   data,
			Reason: err.Error(),
		}
	}

	*ws = WorktreeStatus(temp)
	// No validation needed - all states are valid
	return nil
}

// MarshalYAML serializes the WorktreeStatus to YAML format. This method
// implements the yaml.Marshaler interface and is part of the
// model.Serializable contract through model.Model.
//
// MarshalYAML does NOT validate before marshaling because all WorktreeStatus
// states are valid (Validate() always returns nil).
//
// The YAML structure uses a type alias (worktreeStatusYAML) to delegate to
// the standard YAML encoder while avoiding infinite recursion. The resulting
// YAML has the structure:
//
//	has_unstaged: true
//	has_staged: false
//	has_untracked: true
//
// This method MUST NOT mutate the WorktreeStatus receiver and is safe for
// concurrent use from multiple goroutines.
//
// Example usage:
//
//	status := git.WorktreeStatus{HasUnstaged: true}
//	data, err := yaml.Marshal(status)
//	if err != nil {
//	    return fmt.Errorf("marshal failed: %w", err)
//	}
func (ws WorktreeStatus) MarshalYAML() (interface{}, error) {
	// No validation needed - all states are valid
	type worktreeStatusYAML WorktreeStatus
	return worktreeStatusYAML(ws), nil
}

// UnmarshalYAML deserializes a WorktreeStatus from YAML format. This method
// implements the yaml.Unmarshaler interface and is part of the
// model.Serializable contract through model.Model.
//
// UnmarshalYAML decodes the YAML into the WorktreeStatus structure using a
// type alias (worktreeStatusYAML) to avoid infinite recursion. Since all
// WorktreeStatus states are valid, no validation is performed after decoding.
//
// This method mutates the WorktreeStatus receiver by decoding into it. It is
// NOT safe for concurrent use with other operations on the same WorktreeStatus
// value.
//
// Example usage:
//
//	var status git.WorktreeStatus
//	err := yaml.Unmarshal(data, &status)
//	if err != nil {
//	    return fmt.Errorf("unmarshal failed: %w", err)
//	}
//	// status now contains the decoded WorktreeStatus
func (ws *WorktreeStatus) UnmarshalYAML(node *yaml.Node) error {
	type worktreeStatusYAML WorktreeStatus
	var temp worktreeStatusYAML

	if err := node.Decode(&temp); err != nil {
		return &errors.UnmarshalError{
			Type:   ws.TypeName(),
			Data:   []byte(fmt.Sprintf("%v", node)),
			Reason: err.Error(),
		}
	}

	*ws = WorktreeStatus(temp)
	// No validation needed - all states are valid
	return nil
}
