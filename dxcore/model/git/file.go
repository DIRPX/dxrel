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
	// FilePathMaxLength is the maximum allowed length for a file path in Git
	// repositories, measured in bytes.
	//
	// This limit is derived from typical filesystem constraints (4096 bytes on
	// most Unix-like systems including Linux, macOS, and BSD variants) and Git's
	// internal path handling. While Git itself can theoretically handle longer
	// paths, practical considerations including filesystem limits, terminal
	// display width, and tooling compatibility make 4096 bytes a sensible upper
	// bound.
	//
	// This constraint applies to both Path and OldPath fields in FileChange.
	// Paths exceeding this limit MUST be rejected during validation to prevent
	// abuse and ensure compatibility with standard filesystem operations.
	//
	// Note that this is a byte limit, not a Unicode code point (rune) limit.
	// Multi-byte UTF-8 characters in paths count toward this limit based on
	// their encoded byte length, not the number of visible characters.
	FilePathMaxLength = 4096
)

// FileChangeKind describes the kind of change made to a file in a commit.
//
// This type implements the model.Model interface, providing validation,
// serialization to JSON and YAML, safe logging, type identification, and
// zero-value detection.
//
// The zero value of FileChangeKind (FileChangeUnknown) is valid and represents
// "change kind not determined" or "unknown change type", indicating that the
// nature of the file change has not been classified.
//
// JSON and YAML serialization uses string representations ("added", "modified",
// etc.) rather than numeric values to ensure human readability and forward
// compatibility when new kinds are added in future versions.
//
// Example values:
//   - FileChangeAdded: New file created
//   - FileChangeModified: Existing file content changed
//   - FileChangeDeleted: File removed
//   - FileChangeRenamed: File moved to new path
//   - FileChangeCopied: File copied to new path
//   - FileChangeType: File type changed (e.g., regular file ↔ symlink)
type FileChangeKind uint8

const (
	// FileChangeUnknown represents an unknown or unclassified file change.
	//
	// This is the zero value for FileChangeKind. It is valid and MAY be used
	// in data structures where the change kind has not yet been determined.
	FileChangeUnknown FileChangeKind = iota

	// FileChangeAdded represents a newly created file.
	//
	// The file did not exist in the parent commit and was created in this commit.
	FileChangeAdded

	// FileChangeModified represents a file whose content was changed.
	//
	// The file existed in the parent commit and its content differs in this commit.
	// The path remains the same.
	FileChangeModified

	// FileChangeDeleted represents a file that was removed.
	//
	// The file existed in the parent commit but was deleted in this commit.
	FileChangeDeleted

	// FileChangeRenamed represents a file that was moved to a new path.
	//
	// The file existed in the parent commit at OldPath and now exists at Path.
	// Content may or may not have changed.
	FileChangeRenamed

	// FileChangeCopied represents a file that was copied to a new path.
	//
	// The file existed in the parent commit at OldPath and now also exists at Path.
	// The original file at OldPath still exists.
	FileChangeCopied

	// FileChangeType represents a file whose type changed.
	//
	// For example, a regular file became a symlink, or vice versa.
	// The path remains the same, but the file type differs.
	FileChangeType
)

// String constants for each FileChangeKind value, enabling type-safe string
// comparisons in switch statements and other contexts. These constants represent
// the canonical lowercase form used for JSON and YAML serialization.
//
// Example usage in switch statements:
//
//	switch kindStr {
//	case git.FileChangeAddedStr:
//	    // Handle added files
//	case git.FileChangeModifiedStr:
//	    // Handle modified files
//	}
const (
	// FileChangeUnknownStr is the string representation of FileChangeUnknown.
	FileChangeUnknownStr = "unknown"

	// FileChangeAddedStr is the string representation of FileChangeAdded.
	FileChangeAddedStr = "added"

	// FileChangeModifiedStr is the string representation of FileChangeModified.
	FileChangeModifiedStr = "modified"

	// FileChangeDeletedStr is the string representation of FileChangeDeleted.
	FileChangeDeletedStr = "deleted"

	// FileChangeRenamedStr is the string representation of FileChangeRenamed.
	FileChangeRenamedStr = "renamed"

	// FileChangeCopiedStr is the string representation of FileChangeCopied.
	FileChangeCopiedStr = "copied"

	// FileChangeTypeStr is the canonical string representation of FileChangeType.
	//
	// Alternative formats "type_changed" and "typechanged" are also accepted
	// during parsing for compatibility, but this canonical form with hyphen
	// is used for serialization.
	FileChangeTypeStr = "type-changed"
)

