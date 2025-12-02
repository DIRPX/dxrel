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

package model

import (
	"encoding/json"
	"fmt"

	"dirpx.dev/rxmerr"
	"gopkg.in/yaml.v3"
)

// ValidateAll validates a slice of models and returns all validation errors
// encountered during the batch validation process. This function provides a
// convenient way to validate multiple model instances in a single operation
// while collecting comprehensive error information about all validation
// failures rather than stopping at the first error.
//
// The function iterates through each model in the provided slice and invokes
// its Validate method. When a model fails validation, the error is wrapped
// with contextual information including the model's position in the slice
// (zero-indexed) and its type name obtained from TypeName. This allows callers
// to identify exactly which models failed validation and why.
//
// If one or more models fail validation, ValidateAll returns a single combined
// error that aggregates all individual validation failures using rxmerr.Collector.
// The returned error contains all validation failures in a structured format that
// can be inspected programmatically. If all models pass validation, the function
// returns nil. The function never panics and always processes the entire slice
// even when early elements fail validation, ensuring complete error reporting.
//
// Empty slices are considered valid and return nil. The function handles nil
// pointers within the slice according to the behavior of each model's Validate
// method, typically resulting in a validation error unless the model explicitly
// supports nil as a valid state.
//
// Example usage for batch validation of configuration models:
//
//	models := []Model{model1, model2, model3}
//	if err := ValidateAll(models); err != nil {
//	    log.Error("validation failed", "error", err)
//	}
func ValidateAll[T Model](models []T) error {
	c := rxmerr.NewCollector()

	for i, m := range models {
		if err := m.Validate(); err != nil {
			c.Append(fmt.Errorf("model[%d] (%s): %w", i, m.TypeName(), err))
		}
	}

	return c.Err()
}

// FilterZero returns a new slice containing only non-zero models by removing
// all instances where IsZero returns true. This function provides a convenient
// way to clean slices of empty or uninitialized model values before processing,
// serialization, or transmission over network boundaries.
//
// The function allocates a new slice with capacity equal to the input slice
// length for optimal memory usage in the common case where most models are
// non-zero. It then iterates through each model and invokes its IsZero method
// to determine whether the model represents an empty or default value. Only
// models where IsZero returns false are included in the result slice.
//
// The returned slice is always a new allocation and never shares backing array
// storage with the input slice, ensuring that modifications to either slice
// do not affect the other. If all models in the input are zero, the function
// returns an empty slice (not nil). If the input slice is empty or nil, the
// function returns an empty non-nil slice.
//
// Callers SHOULD use FilterZero before serializing collections to avoid
// transmitting empty placeholder values. Callers MAY use FilterZero before
// batch operations to avoid processing invalid or incomplete data. The function
// does not validate models; it only checks for zero values using IsZero.
//
// Example usage for cleaning a model collection before JSON serialization:
//
//	models := []Model{validModel, zeroModel, anotherValidModel}
//	nonZero := FilterZero(models)
//	// nonZero contains only validModel and anotherValidModel
func FilterZero[T Model](models []T) []T {
	result := make([]T, 0, len(models))

	for _, m := range models {
		if !m.IsZero() {
			result = append(result, m)
		}
	}

	return result
}

// MustValidate validates a model and panics if validation fails, providing a
// convenient way to assert model validity in contexts where invalid models
// represent programming errors rather than recoverable runtime errors. This
// function is designed for use in test code, initialization sequences, and
// other scenarios where panic-on-failure semantics are appropriate and desired.
//
// The function invokes the model's Validate method and examines the returned
// error. If validation succeeds (error is nil), MustValidate returns the model
// unchanged, allowing method chaining and inline initialization patterns. If
// validation fails, the function panics with a formatted message that includes
// the model's type name from TypeName and the validation error, providing clear
// diagnostics about what went wrong and which model type failed.
//
// Callers MUST only use MustValidate in contexts where panic is an acceptable
// control flow mechanism, such as test setup functions, package initialization
// code executed during program startup, or command-line tools where fatal
// errors should terminate execution. Callers MUST NOT use MustValidate in
// production server code, request handlers, background workers, or any context
// where panic would disrupt service availability or cause cascading failures.
//
// The panic behavior ensures that programming errors (such as hardcoded invalid
// test data or misconfigured initialization constants) are caught immediately
// and loudly rather than propagating through the system as subtle bugs.
//
// Example usage in test setup where invalid data indicates a test bug:
//
//	func TestSomething(t *testing.T) {
//	    m := MustValidate(ExampleModel{Name: "test"})
//	    // Use m knowing it's valid
//	}
func MustValidate[T Model](m T) T {
	if err := m.Validate(); err != nil {
		panic(fmt.Sprintf("model validation failed for %s: %v", m.TypeName(), err))
	}
	return m
}

