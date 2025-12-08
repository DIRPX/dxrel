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

package errors

import "testing"

func TestParseError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *ParseError
		want string
	}{
		{
			"Bump type",
			&ParseError{Type: "Bump", Value: "unknown"},
			"dxapi: invalid Bump value: unknown",
		},
		{
			"Kind type",
			&ParseError{Type: "Kind", Value: "invalid"},
			"dxapi: invalid Kind value: invalid",
		},
		{
			"Strategy type",
			&ParseError{Type: "Strategy", Value: "bad"},
			"dxapi: invalid Strategy value: bad",
		},
		{
			"empty value",
			&ParseError{Type: "Mode", Value: ""},
			"dxapi: invalid Mode value: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("ParseError.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMarshalError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *MarshalError
		want string
	}{
		{
			"positive value",
			&MarshalError{Type: "Bump", Value: 99},
			"dxapi: cannot marshal invalid Bump value: 99",
		},
		{
			"negative value",
			&MarshalError{Type: "Kind", Value: -1},
			"dxapi: cannot marshal invalid Kind value: -1",
		},
		{
			"zero value",
			&MarshalError{Type: "Strategy", Value: 0},
			"dxapi: cannot marshal invalid Strategy value: 0",
		},
		{
			"large value",
			&MarshalError{Type: "Mode", Value: 12345},
			"dxapi: cannot marshal invalid Mode value: 12345",
		},
		{
			"value 42 should be decimal not unicode",
			&MarshalError{Type: "Test", Value: 42},
			"dxapi: cannot marshal invalid Test value: 42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("MarshalError.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUnmarshalError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *UnmarshalError
		want string
	}{
		{
			"empty data",
			&UnmarshalError{
				Type:   "Bump",
				Data:   []byte{},
				Reason: "empty data",
			},
			"dxapi: cannot unmarshal Bump: empty data",
		},
		{
			"invalid format",
			&UnmarshalError{
				Type:   "Kind",
				Data:   []byte(`"bad"`),
				Reason: "invalid format",
			},
			"dxapi: cannot unmarshal Kind: invalid format",
		},
		{
			"parse error",
			&UnmarshalError{
				Type:   "Strategy",
				Data:   []byte(`99`),
				Reason: "invalid numeric value",
			},
			"dxapi: cannot unmarshal Strategy: invalid numeric value",
		},
		{
			"json syntax error",
			&UnmarshalError{
				Type:   "Mode",
				Data:   []byte(`{broken`),
				Reason: "unexpected end of JSON input",
			},
			"dxapi: cannot unmarshal Mode: unexpected end of JSON input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("UnmarshalError.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestErrors_Implements_Error_Interface(t *testing.T) {
	// Verify that all error types implement error interface
	var _ error = (*ParseError)(nil)
	var _ error = (*MarshalError)(nil)
	var _ error = (*UnmarshalError)(nil)
}