// ParseFileChangeKind parses a string into a validated FileChangeKind value,
// normalizing and validating the input before matching against known file change
// kind names. This function provides a unified parsing entry point for converting
// external string representations into FileChangeKind values with comprehensive
// input validation.
//
// ParseFileChangeKind recognizes all standard file change kind names: "unknown",
// "added", "modified", "deleted", "renamed", "copied", and "type-changed" (with
// alternative forms "type_changed" and "typechanged" for compatibility). The input
// undergoes normalization before matching: leading and trailing whitespace is
// removed using strings.TrimSpace, and the result is converted to lowercase using
// strings.ToLower. This ensures that inputs like "  ADDED  ", "Added", and "added"
// all parse to the same FileChangeKind value.
//
// ParseFileChangeKind returns an error if the normalized input does not match
// any known kind name. The error message includes the original invalid input
// (before normalization) to aid debugging and provide clear feedback to users
// about what they provided.
//
// Callers MUST check the returned error before using the FileChangeKind value.
// The zero value returned on error (FileChangeKind(0), which equals FileChangeUnknown)
// MUST NOT be used when an error is returned, as it does not represent a
// successfully parsed value.
//
// This function is pure and has no side effects. It is safe to call concurrently
// from multiple goroutines. The normalization process ensures consistent behavior
// regardless of input casing or surrounding whitespace.
//
// Example:
//
//	kind, err := git.ParseFileChangeKind("added")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(kind == git.FileChangeAdded) // Output: true
//
//	kind, err := git.ParseFileChangeKind("  MODIFIED  ")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(kind == git.FileChangeModified) // Output: true
func ParseFileChangeKind(s string) (FileChangeKind, error) {
	// Normalize: trim whitespace and convert to lowercase
	normalized := strings.ToLower(strings.TrimSpace(s))

	switch normalized {
	case FileChangeUnknownStr:
		return FileChangeUnknown, nil
	case FileChangeAddedStr:
		return FileChangeAdded, nil
	case FileChangeModifiedStr:
		return FileChangeModified, nil
	case FileChangeDeletedStr:
		return FileChangeDeleted, nil
	case FileChangeRenamedStr:
		return FileChangeRenamed, nil
	case FileChangeCopiedStr:
		return FileChangeCopied, nil
	case FileChangeTypeStr, "type_changed", "typechanged": // Accept alternatives
		return FileChangeType, nil
	default:
		return FileChangeUnknown, fmt.Errorf("unknown FileChangeKind: %q", s)
	}
}

// String returns the lowercase string representation of the FileChangeKind.
// This method satisfies the model.Loggable interface's String requirement,
// providing a human-readable representation suitable for display and debugging.
//
// The returned strings are: "unknown", "added", "modified", "deleted", "renamed",
// "copied", "type-changed". If the FileChangeKind value is invalid (out of range),
// String returns a formatted representation like "FileChangeKind(255)" to prevent
// crashes or silent failures.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently. The returned string is always a constant
// literal for valid kinds, ensuring zero allocations.
//
// Example:
//
//	kind := git.FileChangeAdded
//	fmt.Println(kind.String()) // Output: "added"
func (k FileChangeKind) String() string {
	switch k {
	case FileChangeUnknown:
		return FileChangeUnknownStr
	case FileChangeAdded:
		return FileChangeAddedStr
	case FileChangeModified:
		return FileChangeModifiedStr
	case FileChangeDeleted:
		return FileChangeDeletedStr
	case FileChangeRenamed:
		return FileChangeRenamedStr
	case FileChangeCopied:
		return FileChangeCopiedStr
	case FileChangeType:
		return FileChangeTypeStr
	default:
		return fmt.Sprintf("FileChangeKind(%d)", uint8(k))
	}
}

// Redacted returns a safe string representation suitable for logging in
// production environments. For FileChangeKind, which contains no sensitive data,
// Redacted is identical to String and returns the lowercase kind name.
//
// This method satisfies the model.Loggable interface's Redacted requirement,
// ensuring that FileChangeKind can be safely logged without risk of exposing
// sensitive information. File change kinds are not sensitive, no masking or
// redaction is necessary.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently.
//
// Example:
//
//	kind := git.FileChangeModified
//	log.Info("processing file", "change", kind.Redacted()) // Safe for production logs
func (k FileChangeKind) Redacted() string {
	return k.String()
}

// TypeName returns the canonical name of this model type for debugging,
// logging, and reflection purposes. This method satisfies the model.Identifiable
// interface's TypeName requirement.
//
// The returned value is always the constant string "FileChangeKind", uniquely
// identifying this type within the dxrel domain. The name follows CamelCase
// convention and omits the package prefix as required by the Model contract.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently. The returned string is a constant literal,
// ensuring zero allocations.
func (k FileChangeKind) TypeName() string {
	return "FileChangeKind"
}