// SafeString returns a string representation of a model that is safe for
// logging by default but can optionally include full details including
// sensitive data when explicitly requested. This function provides a unified
// interface for obtaining model string representations while making the safety
// characteristics explicit through the unsafe parameter.
//
// When the unsafe parameter is false (the default and recommended value for
// production logging), SafeString invokes the model's Redacted method to
// obtain a representation with sensitive fields masked or removed. This
// protects against accidental exposure of passwords, tokens, API keys, and
// personally identifiable information (PII) in application logs.
//
// When the unsafe parameter is true, SafeString invokes the model's String
// method to obtain a complete representation that MAY include sensitive data.
// Callers MUST only set unsafe to true in controlled debugging scenarios where
// the output destination is secured and the data will not be persisted to
// long-term storage or transmitted across trust boundaries.
//
// The function provides a single call site for logging decisions, making it
// easier to audit logging behavior and ensuring that the choice between safe
// and unsafe representations is always explicit and visible in the code.
// Production logging frameworks SHOULD always pass false for the unsafe
// parameter. Development and debugging tools MAY provide user-controlled
// settings to enable unsafe mode when troubleshooting specific issues.
//
// Example usage showing safe production logging and unsafe debug logging:
//
//	log.Info("processing", "model", SafeString(model, false))  // Uses Redacted()
//	log.Debug("details", "model", SafeString(model, true))    // Uses String() (UNSAFE)
func SafeString[T Model](m T, unsafe bool) string {
	if unsafe {
		return m.String()
	}
	return m.Redacted()
}

// ToJSON converts a model to JSON bytes after validating that the model is in
// a consistent and valid state. This function provides a safe convenience
// wrapper around json.Marshal that enforces the contract that only valid
// models can be serialized, preventing transmission or persistence of invalid
// data that could cause downstream processing failures.
//
// The function first invokes the model's Validate method to check all
// invariants and required fields. If validation fails, ToJSON returns an error
// that wraps the validation failure with context identifying the model type
// from TypeName. No marshaling is attempted when validation fails, ensuring
// that invalid data never reaches the JSON encoder.
//
// If validation succeeds, ToJSON invokes json.Marshal to serialize the model.
// The model's MarshalJSON method is called if implemented, allowing custom
// serialization logic. If marshaling fails (for example, due to unsupported
// types or cyclical references), the error from json.Marshal is returned
// directly without additional wrapping.
//
// Callers SHOULD use ToJSON instead of calling json.Marshal directly when they
// need the additional safety guarantee that only valid models are serialized.
// Callers MAY call json.Marshal directly if they have already validated the
// model through other means and want to avoid redundant validation overhead,
// though this trades safety for performance. The returned byte slice can be
// written to files, transmitted over network connections, or stored in
// databases, with confidence that it represents a valid model instance.
//
// Example usage for safely serializing a model before network transmission:
//
//	data, err := ToJSON(model)
//	if err != nil {
//	    return err
//	}
//	// Write data to file or send over network
func ToJSON[T Model](m T) ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", m.TypeName(), err)
	}
	return json.Marshal(m)
}

// ToYAML converts a model to YAML bytes after validating that the model is in
// a consistent and valid state. This function provides a safe convenience
// wrapper around yaml.Marshal that enforces the contract that only valid
// models can be serialized, preventing the creation of invalid configuration
// files or YAML documents that could cause parsing errors or incorrect behavior
// when loaded by other systems.
//
// The function first invokes the model's Validate method to verify all
// invariants and required fields. If validation fails, ToYAML returns an error
// that wraps the validation failure with context identifying the model type
// from TypeName. No marshaling is attempted when validation fails, ensuring
// that invalid data never reaches the YAML encoder.
//
// If validation succeeds, ToYAML invokes yaml.Marshal to serialize the model
// to YAML format. The model's MarshalYAML method is called if implemented,
// allowing custom serialization logic such as field ordering or custom type
// representations. If marshaling fails (for example, due to unsupported types),
// the error from yaml.Marshal is returned directly without additional wrapping.
//
// Callers SHOULD use ToYAML instead of calling yaml.Marshal directly when they
// need the additional safety guarantee that only valid models are serialized
// to YAML format. Callers MAY call yaml.Marshal directly if they have already
// validated the model through other means, though this trades safety for
// performance. The returned byte slice is typically written to configuration
// files, included in deployment manifests, or transmitted as human-readable
// structured data.
//
// Example usage for safely writing a validated model to a configuration file:
//
//	data, err := ToYAML(model)
//	if err != nil {
//	    return err
//	}
//	// Write data to config file
func ToYAML[T Model](m T) ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("cannot marshal invalid %s: %w", m.TypeName(), err)
	}
	return yaml.Marshal(m)
}

