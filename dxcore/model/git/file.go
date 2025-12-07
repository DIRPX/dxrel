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

const (
	// FilePathMaxLength is the maximum length for a file path.
	// This matches typical filesystem and Git limits (4096 bytes on most systems).
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

	// FileChangeTypeStr is the string representation of FileChangeType.
	FileChangeTypeStr = "type-changed"
)

// ParseFileChangeKind parses a string into a validated FileChangeKind value.
//
// ParseFileChangeKind applies normalization and validation to the input string.
// The normalization process trims leading and trailing whitespace via
// strings.TrimSpace and converts the string to lowercase via strings.ToLower.
//
// After normalization, the string is matched against the known kind names.
// If the normalized input matches a known kind name, the corresponding
// FileChangeKind constant is returned. If the input does not match any known
// name, ParseFileChangeKind returns FileChangeUnknown and an error.
//
// Example usage:
//
//	kind, err := git.ParseFileChangeKind("added")
//	// kind = FileChangeAdded, err = nil
//
//	kind, err := git.ParseFileChangeKind("  MODIFIED  ")
//	// kind = FileChangeModified, err = nil (normalized to lowercase, trimmed)
//
//	kind, err := git.ParseFileChangeKind("invalid")
//	// kind = FileChangeUnknown, err = error
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

// String returns the string representation of this FileChangeKind.
//
// This method implements the fmt.Stringer interface and satisfies the
// model.Loggable contract's String() requirement.
//
// The returned string is the canonical name for the kind ("unknown", "added",
// "modified", etc.), suitable for display, logging, and serialization.
//
// Examples:
//
//	FileChangeAdded.String()     // "added"
//	FileChangeModified.String()  // "modified"
//	FileChangeUnknown.String()   // "unknown"
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

// Redacted returns a safe representation of this FileChangeKind.
//
// This method implements the model.Loggable contract's Redacted() requirement.
// For FileChangeKind, the redacted output is identical to String() since
// change kinds do not contain sensitive information.
func (k FileChangeKind) Redacted() string {
	return k.String()
}

// TypeName returns the name of this type for error messages and debugging.
//
// This method implements the model.Identifiable contract.
func (k FileChangeKind) TypeName() string {
	return "FileChangeKind"
}

// IsZero reports whether this FileChangeKind is the zero value.
//
// This method implements the model.ZeroCheckable contract.
// FileChangeKind is considered zero if it equals FileChangeUnknown.
func (k FileChangeKind) IsZero() bool {
	return k == FileChangeUnknown
}

// Equal reports whether this FileChangeKind equals another FileChangeKind.
//
// Two FileChangeKind values are equal if they have the same numeric value.
func (k FileChangeKind) Equal(other FileChangeKind) bool {
	return k == other
}

// Validate checks whether this FileChangeKind is a known, valid value.
//
// This method implements the model.Validatable contract.
// Validation ensures the kind is one of the defined constants.
//
// Returns nil if the kind is valid, or an error if it's an unknown value.
func (k FileChangeKind) Validate() error {
	switch k {
	case FileChangeUnknown, FileChangeAdded, FileChangeModified,
		FileChangeDeleted, FileChangeRenamed, FileChangeCopied, FileChangeType:
		return nil
	default:
		return fmt.Errorf("invalid %s: %d", k.TypeName(), uint8(k))
	}
}

// MarshalJSON serializes this FileChangeKind to JSON.
//
// This method implements the json.Marshaler interface and the
// model.Serializable contract.
//
// The JSON representation is a string containing the kind name.
func (k FileChangeKind) MarshalJSON() ([]byte, error) {
	if err := k.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", k.TypeName(), err)
	}
	return json.Marshal(k.String())
}

// UnmarshalJSON deserializes a FileChangeKind from JSON.
//
// This method implements the json.Unmarshaler interface and the
// model.Serializable contract.
func (k *FileChangeKind) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("cannot unmarshal JSON into %s: %w", k.TypeName(), err)
	}

	parsed, err := ParseFileChangeKind(s)
	if err != nil {
		return fmt.Errorf("cannot parse %s from JSON: %w", k.TypeName(), err)
	}

	*k = parsed
	return nil
}