// IsZero reports whether this FileChangeKind instance is in a zero or empty
// state. For FileChangeKind, the zero value (0) corresponds to FileChangeUnknown,
// which represents "change kind not determined" or "unknown change type".
//
// This method satisfies the model.ZeroCheckable interface's IsZero requirement.
// The zero value is valid and MAY appear in data structures where a file change
// kind has not been classified.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently.
//
// Example:
//
//	var kind git.FileChangeKind // Zero value, equals FileChangeUnknown
//	fmt.Println(kind.IsZero()) // Output: true
//
//	kind = git.FileChangeAdded
//	fmt.Println(kind.IsZero()) // Output: false
func (k FileChangeKind) IsZero() bool {
	return k == FileChangeUnknown
}

// Equal reports whether this FileChangeKind is equal to another FileChangeKind
// value, providing an explicit equality comparison method that follows common Go
// idioms for value types. While FileChangeKind values can be compared using the
// == operator directly, this method offers a named alternative that improves code
// readability and maintains consistency with other model types in the dxrel
// codebase.
//
// Equal performs a simple value comparison and returns true if both FileChangeKind
// values represent the same change kind constant. The comparison is exact and
// considers only the underlying uint8 representation. Zero values (FileChangeUnknown)
// are equal to other zero values, and each defined constant is equal only to itself.
//
// This method is particularly useful in table-driven tests, assertion libraries,
// and comparison functions where a method-based approach is more idiomatic than
// operator-based comparison. It also provides a consistent interface across all
// model types, some of which MAY require more complex equality semantics than
// simple value comparison.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently. The comparison is fast, deterministic,
// and performs no allocations.
//
// Example:
//
//	k1 := git.FileChangeAdded
//	k2 := git.FileChangeModified
//	k3 := git.FileChangeAdded
//	fmt.Println(k1.Equal(k2)) // Output: false
//	fmt.Println(k1.Equal(k3)) // Output: true
func (k FileChangeKind) Equal(other FileChangeKind) bool {
	return k == other
}

// Validate checks that the FileChangeKind value is within the valid range of
// defined constants. This method satisfies the model.Validatable interface's
// Validate requirement, enforcing data integrity.
//
// Validate returns nil if the FileChangeKind is one of the defined constants
// (FileChangeUnknown through FileChangeType). It returns an error if the
// FileChangeKind value is out of range, which can occur through type conversions,
// unsafe operations, or deserialization bugs.
//
// This method MUST be fast, deterministic, and idempotent. It MUST NOT mutate
// the receiver, MUST NOT have side effects, and MUST be safe to call concurrently.
// Validation does not perform I/O or allocate memory except when constructing
// error messages for invalid values.
//
// Callers SHOULD invoke Validate after deserializing FileChangeKind from external
// sources (JSON, YAML, databases) to ensure data integrity. The ToJSON, ToYAML,
// FromJSON, and FromYAML helper functions automatically call Validate to enforce
// this contract.
//
// Example:
//
//	kind := git.FileChangeAdded
//	if err := kind.Validate(); err != nil {
//	    log.Error("invalid kind", "error", err)
//	}
func (k FileChangeKind) Validate() error {
	switch k {
	case FileChangeUnknown, FileChangeAdded, FileChangeModified,
		FileChangeDeleted, FileChangeRenamed, FileChangeCopied, FileChangeType:
		return nil
	default:
		return &errors.ValidationError{
			Type:   k.TypeName(),
			Field:  "",
			Reason: fmt.Sprintf("invalid value: %d", uint8(k)),
			Value:  uint8(k),
		}
	}
}

// MarshalJSON implements json.Marshaler, serializing the FileChangeKind to its
// lowercase string representation as a JSON string. This method satisfies part
// of the model.Serializable interface requirement.
//
// MarshalJSON first validates that the FileChangeKind is in the valid range by
// calling Validate. If validation fails, marshaling fails with the validation
// error, preventing invalid data from being serialized. If validation succeeds,
// the FileChangeKind is converted to its string representation using String and
// marshaled as a JSON string.
//
// The output format is compatible with standard Git diff formats and tooling.
// For example, FileChangeAdded marshals to the JSON string "added", FileChangeModified
// marshals to "modified", and so on.
//
// This method MUST NOT mutate the receiver except as required by the json.Marshaler
// interface contract. It MUST be safe to call concurrently on immutable receivers.
//
// Example:
//
//	kind := git.FileChangeAdded
//	data, _ := json.Marshal(kind)
//	fmt.Println(string(data)) // Output: "added"
func (k FileChangeKind) MarshalJSON() ([]byte, error) {
	if err := k.Validate(); err != nil {
		return nil, &errors.MarshalError{
			Type:  k.TypeName(),
			Value: int(k),
		}
	}
	return json.Marshal(k.String())
}

