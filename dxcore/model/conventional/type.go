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

package conventional

import (
	"encoding/json"
	"fmt"
	"strings"

	"dirpx.dev/dxrel/dxcore/model"
	"gopkg.in/yaml.v3"
)

// Type represents the conventional commit type as defined by the Conventional
// Commits specification version 1.0.0 and its commonly adopted extensions.
// Type categorizes the nature of changes in a commit, enabling automated
// tooling to determine semantic version bumps, generate changelogs, and
// enforce commit message conventions.
//
// This type implements the model.Model interface, providing validation,
// serialization to JSON and YAML, safe logging, type identification, and
// zero-value detection. The zero value of Type (0) corresponds to Feat,
// making the type usable with default initialization.
//
// The enumeration covers the core Conventional Commits types (feat and fix)
// along with widely adopted extensions for documentation, styling, refactoring,
// performance improvements, testing, build system changes, continuous
// integration, maintenance tasks, and commit reverts. Each constant provides
// semantic meaning that SHOULD be respected by commit authors and enforced
// by validation tooling.
//
// Type values serialize to their lowercase string representations in both
// JSON and YAML formats, ensuring compatibility with standard Conventional
// Commits parsers and tools. Deserialization accepts both uppercase and
// lowercase variants for flexibility, though lowercase is canonical.
//
// Example usage:
//
//	t := conventional.Feat
//	fmt.Println(t.String()) // Output: "feat"
//
//	var parsed conventional.Type
//	json.Unmarshal([]byte(`"fix"`), &parsed)
//	fmt.Println(parsed == conventional.Fix) // Output: true
type Type uint8