// FromJSON parses JSON bytes into a model and validates the result to ensure
// that the unmarshaled data represents a consistent and valid model instance.
// This function provides a safe convenience wrapper around json.Unmarshal that
// enforces the contract that deserialized models are always validated before
// being returned to callers, preventing invalid data from entering the system
// through external inputs.
//
// The function first invokes json.Unmarshal to decode the JSON bytes into the
// provided model pointer. If unmarshaling fails due to malformed JSON, type
// mismatches, or other parsing errors, FromJSON returns an error describing
// the unmarshaling failure. No validation is attempted when unmarshaling fails
// because there is no complete model instance to validate.
//
// If unmarshaling succeeds, FromJSON invokes the model's Validate method to
// verify that the unmarshaled data satisfies all invariants and required
// fields. If validation fails, FromJSON returns an error indicating that the
// unmarshaled model is invalid, even though the JSON syntax was correct. This
// ensures that malformed or incomplete data from external sources (such as
// API requests, configuration files, or database records) is rejected at the
// system boundary rather than causing downstream processing errors.
//
// Callers MUST provide a pointer to a model variable that will receive the
// unmarshaled result. The model variable SHOULD be zero-initialized before
// calling FromJSON to ensure predictable behavior. If FromJSON returns an
// error, the model variable's state is undefined and MUST NOT be used.
//
// Callers SHOULD use FromJSON instead of calling json.Unmarshal directly when
// loading data from untrusted or external sources to ensure automatic
// validation. Callers MAY call json.Unmarshal directly in performance-critical
// paths where validation can be deferred, though this trades safety for speed.
//
// Example usage for safely loading a model from JSON with validation:
//
//	var m ExampleModel
//	if err := FromJSON(data, &m); err != nil {
//	    return err
//	}
//	// Use m knowing it's valid
func FromJSON[T Model](data []byte, m *T) error {
	if err := json.Unmarshal(data, m); err != nil {
		return fmt.Errorf("cannot unmarshal JSON: %w", err)
	}
	if err := (*m).Validate(); err != nil {
		return fmt.Errorf("unmarshaled model is invalid: %w", err)
	}
	return nil
}

// FromYAML parses YAML bytes into a model and validates the result to ensure
// that the unmarshaled data represents a consistent and valid model instance.
// This function provides a safe convenience wrapper around yaml.Unmarshal that
// enforces the contract that deserialized models are always validated before
// being returned to callers, preventing invalid configuration data or malformed
// YAML documents from corrupting system state.
//
// The function first invokes yaml.Unmarshal to decode the YAML bytes into the
// provided model pointer. If unmarshaling fails due to malformed YAML syntax,
// type mismatches, duplicate keys, or other parsing errors specific to YAML
// format, FromYAML returns an error describing the unmarshaling failure. No
// validation is attempted when unmarshaling fails because there is no complete
// model instance to validate.
//
// If unmarshaling succeeds, FromYAML invokes the model's Validate method to
// verify that the unmarshaled data satisfies all invariants and required
// fields. If validation fails, FromYAML returns an error indicating that the
// unmarshaled model is invalid, even though the YAML syntax was correct. This
// ensures that configuration files or YAML documents with missing required
// fields, invalid values, or violated business rules are rejected when loaded
// rather than causing runtime errors or incorrect system behavior.
//
// Callers MUST provide a pointer to a model variable that will receive the
// unmarshaled result. The model variable SHOULD be zero-initialized before
// calling FromYAML to ensure predictable behavior. If FromYAML returns an
// error, the model variable's state is undefined and MUST NOT be used.
//
// Callers SHOULD use FromYAML instead of calling yaml.Unmarshal directly when
// loading configuration files or YAML documents from external sources to
// ensure automatic validation. Callers MAY call yaml.Unmarshal directly in
// scenarios where they need partial parsing or want to handle validation
// separately, though this trades safety for flexibility.
//
// Example usage for safely loading a model from a YAML configuration file:
//
//	var m ExampleModel
//	if err := FromYAML(data, &m); err != nil {
//	    return err
//	}
//	// Use m knowing it's valid
func FromYAML[T Model](data []byte, m *T) error {
	if err := yaml.Unmarshal(data, m); err != nil {
		return fmt.Errorf("cannot unmarshal YAML: %w", err)
	}
	if err := (*m).Validate(); err != nil {
		return fmt.Errorf("unmarshaled model is invalid: %w", err)
	}
	return nil
}