// UnmarshalJSON implements json.Unmarshaler, deserializing a JSON string into
// a FileChangeKind value. This method satisfies part of the model.Serializable
// interface requirement.
//
// UnmarshalJSON accepts JSON strings containing lowercase kind names ("added",
// "modified", "deleted", etc.). It also accepts uppercase variants ("ADDED",
// "MODIFIED") for flexibility, though lowercase is canonical and SHOULD be
// preferred in serialized data.
//
// After unmarshaling, Validate is called via ParseFileChangeKind to ensure the
// resulting FileChangeKind is valid. If the input string does not match any
// known kind name, unmarshaling fails with an error indicating the unknown kind.
// This fail-fast behavior prevents invalid data from entering the system through
// external inputs.
//
// The method mutates the receiver to store the unmarshaled value. It is not
// safe for concurrent use during unmarshaling, though the resulting FileChangeKind
// value is safe for concurrent reads after unmarshaling completes.
//
// Example:
//
//	var kind git.FileChangeKind
//	json.Unmarshal([]byte(`"modified"`), &kind)
//	fmt.Println(kind == git.FileChangeModified) // Output: true
func (k *FileChangeKind) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return &errors.UnmarshalError{
			Type:   k.TypeName(),
			Data:   data,
			Reason: err.Error(),
		}
	}

	parsed, err := ParseFileChangeKind(s)
	if err != nil {
		return &errors.UnmarshalError{
			Type:   k.TypeName(),
			Data:   data,
			Reason: fmt.Sprintf("unknown value: %q", s),
		}
	}

	*k = parsed
	return nil
}

// MarshalYAML implements yaml.Marshaler, serializing the FileChangeKind to its
// lowercase string representation for YAML encoding. This method satisfies part
// of the model.Serializable interface requirement.
//
// MarshalYAML first validates that the FileChangeKind is in the valid range by
// calling Validate. If validation fails, marshaling fails with the validation
// error, preventing invalid data from being serialized. If validation succeeds,
// the FileChangeKind is converted to its string representation using String.
//
// The output format is compatible with standard Git configurations and tooling.
// For example, FileChangeAdded marshals to the YAML scalar "added", FileChangeModified
// marshals to "modified", and so on.
//
// This method MUST NOT mutate the receiver except as required by the yaml.Marshaler
// interface contract. It MUST be safe to call concurrently on immutable receivers.
//
// Example:
//
//	kind := git.FileChangeRenamed
//	data, _ := yaml.Marshal(kind)
//	fmt.Println(string(data)) // Output: "renamed\n"
func (k FileChangeKind) MarshalYAML() (interface{}, error) {
	if err := k.Validate(); err != nil {
		return nil, &errors.MarshalError{
			Type:  k.TypeName(),
			Value: int(k),
		}
	}
	return k.String(), nil
}

// UnmarshalYAML implements yaml.Unmarshaler, deserializing a YAML scalar into
// a FileChangeKind value. This method satisfies part of the model.Serializable
// interface requirement.
//
// UnmarshalYAML accepts YAML scalars containing lowercase kind names ("added",
// "modified", "deleted", etc.). It also accepts uppercase variants for flexibility,
// though lowercase is canonical and SHOULD be preferred in YAML files.
//
// After unmarshaling, Validate is called via ParseFileChangeKind to ensure the
// resulting FileChangeKind is valid. If the input string does not match any known
// kind name, unmarshaling fails with an error indicating the unknown kind. This
// fail-fast behavior prevents invalid configuration data from corrupting system state.
//
// The method mutates the receiver to store the unmarshaled value. It is not
// safe for concurrent use during unmarshaling, though the resulting FileChangeKind
// value is safe for concurrent reads after unmarshaling completes.
//
// Example:
//
//	var kind git.FileChangeKind
//	yaml.Unmarshal([]byte("deleted"), &kind)
//	fmt.Println(kind == git.FileChangeDeleted) // Output: true
func (k *FileChangeKind) UnmarshalYAML(node *yaml.Node) error {
	var s string
	if err := node.Decode(&s); err != nil {
		return &errors.UnmarshalError{
			Type:   k.TypeName(),
			Data:   []byte(fmt.Sprintf("%v", node)),
			Reason: err.Error(),
		}
	}

	parsed, err := ParseFileChangeKind(s)
	if err != nil {
		return &errors.UnmarshalError{
			Type:   k.TypeName(),
			Data:   []byte(fmt.Sprintf("%v", node)),
			Reason: fmt.Sprintf("unknown value: %q", s),
		}
	}

	*k = parsed
	return nil
}

// Compile-time check that FileChangeKind implements model.Model
var _ model.Model = (*FileChangeKind)(nil)