const (
	// Feat represents a commit that introduces a new feature to the codebase.
	// Features are user-visible functionality additions that warrant a minor
	// version bump in semantic versioning (x.Y.z). Examples include new API
	// endpoints, new command-line flags, new UI components, or new capabilities
	// exposed to end users or library consumers.
	//
	// Conventional Commits string: "feat"
	//
	// When to use Feat: Use Feat when adding wholly new functionality that
	// expands what the software can do. Do not use Feat for bug fixes
	// (use Fix instead), performance improvements that do not add features
	// (use Perf instead), or code reorganization without functional changes
	// (use Refactor instead).
	Feat Type = iota

	// Fix represents a commit that patches a bug or defect in the codebase.
	// Fixes correct unintended behavior, crashes, incorrect outputs, or other
	// defects. Bug fixes warrant a patch version bump in semantic versioning
	// (x.y.Z). Examples include null pointer dereference fixes, off-by-one
	// error corrections, incorrect calculation fixes, or handling of edge
	// cases that previously caused failures.
	//
	// Conventional Commits string: "fix"
	//
	// When to use Fix: Use Fix when correcting behavior that does not match
	// specifications, documentation, or user expectations. Do not use Fix for
	// performance improvements without bug correction (use Perf instead) or
	// for adding missing features (use Feat instead).
	Fix

	// Docs represents a commit that changes documentation only, without
	// modifying source code, tests, or build configuration. Documentation
	// changes include updates to README files, API documentation, code
	// comments (when standalone), user guides, tutorials, or any other written
	// materials intended to explain the software to users or developers.
	//
	// Conventional Commits string: "docs"
	//
	// When to use Docs: Use Docs when modifying only documentation files or
	// standalone comment blocks. Do not use Docs when documentation changes
	// accompany code changes (use the appropriate code change type and mention
	// documentation updates in the commit body).
	Docs

	// Style represents a commit that changes code formatting, whitespace,
	// missing semicolons, or other style-related aspects without affecting
	// the code's meaning or behavior. Style changes are typically enforced
	// by linters or formatters and do not alter logic, control flow, or
	// output.
	//
	// Conventional Commits string: "style"
	//
	// When to use Style: Use Style when reformatting code with tools like
	// gofmt, prettier, or black; fixing linter warnings about formatting;
	// adjusting indentation or line wrapping; or adding/removing trailing
	// commas or semicolons that do not change behavior. Do not use Style
	// when refactoring logic (use Refactor instead).
	Style

	// Refactor represents a commit that restructures existing code without
	// changing its external behavior. Refactoring improves code quality,
	// readability, maintainability, or structure without fixing bugs or
	// adding features. Examples include extracting functions, renaming
	// variables for clarity, simplifying control flow, removing duplication,
	// or reorganizing module boundaries.
	//
	// Conventional Commits string: "refactor"
	//
	// When to use Refactor: Use Refactor when changing code organization or
	// implementation details while preserving observable behavior. Do not use
	// Refactor when fixing bugs (use Fix instead) or when behavior changes
	// are user-visible (use Feat instead).
	Refactor

	// Perf represents a commit that improves performance without adding
	// features or fixing user-visible bugs. Performance improvements reduce
	// execution time, memory usage, network bandwidth, or other resource
	// consumption. Examples include algorithmic optimizations, caching,
	// lazy evaluation, batch processing, or database query optimization.
	//
	// Conventional Commits string: "perf"
	//
	// When to use Perf: Use Perf when optimizing performance characteristics
	// measurably. The commit message SHOULD include benchmarks or profiling
	// data demonstrating the improvement. Do not use Perf for refactoring
	// without measured performance gains (use Refactor instead).
	Perf

	// Test represents a commit that adds missing tests, corrects existing
	// tests, or improves test coverage without changing production code
	// (except for making code more testable). Test changes ensure correctness
	// and prevent regressions. Examples include adding unit tests, integration
	// tests, end-to-end tests, property-based tests, or fixing flaky tests.
	//
	// Conventional Commits string: "test"
	//
	// When to use Test: Use Test when modifying only test code, test fixtures,
	// or test configuration. Do not use Test when tests accompany functional
	// changes (use the appropriate functional type and mention tests in the
	// commit body).
	Test

	// Build represents a commit that affects the build system, build scripts,
	// or external dependencies. Build changes modify how the software is
	// compiled, packaged, or distributed. Examples include updating build
	// tool versions, modifying Makefiles or build scripts, changing compiler
	// flags, updating dependency declarations in go.mod, package.json, or
	// requirements.txt, or adjusting module bundling configuration.
	//
	// Conventional Commits string: "build"
	//
	// When to use Build: Use Build when changing the build process or
	// dependency graph. Do not use Build for source code changes (use the
	// appropriate functional type) or for CI/CD pipeline changes (use CI
	// instead).
	Build

	// CI represents a commit that changes continuous integration or continuous
	// deployment configuration, scripts, or pipeline definitions. CI changes
	// modify how automated builds, tests, or deployments execute in CI/CD
	// systems. Examples include updating GitHub Actions workflows, modifying
	// Jenkins pipelines, changing Travis CI or CircleCI configuration, or
	// adjusting deployment automation scripts.
	//
	// Conventional Commits string: "ci"
	//
	// When to use CI: Use CI when modifying CI/CD configuration files or
	// scripts that run in automated pipelines. Do not use CI for build system
	// changes that affect local development (use Build instead) or for test
	// code changes (use Test instead).
	CI

	// Chore represents a commit that makes other changes not covered by more
	// specific types. Chore commits typically involve repository maintenance,
	// tooling updates, configuration adjustments, or housekeeping tasks that
	// do not modify production source code or tests. Examples include updating
	// .gitignore, modifying editor configuration, updating copyright headers,
	// bumping version numbers, or regenerating auto-generated files.
	//
	// Conventional Commits string: "chore"
	//
	// When to use Chore: Use Chore as a fallback when no other type fits.
	// Prefer more specific types when applicable (for example, use Docs for
	// documentation, Build for dependencies, or CI for pipeline changes).
	// Chore SHOULD be the least frequently used type in a well-categorized
	// commit history.
	Chore

	// Revert represents a commit that reverts one or more previous commits,
	// undoing their changes. Reverts restore the codebase to a prior state,
	// typically because a previous commit introduced a bug, broke functionality,
	// or caused other issues. The commit message SHOULD reference the reverted
	// commit hash and explain why the revert was necessary.
	//
	// Conventional Commits string: "revert"
	//
	// When to use Revert: Use Revert when undoing previous commits. The commit
	// message SHOULD follow the format "revert: <original commit subject>" and
	// include details about which commit is being reverted and why. Automated
	// git revert commands typically generate appropriate messages.
	Revert

	// maxType is an internal sentinel value representing the upper bound of
	// valid Type values. It is not a valid Type and MUST NOT be used in user
	// code. This constant enables validation logic to detect out-of-range
	// Type values efficiently.
	maxType
)

