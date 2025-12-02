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

// Package model defines the core contracts and interfaces that all dxrel
// domain model types MUST implement to ensure consistency, type safety, and
// proper behavior across the entire system.
//
// Every domain type representing business entities (such as Commit, Version,
// Plan, Module, etc.) SHOULD implement the Model interface or its constituent
// parts (Validatable, Serializable, Loggable, Identifiable, ZeroCheckable).
// These interfaces establish a common contract for validation, serialization,
// logging, and identity that enables generic operations and guarantees safety
// at compile time.
//
// The contracts defined in this package prioritize data integrity, security,
// and debuggability. Validation ensures that invalid states cannot be
// constructed or persisted. Serialization provides round-trip guarantees for
// configuration files and API payloads. Loggable protects sensitive data
// (PII, secrets) from accidental exposure in logs. Identifiable enables
// reflection and structured logging. ZeroCheckable supports optional field
// detection and default value handling.
//
// Unless explicitly documented otherwise, implementations are not thread-safe
// for concurrent mutation. Most model types are designed as immutable value
// types, making them naturally safe for concurrent read access. Callers MUST
// synchronize any concurrent writes to mutable instances.
//
// Types implementing Model can be used with the generic helper functions
// provided in this package, such as ValidateAll, FilterZero, ToJSON, ToYAML,
// Clone, and Equal. These helpers rely on the Model contract and will fail
// at compile time if applied to types that do not implement Model.
package model

import (
	"encoding/json"

	"gopkg.in/yaml.v3"
)

// Model is the root interface combining all fundamental contracts required
// for dxrel domain types. Any type implementing Model gains automatic support
// for validation, serialization to JSON and YAML, safe logging with PII
// protection, type identification, and zero-value detection.
//
// Implementations MUST satisfy all embedded interfaces: Validatable ensures
// data integrity by checking invariants; Serializable provides round-trip
// JSON and YAML encoding; Loggable offers both safe (redacted) and unsafe
// (full) string representations; Identifiable supplies a canonical type name;
// and ZeroCheckable detects empty or uninitialized instances.
//
// All domain types MUST implement Model to participate in generic operations.
// Failure to implement Model will result in compile-time errors when using
// functions that constrain their type parameters to Model. This design
// ensures that all domain objects share a common baseline of safety and
// functionality, preventing runtime surprises and enabling predictable
// behavior across the system.
//
// Model instances are generally treated as immutable value types. Methods
// defined on Model SHOULD NOT mutate the receiver unless explicitly
// documented. Concurrent reads are safe; concurrent writes require external
// synchronization.
//
// Example implementation:
//
//	type MyModel struct {
//	    Field string
//	}
//
//	func (m MyModel) Validate() error {
//	    if m.Field == "" {
//	        return errors.New("field required")
//	    }
//	    return nil
//	}
//
//	func (m MyModel) TypeName() string { return "MyModel" }
//	func (m MyModel) IsZero() bool { return m.Field == "" }
//	func (m MyModel) Redacted() string { return "MyModel{...}" }
//	func (m MyModel) String() string { return "MyModel{Field:" + m.Field + "}" }
//	// ... MarshalJSON, UnmarshalJSON, MarshalYAML, UnmarshalYAML
//
//	var _ Model = (*MyModel)(nil)  // Compile-time check
type Model interface {
	Validatable
	Serializable
	Loggable
	Identifiable
	ZeroCheckable
}

// Validatable defines the contract for types that validate their own state
// to ensure data integrity. Every model type MUST implement Validate to
// verify that all invariants hold and that the instance is in a consistent
// state suitable for use in business logic, persistence, or transmission.
//
// The Validate method MUST check all required fields for non-empty or
// non-zero values, verify cross-field consistency (for example, ensuring
// that a "Before" version is less than an "After" version), recursively
// validate any nested objects by calling their Validate methods, and return
// nil if and only if the instance is fully valid. When validation fails,
// the returned error MUST describe what is invalid in a way that helps
// callers diagnose and fix the problem. Generic error messages such as
// "validation failed" are discouraged; prefer specific messages like
// "Commit.Hash MUST NOT be empty" or "Step.Index MUST be non-negative".
//
// Validate MUST be fast, avoiding expensive operations such as I/O, network
// calls, or database queries. It MUST be deterministic, producing the same
// result when called multiple times on the same instance. It MUST be
// idempotent, meaning that calling Validate multiple times does not change
// the outcome or the state of the receiver. Validate MUST NOT mutate the
// receiver, MUST NOT have side effects such as logging or emitting metrics,
// and MUST NOT depend on external mutable state.
//
// Callers SHOULD invoke Validate at critical boundaries: immediately after
// unmarshaling data from JSON or YAML to ensure that external input is valid;
// after constructing instances from user input to catch errors early; before
// persisting to storage to prevent invalid data from being saved; before
// sending over the network to avoid transmitting bad data; and at any API
// boundary where data crosses trust or ownership boundaries.
//
// If Validate is called on a zero-value instance (an instance with all fields
// set to their type's zero value), it SHOULD typically return an error unless
// the zero value represents a valid state. Most domain objects require at
// least one non-zero field to be meaningful.
//
// Example:
//
//	type User struct {
//	    Name  string
//	    Email string
//	}
//
//	func (u User) Validate() error {
//	    if u.Name == "" {
//	        return errors.New("User.Name must not be empty")
//	    }
//	    if u.Email == "" {
//	        return errors.New("User.Email must not be empty")
//	    }
//	    if !strings.Contains(u.Email, "@") {
//	        return fmt.Errorf("User.Email %q is not a valid email", u.Email)
//	    }
//	    return nil
//	}
type Validatable interface {
	// Validate checks that the instance satisfies all invariants and is
	// ready for use. It returns nil if the instance is valid, or a
	// descriptive error explaining what is wrong if validation fails.
	//
	// This method MUST NOT mutate the receiver and MUST NOT have side
	// effects. It MUST be safe to call concurrently with other reads
	// but not with concurrent writes.
	Validate() error
}