// FileChange describes a single file change in a Git commit, tracking the
// path(s) affected and the nature of the change.
//
// This type implements the model.Model interface, providing validation,
// serialization to JSON and YAML, safe logging, type identification, and
// zero-value detection.
//
// A FileChange combines path information with change classification:
//   - Path: The current/destination path
//   - OldPath: The previous/source path (for renames and copies)
//   - Kind: The type of change (added, modified, deleted, etc.)
//
// The zero value of FileChange (all fields zero) is valid and represents
// "no change specified", but will fail validation if Validate() is called.
//
// Example values:
//
//	FileChange{
//	    Path: "src/main.go",
//	    Kind: FileChangeModified,
//	}
//
//	FileChange{
//	    Path:    "internal/config/settings.go",
//	    OldPath: "pkg/config/settings.go",
//	    Kind:    FileChangeRenamed,
//	}
type FileChange struct {
	// Path is the new/current path of the file relative to the repository root.
	//
	// For added, modified, deleted changes this is the canonical path.
	// For renames and copies this is the destination path.
	//
	// Path MUST use forward slashes (/) as path separators, even on Windows.
	// Path MUST be relative to the repository root (no leading slash).
	// Path SHOULD NOT be empty for a valid FileChange.
	//
	// Maximum length is 4096 characters.
	//
	// Examples:
	//   - "README.md"
	//   - "src/main.go"
	//   - "internal/pkg/util/helper.go"
	Path string `json:"path" yaml:"path"`

	// OldPath is the previous path for renames and copies.
	//
	// This field is ONLY meaningful when Kind is FileChangeRenamed or
	// FileChangeCopied. For all other kinds, OldPath SHOULD be empty.
	//
	// When present, OldPath follows the same format rules as Path.
	//
	// Examples:
	//   - For rename: OldPath="old/path.go", Path="new/path.go"
	//   - For copy: OldPath="template.txt", Path="copy.txt"
	OldPath string `json:"old_path,omitempty" yaml:"old_path,omitempty"`

	// Kind is the kind of change applied to this file.
	//
	// This categorizes the nature of the change (added, modified, deleted,
	// renamed, copied, or type-changed).
	//
	// Kind SHOULD NOT be FileChangeUnknown for a fully resolved FileChange.
	Kind FileChangeKind `json:"kind" yaml:"kind"`
}

// Compile-time check that FileChange implements model.Model
var _ model.Model = (*FileChange)(nil)

// NewFileChange creates a new FileChange with the given Path and Kind, validating
// the result before returning.
//
// This is a convenience constructor that creates and validates a FileChange in
// one step, primarily intended for simple file changes (added, modified, deleted,
// type-changed) that do not involve path changes. For renames and copies that
// require both Path and OldPath, callers SHOULD construct FileChange directly
// using a struct literal and set both path fields explicitly.
//
// If the Path is invalid, Kind is invalid, or the combination violates FileChange
// validation rules, NewFileChange returns a zero FileChange and an error describing
// the validation failure.
//
// This function is pure and has no side effects. It is safe to call concurrently
// from multiple goroutines.
//
// Example usage:
//
//	// Simple modification
//	change, err := git.NewFileChange("src/main.go", git.FileChangeModified)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// For renames, use struct literal instead
//	rename := git.FileChange{
//	    Path:    "new/location.go",
//	    OldPath: "old/location.go",
//	    Kind:    git.FileChangeRenamed,
//	}
//	if err := rename.Validate(); err != nil {
//	    log.Fatal(err)
//	}
func NewFileChange(path string, kind FileChangeKind) (FileChange, error) {
	fc := FileChange{
		Path: path,
		Kind: kind,
	}

	if err := fc.Validate(); err != nil {
		return FileChange{}, err
	}

	return fc, nil
}

// String returns the human-readable representation of the FileChange for display
// and debugging purposes. This method implements the fmt.Stringer interface and
// satisfies the model.Loggable contract's String() requirement.
//
// The output format varies based on whether OldPath is present. For simple changes
// (added, modified, deleted, type-changed), the format is:
//
//	FileChange{Path:<path>, Kind:<kind>}
//
// For renames and copies where OldPath is set, the format includes the old path:
//
//	FileChange{Path:<path>, OldPath:<oldpath>, Kind:<kind>}
//
// All components are rendered using their String() methods, providing full detail
// for debugging. For production logging where sensitive information might be
// present, use Redacted() instead (though file paths are typically not sensitive).
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently.
//
// Examples:
//
//	fc := git.FileChange{Path: "main.go", Kind: git.FileChangeModified}
//	fmt.Println(fc.String())
//	// Output: "FileChange{Path:main.go, Kind:modified}"
//
//	fc = git.FileChange{Path: "new.go", OldPath: "old.go", Kind: git.FileChangeRenamed}
//	fmt.Println(fc.String())
//	// Output: "FileChange{Path:new.go, OldPath:old.go, Kind:renamed}"
func (fc FileChange) String() string {
	if fc.OldPath != "" {
		return fmt.Sprintf("FileChange{Path:%s, OldPath:%s, Kind:%s}",
			fc.Path, fc.OldPath, fc.Kind.String())
	}
	return fmt.Sprintf("FileChange{Path:%s, Kind:%s}",
		fc.Path, fc.Kind.String())
}