// String constants for each Type value, enabling type-safe string comparisons
// in switch statements and other contexts. These constants represent the
// canonical lowercase form as specified by the Conventional Commits specification.
//
// Example usage in switch statements:
//
//	switch typeStr {
//	case conventional.FeatStr:
//	    // Handle feat
//	case conventional.FixStr:
//	    // Handle fix
//	}
const (
	FeatStr     = "feat"
	FixStr      = "fix"
	DocsStr     = "docs"
	StyleStr    = "style"
	RefactorStr = "refactor"
	PerfStr     = "perf"
	TestStr     = "test"
	BuildStr    = "build"
	CIStr       = "ci"
	ChoreStr    = "chore"
	RevertStr   = "revert"
)

// ParseType parses a string into a Type value, normalizing and validating the
// input before matching against canonical type names. This function provides a
// unified parsing entry point for converting external string representations
// into Type values with comprehensive input validation.
//
// ParseType recognizes all Conventional Commits type names: "feat", "fix",
// "docs", "style", "refactor", "perf", "test", "build", "ci", "chore", and
// "revert". The input undergoes normalization before matching: leading and
// trailing whitespace is removed using strings.TrimSpace, and the result is
// converted to lowercase using strings.ToLower. This ensures that inputs like
// "  FEAT  ", "Feat", and "feat" all parse to the same Type value.
//
// ParseType returns an error in the following cases: if the input is an empty
// string, if the input contains only whitespace characters, or if the normalized
// input does not match any known type name. The error message includes the
// original invalid input (before normalization) to aid debugging and provide
// clear feedback to users about what they provided.
//
// Callers MUST check the returned error before using the Type value. The zero
// value returned on error (Type(0), which equals Feat) MUST NOT be used when
// an error is returned, as it does not represent a successfully parsed value.
//
// This function is pure and has no side effects. It is safe to call concurrently
// from multiple goroutines. The normalization process ensures consistent behavior
// regardless of input casing or surrounding whitespace.
//
// Example:
//
//	t, err := conventional.ParseType("  FIX  ")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(t == conventional.Fix) // Output: true
func ParseType(s string) (Type, error) {
	// Validate that input is not empty before normalization
	if s == "" {
		return 0, fmt.Errorf("Type string cannot be empty")
	}

	// Normalize input: trim whitespace and convert to lowercase
	normalized := strings.ToLower(strings.TrimSpace(s))

	// Check if normalized result is empty (input was only whitespace)
	if normalized == "" {
		return 0, fmt.Errorf("Type string cannot contain only whitespace: %q", s)
	}

	switch normalized {
	case FeatStr:
		return Feat, nil
	case FixStr:
		return Fix, nil
	case DocsStr:
		return Docs, nil
	case StyleStr:
		return Style, nil
	case RefactorStr:
		return Refactor, nil
	case PerfStr:
		return Perf, nil
	case TestStr:
		return Test, nil
	case BuildStr:
		return Build, nil
	case CIStr:
		return CI, nil
	case ChoreStr:
		return Chore, nil
	case RevertStr:
		return Revert, nil
	default:
		return 0, fmt.Errorf("unknown Type: %q (normalized: %q)", s, normalized)
	}
}

// String returns the lowercase string representation of the Type as specified
// by the Conventional Commits specification. This method satisfies the
// model.Loggable interface's String requirement, providing a human-readable
// representation suitable for display and debugging.
//
// The returned strings are: "feat", "fix", "docs", "style", "refactor",
// "perf", "test", "build", "ci", "chore", "revert". If the Type value is
// invalid (out of range), String returns "unknown" to prevent crashes or
// silent failures.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently. The returned string is always a constant
// literal, ensuring zero allocations.
//
// Example:
//
//	t := conventional.Feat
//	fmt.Println(t.String()) // Output: "feat"
func (t Type) String() string {
	switch t {
	case Feat:
		return FeatStr
	case Fix:
		return FixStr
	case Docs:
		return DocsStr
	case Style:
		return StyleStr
	case Refactor:
		return RefactorStr
	case Perf:
		return PerfStr
	case Test:
		return TestStr
	case Build:
		return BuildStr
	case CI:
		return CIStr
	case Chore:
		return ChoreStr
	case Revert:
		return RevertStr
	default:
		return "unknown"
	}
}