// Serializable defines the contract for types that can be serialized to and
// deserialized from JSON and YAML formats. All model types MUST support both
// formats to enable configuration files (typically YAML), API request and
// response bodies (typically JSON), logging and debugging output, and
// persistence to storage systems.
//
// Implementations MUST call Validate before marshaling to ensure that only
// valid instances are serialized. This prevents invalid data from leaking
// into configuration files, API responses, or logs. If the instance fails
// validation, the marshal method MUST return the validation error rather
// than serializing the invalid state. Similarly, implementations MUST call
// Validate after unmarshaling to ensure that deserialized data meets all
// invariants. If the deserialized instance is invalid, the unmarshal method
// MUST return the validation error and leave the receiver in a well-defined
// state (typically its zero value or the partially unmarshaled value, but
// callers MUST NOT use it).
//
// Both JSON and YAML formats MUST be handled consistently. A value serialized
// to JSON and then deserialized MUST equal the original value, and the same
// MUST hold for YAML. All fields SHOULD be preserved during a round-trip
// unless explicitly documented otherwise. Fields marked as "omitempty" or
// similar tags MAY be omitted if they hold zero values, but deserializing
// such data MUST reconstruct a semantically equivalent instance.
//
// Marshal methods (MarshalJSON, MarshalYAML) are safe for concurrent use on
// immutable receivers because they do not mutate the receiver. However,
// unmarshal methods (UnmarshalJSON, UnmarshalYAML) mutate the receiver and
// are not safe for concurrent use. Callers MUST ensure exclusive access to
// the receiver during unmarshaling.
//
// Implementations SHOULD use the "type alias" pattern to avoid infinite
// recursion: define a local type alias to the model type, cast the receiver
// to the alias, and delegate to the standard library's marshal or unmarshal
// function. This avoids re-entering the custom method and keeps the code
// simple.
//
// Example:
//
//	func (m MyModel) MarshalJSON() ([]byte, error) {
//	    if err := m.Validate(); err != nil {
//	        return nil, fmt.Errorf("cannot marshal invalid %s: %w", m.TypeName(), err)
//	    }
//	    type alias MyModel
//	    return json.Marshal((alias)(m))
//	}
//
//	func (m *MyModel) UnmarshalJSON(data []byte) error {
//	    type alias MyModel
//	    if err := json.Unmarshal(data, (*alias)(m)); err != nil {
//	        return &errors.UnmarshalError{
//	            Type:   "MyModel",
//	            Data:   data,
//	            Reason: err.Error(),
//	        }
//	    }
//	    if err := m.Validate(); err != nil {
//	        return fmt.Errorf("unmarshaled MyModel is invalid: %w", err)
//	    }
//	    return nil
//	}
//
//	func (m MyModel) MarshalYAML() (interface{}, error) {
//	    if err := m.Validate(); err != nil {
//	        return nil, fmt.Errorf("cannot marshal invalid %s: %w", m.TypeName(), err)
//	    }
//	    type alias MyModel
//	    return (alias)(m), nil
//	}
//
//	func (m *MyModel) UnmarshalYAML(node *yaml.Node) error {
//	    type alias MyModel
//	    if err := node.Decode((*alias)(m)); err != nil {
//	        return &errors.UnmarshalError{
//	            Type:   "MyModel",
//	            Data:   nil,
//	            Reason: err.Error(),
//	        }
//	    }
//	    if err := m.Validate(); err != nil {
//	        return fmt.Errorf("unmarshaled MyModel is invalid: %w", err)
//	    }
//	    return nil
//	}
type Serializable interface {
	json.Marshaler
	json.Unmarshaler
	yaml.Marshaler
	yaml.Unmarshaler
}