// Redacted returns a safe string representation suitable for logging in
// production environments. For FileChange, which typically contains non-sensitive
// public repository paths, Redacted delegates to component Redacted() methods
// for consistency with the model.Loggable contract.
//
// This method satisfies the model.Loggable interface's Redacted requirement.
// File paths in Git repositories are generally public metadata and do not contain
// passwords, tokens, API keys, or personally identifiable information. However,
// if file paths in your specific context might contain sensitive information
// (e.g., usernames in home directory paths), callers SHOULD sanitize paths
// before creating FileChange instances.
//
// The implementation delegates to Kind.Redacted(), ensuring consistent redaction
// behavior across all model types, even though Kind redaction is currently
// identical to its String() output.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently.
//
// Example:
//
//	fc := git.FileChange{Path: "config/settings.yaml", Kind: git.FileChangeModified}
//	log.Info("processing file change", "change", fc.Redacted())
func (fc FileChange) Redacted() string {
	if fc.OldPath != "" {
		return fmt.Sprintf("FileChange{Path:%s, OldPath:%s, Kind:%s}",
			fc.Path, fc.OldPath, fc.Kind.Redacted())
	}
	return fmt.Sprintf("FileChange{Path:%s, Kind:%s}",
		fc.Path, fc.Kind.Redacted())
}

// TypeName returns the canonical name of this model type for debugging,
// logging, and reflection purposes. This method satisfies the model.Identifiable
// interface's TypeName requirement.
//
// The returned value is always the constant string "FileChange", uniquely
// identifying this type within the dxrel domain. The name follows CamelCase
// convention and omits the package prefix as required by the Model contract.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently. The returned string is a constant literal,
// ensuring zero allocations.
func (fc FileChange) TypeName() string {
	return "FileChange"
}

// IsZero reports whether this FileChange instance is in a zero or empty state,
// meaning no file change has been specified. For FileChange, the zero value
// represents "no change specified" or "change not initialized".
//
// This method satisfies the model.ZeroCheckable interface's IsZero requirement.
// A FileChange is considered zero if all three fields (Path, OldPath, Kind) are
// their zero values: empty strings for paths and FileChangeUnknown for Kind.
//
// Zero-value FileChanges are semantically invalid for most operations and will
// fail validation if Validate() is called. However, the zero value is useful
// as a sentinel for "no change specified" in optional fields or when initializing
// data structures.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently.
//
// Example:
//
//	var fc git.FileChange // Zero value
//	fmt.Println(fc.IsZero()) // Output: true
//
//	fc = git.FileChange{Path: "main.go", Kind: git.FileChangeModified}
//	fmt.Println(fc.IsZero()) // Output: false
func (fc FileChange) IsZero() bool {
	return fc.Path == "" && fc.OldPath == "" && fc.Kind.IsZero()
}

// Equal reports whether this FileChange is equal to another FileChange value.
//
// Two FileChanges are equal if and only if all three components match:
//   - Path must be equal (case-sensitive string comparison)
//   - OldPath must be equal (case-sensitive string comparison)
//   - Kind must be equal (delegates to Kind.Equal for consistency)
//
// This method is particularly useful in table-driven tests, assertion libraries,
// deduplication logic, and comparison operations where a method-based approach
// is more idiomatic than manual field-by-field comparison.
//
// File paths in Git are case-sensitive on case-sensitive filesystems (Linux,
// macOS with case sensitivity enabled) and case-preserving on case-insensitive
// filesystems (Windows, macOS default). This method uses exact string comparison
// to match Git's behavior on case-sensitive systems.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently. The comparison is fast, deterministic,
// and performs no allocations beyond the standard string comparison.
//
// Example:
//
//	fc1 := git.FileChange{Path: "main.go", Kind: git.FileChangeAdded}
//	fc2 := git.FileChange{Path: "main.go", Kind: git.FileChangeAdded}
//	fc3 := git.FileChange{Path: "test.go", Kind: git.FileChangeAdded}
//	fmt.Println(fc1.Equal(fc2)) // Output: true
//	fmt.Println(fc1.Equal(fc3)) // Output: false (different Path)
func (fc FileChange) Equal(other FileChange) bool {
	return fc.Path == other.Path &&
		fc.OldPath == other.OldPath &&
		fc.Kind.Equal(other.Kind)
}

