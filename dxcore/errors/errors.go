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

// Package errors provides reusable error types for dxapi enum-like types.
//
// This package defines common error types used across multiple dxapi packages
// (such as bump, kind, strategy, range) when parsing, marshaling and
// unmarshaling strongly typed enum-like values. By centralizing these types,
// the package eliminates code duplication and provides a consistent error
// handling story across the entire dxapi surface.
//
// The errors in this package are intentionally simple value carriers with
// stable message formats. They are designed to be:
//
//   - easy to construct from parsing / marshaling / unmarshaling code,
//   - easy to recognize via type assertions,
//   - and easy for users to understand when surfaced in logs or diagnostics.
//
// # Error Types
//
//   - ParseError
//     Returned when parsing a string into an enum-like type fails.
//     Use this when implementing ParseXxx helpers that accept textual input
//     (for example, from configuration files, environment variables or CLI).
//
//   - MarshalError
//     Returned when marshaling an invalid enum-like value fails.
//     Use this in MarshalJSON / MarshalText implementations to reject values
//     that do not correspond to known constants.
//
//   - UnmarshalError
//     Returned when unmarshaling data into an enum-like type fails due to
//     invalid input, parse errors or constraint violations.
//     Use this in UnmarshalJSON / UnmarshalText implementations to provide
//     precise diagnostics to callers.
//
//   - ValidationError
//     Returned when validation of a model type fails.
//     Use this in Validate() methods to report constraint violations,
//     missing required fields, or invalid field values.
//
// # Usage
//
// Each package that defines enum-like types can use these error types
// directly or create type aliases for better API clarity:
//
//	import "dirpx.dev/dxrel/dxapi/errors"
//
//	// Direct usage:
//	func ParseBump(s string) (Bump, error) {
//	    switch s {
//	    case "patch":
//	        return Patch, nil
//	    case "minor":
//	        return Minor, nil
//	    default:
//	        return 0, &errors.ParseError{Type: "Bump", Value: s}
//	    }
//	}
//
//	// Or with a type alias for API consistency in the local package:
//	type ParseError = errors.ParseError
package errors

import "strconv"

// ParseError is returned when parsing a string into a strongly typed enum-like
// value fails.
//
// Type identifies the logical type being parsed (for example, "Bump", "Kind",
// "Strategy"), and Value contains the exact string that could not be
// interpreted. This struct is intended for use in error messages and
// diagnostics; callers MAY pattern-match on Type to provide type-specific
// guidance to users or to translate errors into friendlier messages.
//
// # Example
//
//	func ParseBump(s string) (Bump, error) {
//	    switch s {
//	    case "patch":
//	        return Patch, nil
//	    case "minor":
//	        return Minor, nil
//	    default:
//	        // Returned error will format as:
//	        // "dxapi: invalid Bump value: <value>"
//	        return 0, &errors.ParseError{
//	            Type:  "Bump",
//	            Value: s,
//	        }
//	    }
//	}
type ParseError struct {
	// Type is the logical name of the type being parsed (for example, "Bump").
	Type string

	// Value is the invalid textual representation that was provided.
	Value string
}

// Error implements the error interface for ParseError.
//
// The error message format is:
//
//	"dxapi: invalid {Type} value: {Value}"
//
// For example:
//
//	"dxapi: invalid Bump value: unknown"
//
// The format is intentionally stable so that callers can rely on it for
// diagnostics, while still preferring type assertions where possible.
func (e *ParseError) Error() string {
	return "dxapi: invalid " + e.Type + " value: " + e.Value
}

// MarshalError is returned when marshaling a typed value fails due to it being
// outside the set of valid constants.
//
// Type identifies the logical type being marshaled (for example, "Bump"), and
// Value contains the underlying numeric value that was deemed invalid.
//
// This error is primarily used as a guardrail: it prevents invalid enum-like
// values from being silently emitted into JSON, YAML or other serialized
// forms. In most cases a MarshalError indicates a programming error (for
// example, a zero value that was never validated).
//
// # Example
//
//	func (b Bump) MarshalJSON() ([]byte, error) {
//	    if !b.Valid() {
//	        // Returned error will format as:
//	        // "dxapi: cannot marshal invalid Bump value: <int>"
//	        return nil, &errors.MarshalError{
//	            Type:  "Bump",
//	            Value: int(b),
//	        }
//	    }
//	    return []byte(`"` + b.String() + `"`), nil
//	}
type MarshalError struct {
	// Type is the logical name of the type being marshaled (for example, "Bump").
	Type string

	// Value is the underlying numeric representation that could not be
	// marshaled because it does not correspond to a known constant.
	Value int
}