// Loggable defines the contract for types that provide safe string
// representations for logging and debugging. All model types MUST implement
// Loggable to prevent accidental exposure of sensitive data such as passwords,
// authentication tokens, API keys, and personally identifiable information
// (PII) in application logs.
//
// The Redacted method returns a string representation suitable for production
// logging. It MUST hide or mask sensitive fields while preserving enough
// information for debugging and troubleshooting. For example, an email address
// might be redacted as "u***@example.com" to show the domain while hiding the
// local part. Passwords, tokens, and secrets MUST be completely hidden,
// typically shown as "[REDACTED]" or similar. Non-sensitive fields can be
// shown in full. The redacted representation SHOULD include the type name and
// key identifying information to help correlate log entries with specific
// instances.
//
// Redacted MUST be fast because it is called frequently during logging. It
// SHOULD avoid allocations where possible and MUST NOT perform I/O or other
// expensive operations. It MUST be safe to call concurrently and MUST NOT
// mutate the receiver or have side effects.
//
// The String method returns a human-readable representation that MAY include
// sensitive data. It is intended for use in development, debugging, and
// internal tooling where full visibility is needed and acceptable. String
// MUST NEVER be used for production logging; always use Redacted instead.
// The distinction between String and Redacted is critical: String is for
// humans in controlled environments, Redacted is for logs that might be
// stored, transmitted, or shared.
//
// Implementations SHOULD be consistent: the same instance SHOULD always
// produce the same redacted and string representations (barring changes to
// the instance's fields). Redacted and String SHOULD both be human-readable,
// but Redacted MUST prioritize safety over completeness.
//
// If a type contains nested objects that are also Loggable, Redacted SHOULD
// call Redacted on those nested objects to ensure consistent redaction
// throughout the object graph.
//
// Example:
//
//	type User struct {
//	    ID       string
//	    Email    string
//	    Password string
//	}
//
//	func (u User) Redacted() string {
//	    return fmt.Sprintf("User{ID:%s, Email:%s, Password:[REDACTED]}",
//	        u.ID, redactEmail(u.Email))
//	}
//
//	func (u User) String() string {
//	    return fmt.Sprintf("User{ID:%s, Email:%s, Password:%s}",
//	        u.ID, u.Email, u.Password)  // UNSAFE: contains password
//	}
//
//	func redactEmail(email string) string {
//	    if idx := strings.IndexByte(email, '@'); idx > 0 {
//	        if idx == 1 {
//	            return "*@" + email[idx+1:]
//	        }
//	        return string(email[0]) + "***@" + email[idx+1:]
//	    }
//	    return "[INVALID]"
//	}
type Loggable interface {
	// Redacted returns a safe string representation suitable for logging in
	// production. This method MUST redact or mask all sensitive fields
	// (passwords, tokens, PII) while preserving enough information for
	// debugging.
	//
	// Use this method for application logs, error messages shown to users,
	// and debugging output in production environments.
	//
	// This method MUST NOT mutate the receiver, MUST NOT have side effects,
	// and MUST be safe to call concurrently.
	Redacted() string

	// String returns a human-readable representation of the instance. This
	// method MAY include sensitive data and MUST NOT be used for production
	// logging. Use Redacted instead for logging.
	//
	// String is primarily for development debugging, test assertions, and
	// internal tooling where full visibility is acceptable.
	//
	// This method MUST NOT mutate the receiver, MUST NOT have side effects,
	// and MUST be safe to call concurrently.
	String() string
}

// Identifiable defines the contract for types that can identify themselves
// by a canonical type name. All model types MUST provide a type name to
// enable debugging, logging, reflection, and schema evolution tracking.
//
// The type name returned by TypeName MUST be constant for a given type,
// meaning that all instances of the same type MUST return the same name.
// The name MUST be unique within the dxrel domain to avoid ambiguity. It
// SHOULD follow CamelCase convention (for example, "Commit", "ModulePlan",
// "VersionRange") and MUST NOT include a package prefix. The name identifies
// the type, not the instance, so it SHOULD NOT vary based on the instance's
// field values.
//
// Type names are used in several contexts: structured logging systems use
// the type name as a field to categorize log entries; error messages include
// the type name to clarify what kind of object failed validation or
// processing; metrics and tracing systems use type names to aggregate data
// by type; and schema versioning or migration logic MAY use type names to
// route objects to appropriate handlers.
//
// TypeName MUST be fast and MUST NOT allocate memory. It SHOULD typically
// return a string constant. It MUST NOT have side effects and MUST be safe
// to call concurrently.
//
// Example:
//
//	type User struct { ... }
//
//	func (u User) TypeName() string { return "User" }
type Identifiable interface {
	// TypeName returns the canonical name of this model type. The name MUST
	// be constant for the type, unique within dxrel, in CamelCase, and
	// without a package prefix.
	//
	// This method MUST NOT mutate the receiver, MUST NOT have side effects,
	// and MUST be safe to call concurrently. It SHOULD return a string
	// constant.
	TypeName() string
}