// Validate checks whether this FileChange satisfies all model contracts and
// invariants. This method implements the model.Validatable interface's Validate
// requirement, enforcing data integrity for file change records.
//
// Validate returns nil if the FileChange conforms to all of the following
// requirements:
//
// Path validation:
//   - Path MUST NOT be empty (empty paths are invalid for non-zero FileChanges)
//   - Path length MUST NOT exceed FilePathMaxLength (4096 bytes)
//   - Path MUST be relative (MUST NOT start with "/" to ensure repository-relative paths)
//
// Kind validation:
//   - Kind MUST be a valid FileChangeKind value (delegates to Kind.Validate())
//
// OldPath validation and consistency:
//   - If OldPath is non-empty, Kind MUST be FileChangeRenamed or FileChangeCopied
//   - If Kind is FileChangeRenamed or FileChangeCopied, OldPath SHOULD be set
//     (but this is not enforced as a hard error for partial data scenarios)
//   - When OldPath is present, it MUST follow the same format rules as Path:
//     - Maximum length of FilePathMaxLength (4096 bytes)
//     - Must be relative (no leading slash)
//
// This method MUST be fast, deterministic, and idempotent. It MUST NOT mutate
// the receiver, MUST NOT have side effects, and MUST be safe to call concurrently.
// Validation does not perform I/O or allocate memory except when constructing
// error messages for invalid values.
//
// Callers SHOULD invoke Validate after creating FileChange instances from external
// sources (JSON, YAML, Git commands, user input) to ensure data integrity. The
// marshal/unmarshal methods automatically call Validate to enforce this contract.
//
// Example:
//
//	fc := git.FileChange{Path: "main.go", Kind: git.FileChangeModified}
//	if err := fc.Validate(); err != nil {
//	    log.Error("invalid file change", "error", err)
//	}
//
//	// Invalid: OldPath set for non-rename/copy
//	fc = git.FileChange{Path: "new.go", OldPath: "old.go", Kind: git.FileChangeModified}
//	err := fc.Validate()
//	// err: "FileChange OldPath should only be set for renamed/copied files (got kind=modified)"
func (fc FileChange) Validate() error {
	// Validate Path
	if fc.Path == "" {
		return &errors.ValidationError{
			Type:   fc.TypeName(),
			Field:  "Path",
			Reason: "must not be empty",
		}
	}
	if len(fc.Path) > FilePathMaxLength {
		return &errors.ValidationError{
			Type:   fc.TypeName(),
			Field:  "Path",
			Reason: fmt.Sprintf("exceeds maximum length of %d characters (got %d)", FilePathMaxLength, len(fc.Path)),
		}
	}
	if strings.HasPrefix(fc.Path, "/") {
		return &errors.ValidationError{
			Type:   fc.TypeName(),
			Field:  "Path",
			Reason: fmt.Sprintf("must be relative (no leading slash): %q", fc.Path),
		}
	}

	// Validate Kind
	if err := fc.Kind.Validate(); err != nil {
		return &errors.ValidationError{
			Type:   fc.TypeName(),
			Field:  "Kind",
			Reason: fmt.Sprintf("invalid: %v", err),
		}
	}

	// Validate OldPath consistency
	if fc.OldPath != "" {
		// OldPath should only be set for renames and copies
		if fc.Kind != FileChangeRenamed && fc.Kind != FileChangeCopied {
			return &errors.ValidationError{
				Type:   fc.TypeName(),
				Field:  "OldPath",
				Reason: fmt.Sprintf("should only be set for renamed/copied files (got kind=%s)", fc.Kind.String()),
			}
		}

		// Validate OldPath format
		if len(fc.OldPath) > FilePathMaxLength {
			return &errors.ValidationError{
				Type:   fc.TypeName(),
				Field:  "OldPath",
				Reason: fmt.Sprintf("exceeds maximum length of %d characters (got %d)", FilePathMaxLength, len(fc.OldPath)),
			}
		}
		if strings.HasPrefix(fc.OldPath, "/") {
			return &errors.ValidationError{
				Type:   fc.TypeName(),
				Field:  "OldPath",
				Reason: fmt.Sprintf("must be relative (no leading slash): %q", fc.OldPath),
			}
		}
	} else {
		// Warn if OldPath is missing for rename/copy (but don't fail - might be partial data)
		// This is informational only, not a hard error
	}

	return nil
}

// MarshalJSON implements json.Marshaler, serializing the FileChange to JSON
// object format. This method satisfies part of the model.Serializable interface
// requirement.
//
// MarshalJSON first validates that the FileChange conforms to all constraints
// by calling Validate. If validation fails, marshaling fails with the validation
// error, preventing invalid data from being serialized. If validation succeeds,
// the FileChange is serialized to a JSON object with three fields:
//
//	{
//	  "path": "src/main.go",
//	  "old_path": "old/main.go",  // omitted if empty
//	  "kind": "modified"
//	}
//
// The "old_path" field is omitted (via `omitempty` JSON tag) when OldPath is
// empty, reducing JSON payload size for simple changes. The "kind" field is
// always present and serializes to the lowercase string representation.
//
// A type alias is used internally to avoid infinite recursion during the
// standard library json.Marshal call.
//
// This method MUST NOT mutate the receiver except as required by the json.Marshaler
// interface contract. It MUST be safe to call concurrently on immutable receivers.
//
// Example:
//
//	fc := git.FileChange{Path: "src/main.go", Kind: git.FileChangeModified}
//	data, _ := json.Marshal(fc)
//	fmt.Println(string(data))
//	// Output: {"path":"src/main.go","kind":"modified"}
func (fc FileChange) MarshalJSON() ([]byte, error) {
	if err := fc.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", fc.TypeName(), err)
	}

	// Use type alias to avoid infinite recursion
	type fileChange FileChange
	return json.Marshal(fileChange(fc))
}