// Redacted returns a safe string representation suitable for logging in
// production environments. For Type, which contains no sensitive data, Redacted
// is identical to String and returns the lowercase type name.
//
// This method satisfies the model.Loggable interface's Redacted requirement,
// ensuring that Type can be safely logged without risk of exposing sensitive
// information. Since commit types are not sensitive, no masking or redaction
// is necessary.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently.
//
// Example:
//
//	t := conventional.Fix
//	log.Info("processing commit", "type", t.Redacted()) // Safe for production logs
func (t Type) Redacted() string {
	return t.String()
}

// TypeName returns the canonical name of this model type for debugging,
// logging, and reflection purposes. This method satisfies the model.Identifiable
// interface's TypeName requirement.
//
// The returned value is always the constant string "Type", uniquely identifying
// this type within the dxrel domain. The name follows CamelCase convention and
// omits the package prefix as required by the Model contract.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently. The returned string is a constant literal,
// ensuring zero allocations.
func (t Type) TypeName() string {
	return "Type"
}

// IsZero reports whether this Type instance is in a zero or empty state.
// For Type, the zero value (0) corresponds to Feat, which is a valid and
// meaningful type. Therefore, IsZero always returns false because there is
// no distinguished "empty" or "uninitialized" Type value.
//
// This method satisfies the model.ZeroCheckable interface's IsZero requirement.
// The semantics differ from typical zero-value checking because Type's zero
// value represents a legitimate commit type rather than absence of data.
//
// Callers needing to distinguish "not set" from "explicitly set to Feat" SHOULD
// use a pointer type (*Type) where nil indicates "not set" and non-nil indicates
// a set value.
//
// This method MUST NOT mutate the receiver, MUST NOT have side effects, and
// MUST be safe to call concurrently.
//
// Example:
//
//	var t conventional.Type // Zero value, equals Feat
//	fmt.Println(t.IsZero()) // Output: false (Feat is valid)
func (t Type) IsZero() bool {
	return false
}

// Validate checks that the Type value is within the valid range of defined
// constants. This method satisfies the model.Validatable interface's Validate
// requirement, enforcing data integrity.
//
// Validate returns nil if the Type is one of the defined constants (Feat through
// Revert). It returns an error if the Type value is out of range, which can
// occur through type conversions, unsafe operations, or deserialization bugs.
//
// This method MUST be fast, deterministic, and idempotent. It MUST NOT mutate
// the receiver, MUST NOT have side effects, and MUST be safe to call concurrently.
// Validation does not perform I/O or allocate memory except when constructing
// error messages for invalid values.
//
// Callers SHOULD invoke Validate after deserializing Type from external sources
// (JSON, YAML, databases) to ensure data integrity. The ToJSON, ToYAML, FromJSON,
// and FromYAML helper functions automatically call Validate to enforce this
// contract.
//
// Example:
//
//	t := conventional.Feat
//	if err := t.Validate(); err != nil {
//	    log.Error("invalid type", "error", err)
//	}
func (t Type) Validate() error {
	if t >= maxType {
		return fmt.Errorf("Type value %d is out of valid range [0, %d)", t, maxType)
	}
	return nil
}

// MarshalJSON implements json.Marshaler, serializing the Type to its lowercase
// string representation as a JSON string. This method satisfies part of the
// model.Serializable interface requirement.
//
// MarshalJSON first validates that the Type is in the valid range by calling
// Validate. If validation fails, marshaling fails with the validation error,
// preventing invalid data from being serialized. If validation succeeds, the
// Type is converted to its string representation using String and marshaled
// as a JSON string.
//
// The output format is compatible with the Conventional Commits specification
// and standard tooling. For example, Feat marshals to the JSON string "feat",
// Fix marshals to "fix", and so on.
//
// This method MUST NOT mutate the receiver except as required by the json.Marshaler
// interface contract. It MUST be safe to call concurrently on immutable receivers.
//
// Example:
//
//	t := conventional.Feat
//	data, _ := json.Marshal(t)
//	fmt.Println(string(data)) // Output: "feat"
func (t Type) MarshalJSON() ([]byte, error) {
	if err := t.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", t.TypeName(), err)
	}
	return json.Marshal(t.String())
}