// ZeroCheckable defines the contract for types that can report whether they
// are in a zero or empty state. This enables optional field detection,
// default value handling, and conditional logic based on whether an instance
// contains meaningful data.
//
// An instance is considered zero if all of its fields are at their type's
// zero value, no meaningful data is present, and the instance would fail
// validation if Validate were called. For example, a Commit with an empty
// Hash and empty Subject is zero. A Version with Major, Minor, and Patch
// all set to 0 and empty Prerelease and Metadata is zero.
//
// IsZero MUST return true if and only if the instance is semantically empty.
// For types with a single field, this typically means checking if that field
// is zero. For types with multiple fields, IsZero SHOULD return true only if
// all fields are zero (logical AND). For types with optional fields, IsZero
// SHOULD ignore optional fields or treat them as zero if they are not set.
//
// IsZero is used to filter slices (removing zero values), detect whether
// optional configuration fields have been set, short-circuit processing when
// an instance is empty, and provide better error messages (for example,
// "no commits provided" instead of "validation failed").
//
// IsZero MUST be fast and MUST NOT allocate memory. It MUST be deterministic
// and idempotent. It MUST NOT have side effects and MUST be safe to call
// concurrently.
//
// Example:
//
//	type User struct {
//	    Name  string
//	    Email string
//	}
//
//	func (u User) IsZero() bool {
//	    return u.Name == "" && u.Email == ""
//	}
type ZeroCheckable interface {
	// IsZero reports whether this instance is in a zero or empty state,
	// meaning it contains no meaningful data.
	//
	// This method MUST NOT mutate the receiver, MUST NOT have side effects,
	// and MUST be safe to call concurrently.
	IsZero() bool
}

// Comparable defines the contract for types that can be compared for equality.
// This interface is optional but recommended for value types that require
// equality testing in tests, assertions, or business logic.
//
// The Equal method MUST be reflexive (x.Equal(x) is always true), symmetric
// (x.Equal(y) implies y.Equal(x)), transitive (if x.Equal(y) and y.Equal(z),
// then x.Equal(z)), and consistent (multiple calls with the same arguments
// return the same result). Equal MUST return false when comparing to a zero
// value of T or when comparing instances of different types (though the type
// parameter T enforces type safety at compile time).
//
// Equal SHOULD compare all semantically significant fields. Internal or
// cached fields that do not affect the logical value SHOULD be ignored. For
// example, a cached hash code or a lazily computed field SHOULD NOT influence
// equality. Nested objects SHOULD be compared using deep equality, recursively
// calling Equal if they are Comparable or using other appropriate comparison
// logic.
//
// Equal MUST NOT mutate the receiver or the argument, MUST NOT have side
// effects, and MUST be safe to call concurrently.
//
// Example:
//
//	func (u User) Equal(other User) bool {
//	    return u.Name == other.Name && u.Email == other.Email
//	}
type Comparable[T any] interface {
	// Equal reports whether this instance is equal to another instance of
	// the same type. It returns true if both instances represent the same
	// logical value, false otherwise.
	//
	// This method MUST NOT mutate the receiver or the argument, MUST NOT
	// have side effects, and MUST be safe to call concurrently.
	Equal(other T) bool
}

// Cloneable defines the contract for types that can create deep copies of
// themselves. This interface is optional but recommended for mutable types
// or types containing references to shared data structures.
//
// The Clone method MUST create a deep copy, meaning that the returned instance
// shares no references with the original. Modifying the clone MUST NOT affect
// the original, and vice versa. All fields MUST be copied exactly, preserving
// the logical value. The cloned instance MUST be valid (it MUST pass
// Validate) if the original is valid. Clone MUST be idempotent: cloning a
// clone produces an instance equal to the first clone.
//
// Clone SHOULD be fast and avoid excessive memory allocation where possible.
// For immutable value types, Clone MAY simply return the receiver if sharing
// is safe. For types with large nested structures, Clone SHOULD recursively
// clone nested objects.
//
// Clone MUST NOT mutate the receiver, MUST NOT have side effects, and MUST
// be safe to call concurrently.
//
// Example:
//
//	func (u User) Clone() User {
//	    return User{
//	        Name:  u.Name,
//	        Email: u.Email,
//	    }
//	}
type Cloneable[T any] interface {
	// Clone creates a deep copy of this instance. The returned instance has
	// the same value but shares no references with the original.
	//
	// This method MUST NOT mutate the receiver, MUST NOT have side effects,
	// and MUST be safe to call concurrently.
	Clone() T
}