// UnmarshalJSON implements json.Unmarshaler, deserializing a JSON object into
// a FileChange value. This method satisfies part of the model.Serializable
// interface requirement.
//
// UnmarshalJSON accepts JSON objects with the following structure:
//
//	{
//	  "path": "src/main.go",
//	  "old_path": "old/main.go",  // optional
//	  "kind": "modified"
//	}
//
// After unmarshaling the JSON data, Validate is called to ensure the resulting
// FileChange conforms to all constraints. If validation fails (for example, invalid
// path, invalid kind, or OldPath inconsistency), unmarshaling fails with an error
// describing the validation failure. This fail-fast behavior prevents invalid
// data from entering the system through external inputs.
//
// The method mutates the receiver to store the unmarshaled value. It is not
// safe for concurrent use during unmarshaling, though the resulting FileChange
// value is safe for concurrent reads after unmarshaling completes.
//
// Example:
//
//	var fc git.FileChange
//	json.Unmarshal([]byte(`{"path":"main.go","kind":"added"}`), &fc)
//	fmt.Println(fc.Path, fc.Kind)
//	// Output: main.go added
func (fc *FileChange) UnmarshalJSON(data []byte) error {
	type fileChange FileChange
	if err := json.Unmarshal(data, (*fileChange)(fc)); err != nil {
		return &errors.UnmarshalError{
			Type:   fc.TypeName(),
			Data:   data,
			Reason: err.Error(),
		}
	}

	if err := fc.Validate(); err != nil {
		return &errors.UnmarshalError{
			Type:   fc.TypeName(),
			Data:   data,
			Reason: fmt.Sprintf("validation failed: %v", err),
		}
	}

	return nil
}

// MarshalYAML implements yaml.Marshaler, serializing the FileChange to YAML
// object format. This method satisfies part of the model.Serializable interface
// requirement.
//
// MarshalYAML first validates that the FileChange conforms to all constraints
// by calling Validate. If validation fails, marshaling fails with the validation
// error, preventing invalid data from being serialized. If validation succeeds,
// the FileChange is serialized to a YAML object:
//
//	path: src/main.go
//	old_path: old/main.go  # omitted if empty
//	kind: modified
//
// The "old_path" field is omitted (via `omitempty` YAML tag) when OldPath is
// empty, improving YAML readability for simple changes.
//
// A type alias is used internally to avoid infinite recursion during marshaling.
//
// This method MUST NOT mutate the receiver except as required by the yaml.Marshaler
// interface contract. It MUST be safe to call concurrently on immutable receivers.
//
// Example:
//
//	fc := git.FileChange{Path: "src/main.go", Kind: git.FileChangeDeleted}
//	data, _ := yaml.Marshal(fc)
//	fmt.Println(string(data))
//	// Output:
//	// path: src/main.go
//	// kind: deleted
func (fc FileChange) MarshalYAML() (interface{}, error) {
	if err := fc.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", fc.TypeName(), err)
	}

	// Use type alias to avoid infinite recursion
	type fileChange FileChange
	return fileChange(fc), nil
}

// UnmarshalYAML implements yaml.Unmarshaler, deserializing a YAML object into
// a FileChange value. This method satisfies part of the model.Serializable
// interface requirement.
//
// UnmarshalYAML accepts YAML objects with the following structure:
//
//	path: src/main.go
//	old_path: old/main.go  # optional
//	kind: modified
//
// After unmarshaling the YAML data, Validate is called to ensure the resulting
// FileChange conforms to all constraints. If validation fails, unmarshaling fails
// with an error describing the validation failure. This fail-fast behavior prevents
// invalid configuration data from corrupting system state.
//
// The method mutates the receiver to store the unmarshaled value. It is not
// safe for concurrent use during unmarshaling, though the resulting FileChange
// value is safe for concurrent reads after unmarshaling completes.
//
// Example:
//
//	var fc git.FileChange
//	yaml.Unmarshal([]byte("path: test.go\nkind: added"), &fc)
//	fmt.Println(fc.Path, fc.Kind)
//	// Output: test.go added
func (fc *FileChange) UnmarshalYAML(node *yaml.Node) error {
	type fileChange FileChange
	if err := node.Decode((*fileChange)(fc)); err != nil {
		return &errors.UnmarshalError{
			Type:   fc.TypeName(),
			Data:   []byte(fmt.Sprintf("%v", node)),
			Reason: err.Error(),
		}
	}

	if err := fc.Validate(); err != nil {
		return &errors.UnmarshalError{
			Type:   fc.TypeName(),
			Data:   []byte(fmt.Sprintf("%v", node)),
			Reason: fmt.Sprintf("validation failed: %v", err),
		}
	}

	return nil
}