// MarshalYAML serializes this FileChangeKind to YAML.
//
// This method implements the yaml.Marshaler interface and the
// model.Serializable contract.
func (k FileChangeKind) MarshalYAML() (interface{}, error) {
	if err := k.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", k.TypeName(), err)
	}
	return k.String(), nil
}

// UnmarshalYAML deserializes a FileChangeKind from YAML.
//
// This method implements the yaml.Unmarshaler interface and the
// model.Serializable contract.
func (k *FileChangeKind) UnmarshalYAML(node *yaml.Node) error {
	var s string
	if err := node.Decode(&s); err != nil {
		return fmt.Errorf("cannot unmarshal YAML into %s: %w", k.TypeName(), err)
	}

	parsed, err := ParseFileChangeKind(s)
	if err != nil {
		return fmt.Errorf("cannot parse %s from YAML: %w", k.TypeName(), err)
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

// NewFileChange creates a new FileChange with the given Path and Kind.
//
// This is a convenience constructor for simple changes (added, modified, deleted).
// For renames and copies, construct FileChange directly and set OldPath.
//
// Example usage:
//
//	change, err := git.NewFileChange("src/main.go", git.FileChangeModified)
//	if err != nil {
//	    // handle error
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

// String returns the human-readable representation of the FileChange.
//
// This method implements the fmt.Stringer interface and satisfies the
// model.Loggable contract's String() requirement.
//
// Format varies based on change kind:
//   - Simple: "FileChange{Path:<path>, Kind:<kind>}"
//   - Rename/Copy: "FileChange{Path:<path>, OldPath:<oldpath>, Kind:<kind>}"
//
// Examples:
//
//	FileChange{Path: "main.go", Kind: FileChangeModified}.String()
//	// "FileChange{Path:main.go, Kind:modified}"
//
//	FileChange{Path: "new.go", OldPath: "old.go", Kind: FileChangeRenamed}.String()
//	// "FileChange{Path:new.go, OldPath:old.go, Kind:renamed}"
func (fc FileChange) String() string {
	if fc.OldPath != "" {
		return fmt.Sprintf("FileChange{Path:%s, OldPath:%s, Kind:%s}",
			fc.Path, fc.OldPath, fc.Kind.String())
	}
	return fmt.Sprintf("FileChange{Path:%s, Kind:%s}",
		fc.Path, fc.Kind.String())
}

// Redacted returns a safe representation of the FileChange suitable for
// production logging.
//
// This method implements the model.Loggable contract's Redacted() requirement.
// For FileChange, all components are safe to log (paths and kind are not
// sensitive), so Redacted() delegates to component Redacted() methods.
func (fc FileChange) Redacted() string {
	if fc.OldPath != "" {
		return fmt.Sprintf("FileChange{Path:%s, OldPath:%s, Kind:%s}",
			fc.Path, fc.OldPath, fc.Kind.Redacted())
	}
	return fmt.Sprintf("FileChange{Path:%s, Kind:%s}",
		fc.Path, fc.Kind.Redacted())
}

// TypeName returns the name of this type for error messages and debugging.
//
// This method implements the model.Identifiable contract.
func (fc FileChange) TypeName() string {
	return "FileChange"
}

// IsZero reports whether this FileChange is the zero value.
//
// This method implements the model.ZeroCheckable contract.
// A FileChange is considered zero if all fields are zero.
func (fc FileChange) IsZero() bool {
	return fc.Path == "" && fc.OldPath == "" && fc.Kind.IsZero()
}

// Equal reports whether this FileChange equals another FileChange.
//
// Two FileChanges are equal if all fields match (Path, OldPath, Kind).
func (fc FileChange) Equal(other FileChange) bool {
	return fc.Path == other.Path &&
		fc.OldPath == other.OldPath &&
		fc.Kind.Equal(other.Kind)
}

// Validate checks whether this FileChange satisfies all model contracts and
// invariants.
//
// This method implements the model.Validatable contract. Validation ensures:
//   - Path is non-empty and within length limits
//   - Path does not start with "/" (must be relative)
//   - Kind is valid (delegates to FileChangeKind.Validate)
//   - OldPath consistency: only present for renames and copies
//   - OldPath format: same rules as Path when present
//
// Returns nil if Changethe FileChange is valid, or a descriptive error if validation fails.
//
// Examples:
//
//	FileChange{Path: "main.go", Kind: FileChangeModified}.Validate()
//	// Returns: nil (valid)
//
//	FileChange{}.Validate()
//	// Returns: error "FileChange Path must not be empty"
//
//	FileChange{Path: "new.go", OldPath: "old.go", Kind: FileChangeModified}.Validate()
//	// Returns: error "FileChange OldPath should only be set for renamed/copied files"
func (fc FileChange) Validate() error {
	// Validate Path
	if fc.Path == "" {
		return fmt.Errorf("%s Path must not be empty", fc.TypeName())
	}
	if len(fc.Path) > FilePathMaxLength {
		return fmt.Errorf("%s Path exceeds maximum length of %d characters (got %d)",
			fc.TypeName(), FilePathMaxLength, len(fc.Path))
	}
	if strings.HasPrefix(fc.Path, "/") {
		return fmt.Errorf("%s Path must be relative (no leading slash): %q",
			fc.TypeName(), fc.Path)
	}

	// Validate Kind
	if err := fc.Kind.Validate(); err != nil {
		return fmt.Errorf("invalid %s Kind: %w", fc.TypeName(), err)
	}

	// Validate OldPath consistency
	if fc.OldPath != "" {
		// OldPath should only be set for renames and copies
		if fc.Kind != FileChangeRenamed && fc.Kind != FileChangeCopied {
			return fmt.Errorf("%s OldPath should only be set for renamed/copied files (got kind=%s)",
				fc.TypeName(), fc.Kind.String())
		}

		// Validate OldPath format
		if len(fc.OldPath) > FilePathMaxLength {
			return fmt.Errorf("%s OldPath exceeds maximum length of %d characters (got %d)",
				fc.TypeName(), FilePathMaxLength, len(fc.OldPath))
		}
		if strings.HasPrefix(fc.OldPath, "/") {
			return fmt.Errorf("%s OldPath must be relative (no leading slash): %q",
				fc.TypeName(), fc.OldPath)
		}
	} else {
		// Warn if OldPath is missing for rename/copy (but don't fail - might be partial data)
		// This is informational only, not a hard error
	}

	return nil
}

// MarshalJSON serializes this FileChange to JSON.
//
// This method implements the json.Marshaler interface and the
// model.Serializable contract.
//
// Before encoding, MarshalJSON calls Validate. If the FileChange is invalid,
// the validation error is returned and no JSON is produced.
func (fc FileChange) MarshalJSON() ([]byte, error) {
	if err := fc.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", fc.TypeName(), err)
	}

	// Use type alias to avoid infinite recursion
	type fileChange FileChange
	return json.Marshal(fileChange(fc))
}

// UnmarshalJSON deserializes a FileChange from JSON.
//
// This method implements the json.Unmarshaler interface and the
// model.Serializable contract.
//
// After unmarshaling, Validate is called to ensure the deserialized FileChange
// satisfies all invariants.
func (fc *FileChange) UnmarshalJSON(data []byte) error {
	type fileChange FileChange
	if err := json.Unmarshal(data, (*fileChange)(fc)); err != nil {
		return fmt.Errorf("cannot unmarshal JSON into %s: %w", fc.TypeName(), err)
	}

	if err := fc.Validate(); err != nil {
		return fmt.Errorf("unmarshaled %s is invalid: %w", fc.TypeName(), err)
	}

	return nil
}

// MarshalYAML serializes this FileChange to YAML.
//
// This method implements the yaml.Marshaler interface and the
// model.Serializable contract.
func (fc FileChange) MarshalYAML() (interface{}, error) {
	if err := fc.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", fc.TypeName(), err)
	}

	// Use type alias to avoid infinite recursion
	type fileChange FileChange
	return fileChange(fc), nil
}

// UnmarshalYAML deserializes a FileChange from YAML.
//
// This method implements the yaml.Unmarshaler interface and the
// model.Serializable contract.
func (fc *FileChange) UnmarshalYAML(node *yaml.Node) error {
	type fileChange FileChange
	if err := node.Decode((*fileChange)(fc)); err != nil {
		return fmt.Errorf("cannot unmarshal YAML into %s: %w", fc.TypeName(), err)
	}

	if err := fc.Validate(); err != nil {
		return fmt.Errorf("unmarshaled %s is invalid: %w", fc.TypeName(), err)
	}

	return nil
}