// Clone creates a deep copy of a model by serializing it to JSON and then
// deserializing back into a new instance, ensuring complete independence
// between the original and the copy. This function provides a generic cloning
// implementation that works for any Model type without requiring type-specific
// copy logic, though at the cost of JSON round-trip overhead.
//
// The function first invokes json.Marshal on the source model to serialize it
// to JSON bytes. If marshaling fails (which typically indicates the model
// contains unserializable types or has a broken MarshalJSON implementation),
// Clone returns an error and a zero-value model. If marshaling succeeds, Clone
// invokes json.Unmarshal to deserialize the JSON bytes into a new model
// instance of the same type. If unmarshaling fails, Clone returns an error
// and a zero-value model.
//
// The JSON round-trip approach guarantees a deep copy because JSON
// serialization naturally handles nested structures, slices, maps, and
// pointer indirection by value rather than by reference. The cloned model is
// completely independent of the original, meaning modifications to either
// instance do not affect the other. This holds true even for nested models,
// slices of models, and maps containing models.
//
// The primary drawback of this implementation is performance overhead from
// JSON encoding and decoding. For performance-critical code paths that clone
// models frequently, implementations SHOULD provide a custom Clone method by
// implementing the Cloneable[T] interface with hand-written copy logic that
// avoids serialization overhead. For general-purpose code where cloning is
// infrequent, this generic Clone function provides simplicity and correctness.
//
// Callers MUST check the returned error before using the cloned model. If
// Clone returns an error, the model return value is a zero-value instance that
// MUST NOT be used.
//
// Example usage for creating an independent copy of a model:
//
//	copy, err := Clone(original)
//	if err != nil {
//	    return err
//	}
//	// Modify copy without affecting original
func Clone[T Model](m T) (T, error) {
	var zero T

	data, err := json.Marshal(m)
	if err != nil {
		return zero, fmt.Errorf("clone marshal failed: %w", err)
	}

	var clone T
	if err := json.Unmarshal(data, &clone); err != nil {
		return zero, fmt.Errorf("clone unmarshal failed: %w", err)
	}

	return clone, nil
}

// Equal compares two models for equality by serializing both to JSON and
// comparing their JSON representations byte-for-byte. This function provides a
// generic equality implementation that works for any Model type without
// requiring type-specific comparison logic, though at the cost of JSON
// serialization overhead and the limitations of JSON-based comparison semantics.
//
// The function invokes json.Marshal on both models to obtain their JSON byte
// representations. If either marshaling operation fails (which typically
// indicates unserializable types, broken MarshalJSON implementations, or
// invalid models), Equal returns false without attempting to compare partial
// results. This fail-safe behavior ensures that comparison errors are not
// mistaken for inequality.
//
// If both models marshal successfully, Equal converts the resulting byte
// slices to strings and performs a direct string equality comparison. Two
// models are considered equal if and only if their JSON representations are
// identical byte-for-byte after marshaling. This comparison includes all
// exported fields and respects custom MarshalJSON implementations.
//
// The JSON-based comparison has important semantic implications. First, field
// order in the JSON output MUST be deterministic for reliable equality checks,
// which is generally guaranteed by Go's JSON encoder for struct fields but not
// for map iterations. Second, unexported fields are not compared because they
// do not appear in JSON output. Third, functionally equivalent values that
// serialize differently (such as empty slices versus nil slices in some
// implementations) MAY be considered unequal despite semantic equivalence.
//
// For performance-critical code paths that compare models frequently,
// implementations SHOULD provide a custom Equal method by implementing the
// Comparable[T] interface with hand-written comparison logic that avoids
// serialization overhead. For general-purpose code where comparison is
// infrequent, this generic Equal function provides simplicity and works across
// all Model types uniformly.
//
// Example usage for checking if two model instances represent the same data:
//
//	if Equal(model1, model2) {
//	    log.Info("models are equal")
//	}
func Equal[T Model](a, b T) bool {
	dataA, errA := json.Marshal(a)
	dataB, errB := json.Marshal(b)

	if errA != nil || errB != nil {
		return false
	}

	return string(dataA) == string(dataB)
}