// UnmarshalJSON implements json.Unmarshaler, deserializing a JSON string into
// a Type value. This method satisfies part of the model.Serializable interface
// requirement.
//
// UnmarshalJSON accepts JSON strings containing lowercase type names ("feat",
// "fix", "docs", etc.) as specified by the Conventional Commits specification.
// It also accepts uppercase variants ("FEAT", "FIX") for flexibility, though
// lowercase is canonical and SHOULD be preferred in serialized data.
//
// After unmarshaling, Validate is called to ensure the resulting Type is valid.
// If the input string does not match any known type name, unmarshaling fails
// with an error indicating the unknown type. This fail-fast behavior prevents
// invalid data from entering the system through external inputs.
//
// The method mutates the receiver to store the unmarshaled value. It is not
// safe for concurrent use during unmarshaling, though the resulting Type value
// is safe for concurrent reads after unmarshaling completes.
//
// Example:
//
//	var t conventional.Type
//	json.Unmarshal([]byte(`"fix"`), &t)
//	fmt.Println(t == conventional.Fix) // Output: true
func (t *Type) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("cannot unmarshal JSON: %w", err)
	}

	parsed, err := ParseType(s)
	if err != nil {
		return fmt.Errorf("unmarshaled model is invalid: %w", err)
	}

	*t = parsed
	return t.Validate()
}

// MarshalYAML implements yaml.Marshaler, serializing the Type to its lowercase
// string representation for YAML encoding. This method satisfies part of the
// model.Serializable interface requirement.
//
// MarshalYAML first validates that the Type is in the valid range by calling
// Validate. If validation fails, marshaling fails with the validation error,
// preventing invalid data from being serialized. If validation succeeds, the
// Type is converted to its string representation using String.
//
// The output format is compatible with the Conventional Commits specification.
// For example, Feat marshals to the YAML scalar "feat", Fix marshals to "fix",
// and so on.
//
// This method MUST NOT mutate the receiver except as required by the yaml.Marshaler
// interface contract. It MUST be safe to call concurrently on immutable receivers.
//
// Example:
//
//	t := conventional.Perf
//	data, _ := yaml.Marshal(t)
//	fmt.Println(string(data)) // Output: "perf\n"
func (t Type) MarshalYAML() (interface{}, error) {
	if err := t.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", t.TypeName(), err)
	}
	return t.String(), nil
}

// UnmarshalYAML implements yaml.Unmarshaler, deserializing a YAML scalar into
// a Type value. This method satisfies part of the model.Serializable interface
// requirement.
//
// UnmarshalYAML accepts YAML scalars containing lowercase type names ("feat",
// "fix", "docs", etc.) as specified by the Conventional Commits specification.
// It also accepts uppercase variants for flexibility, though lowercase is
// canonical and SHOULD be preferred in YAML files.
//
// After unmarshaling, Validate is called to ensure the resulting Type is valid.
// If the input string does not match any known type name, unmarshaling fails
// with an error indicating the unknown type. This fail-fast behavior prevents
// invalid configuration data from corrupting system state.
//
// The method mutates the receiver to store the unmarshaled value. It is not
// safe for concurrent use during unmarshaling, though the resulting Type value
// is safe for concurrent reads after unmarshaling completes.
//
// Example:
//
//	var t conventional.Type
//	yaml.Unmarshal([]byte("test"), &t)
//	fmt.Println(t == conventional.Test) // Output: true
func (t *Type) UnmarshalYAML(node *yaml.Node) error {
	var s string
	if err := node.Decode(&s); err != nil {
		return fmt.Errorf("cannot unmarshal YAML: %w", err)
	}

	parsed, err := ParseType(s)
	if err != nil {
		return fmt.Errorf("unmarshaled model is invalid: %w", err)
	}

	*t = parsed
	return t.Validate()
}

// Compile-time verification that Type implements model.Model interface.
var _ model.Model = (*Type)(nil)