// Error implements the error interface for MarshalError.
//
// The error message format is:
//
//	"dxapi: cannot marshal invalid {Type} value: {Value}"
//
// where Value is rendered as a decimal integer.
//
// For example:
//
//	"dxapi: cannot marshal invalid Bump value: 99"
//
// This ensures that invalid numeric values are clearly displayed in error
// messages, making it easy to identify and debug marshaling failures.
func (e *MarshalError) Error() string {
	return "dxapi: cannot marshal invalid " + e.Type + " value: " + strconv.Itoa(e.Value)
}

// UnmarshalError is returned when unmarshaling data into a typed value fails.
//
// Type identifies the logical type being populated (for example, "Bump"),
// Data contains the original raw payload (typically a JSON fragment), and
// Reason provides a human-readable description of what went wrong (for
// example, parse errors, invalid numeric value or empty input).
//
// This struct is intended to be surfaced directly in diagnostics or logs so
// that users can understand why their configuration or payload could not be
// interpreted. Callers MAY wrap UnmarshalError with additional context when
// propagating it further up the stack.
//
// # Example
//
//	func (b *Bump) UnmarshalJSON(data []byte) error {
//	    if len(data) == 0 {
//	        return &errors.UnmarshalError{
//	            Type:   "Bump",
//	            Data:   data,
//	            Reason: "empty data",
//	        }
//	    }
//
//	    // ... parsing logic ...
//
//	    // On invalid value:
//	    // return &errors.UnmarshalError{
//	    //     Type:   "Bump",
//	    //     Data:   data,
//	    //     Reason: "unknown value",
//	    // }
//	}
type UnmarshalError struct {
	// Type is the logical name of the type being unmarshaled into.
	Type string

	// Data is the raw input that failed to unmarshal.
	//
	// Callers MAY choose to log or redact this field depending on privacy
	// and size considerations.
	Data []byte

	// Reason is a short, human-readable explanation of the failure.
	//
	// Reason SHOULD describe what went wrong (for example, "empty data" or
	// "unknown value 'foo'") rather than repeating the type name; the type
	// name is already available in the Type field and reflected in Error().
	Reason string
}

// Error implements the error interface for UnmarshalError.
//
// The error message format is:
//
//	"dxapi: cannot unmarshal {Type}: {Reason}"
//
// For example:
//
//	"dxapi: cannot unmarshal Bump: empty data"
//
// The Data field is intentionally not included in the formatted message to
// avoid excessively verbose or sensitive logs; callers can log it separately
// when appropriate.
func (e *UnmarshalError) Error() string {
	return "dxapi: cannot unmarshal " + e.Type + ": " + e.Reason
}

// ValidationError is returned when validation of a model type fails.
//
// Type identifies the logical name of the type being validated (for example,
// "Commit", "Ref"), Field optionally identifies which field failed validation,
// Reason provides a human-readable explanation of the validation failure, and
// Value optionally contains the problematic value that failed validation.
//
// This error is used by Validate() methods in model types to report
// constraint violations, missing required fields, or invalid field values.
//
// # Example
//
//	func (c Commit) Validate() error {
//	    if c.Hash.IsZero() {
//	        return &errors.ValidationError{
//	            Type:   "Commit",
//	            Field:  "Hash",
//	            Reason: "must not be empty",
//	        }
//	    }
//	    return nil
//	}
type ValidationError struct {
	// Type is the logical name of the type being validated.
	Type string

	// Field is the name of the field that failed validation.
	// May be empty if the error applies to the entire type.
	Field string

	// Reason is a short, human-readable explanation of why validation failed.
	Reason string

	// Value optionally contains the invalid value.
	// May be nil if not applicable or if the value should not be logged.
	Value any
}

// Error implements the error interface for ValidationError.
//
// The error message format is:
//
//	"dxapi: invalid {Type}.{Field}: {Reason}" (when Field is specified)
//	"dxapi: invalid {Type}: {Reason}" (when Field is empty)
//
// For example:
//
//	"dxapi: invalid Commit.Hash: must not be empty"
//	"dxapi: invalid Strategy: invalid value"
func (e *ValidationError) Error() string {
	if e.Field != "" {
		return "dxapi: invalid " + e.Type + "." + e.Field + ": " + e.Reason
	}
	return "dxapi: invalid " + e.Type + ": " + e.Reason
}
