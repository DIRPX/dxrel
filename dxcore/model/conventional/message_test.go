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

package conventional_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"dirpx.dev/dxrel/dxcore/model/conventional"
	"gopkg.in/yaml.v3"
)

func TestParseMessage(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    conventional.Message
		wantErr bool
	}{
		{
			name:  "simple_message",
			input: "feat: add user authentication",
			want: conventional.Message{
				Type:    conventional.Feat,
				Subject: "add user authentication",
			},
			wantErr: false,
		},
		{
			name:  "with_scope",
			input: "fix(api): resolve timeout issue",
			want: conventional.Message{
				Type:    conventional.Fix,
				Scope:   "api",
				Subject: "resolve timeout issue",
			},
			wantErr: false,
		},
		{
			name:  "breaking_change",
			input: "feat!: remove deprecated endpoint",
			want: conventional.Message{
				Type:     conventional.Feat,
				Subject:  "remove deprecated endpoint",
				Breaking: true,
			},
			wantErr: false,
		},
		{
			name:  "breaking_with_scope",
			input: "fix(auth)!: change authentication flow",
			want: conventional.Message{
				Type:     conventional.Fix,
				Scope:    "auth",
				Subject:  "change authentication flow",
				Breaking: true,
			},
			wantErr: false,
		},
		{
			name:  "with_body",
			input: "feat: add caching\n\nImproves performance significantly",
			want: conventional.Message{
				Type:    conventional.Feat,
				Subject: "add caching",
				Body:    "Improves performance significantly",
			},
			wantErr: false,
		},
		{
			name:  "with_multiline_body",
			input: "docs: update README\n\nAdd installation instructions.\nAdd usage examples.\nFix typos.",
			want: conventional.Message{
				Type:    conventional.Docs,
				Subject: "update README",
				Body:    "Add installation instructions.\nAdd usage examples.\nFix typos.",
			},
			wantErr: false,
		},
		{
			name:  "with_trailers",
			input: "fix: resolve bug\n\nFixes: #123\nReviewed-by: Alice",
			want: conventional.Message{
				Type:    conventional.Fix,
				Subject: "resolve bug",
				Trailers: []conventional.Trailer{
					{Key: "Fixes", Value: "#123"},
					{Key: "Reviewed-by", Value: "Alice"},
				},
			},
			wantErr: false,
		},
		{
			name:  "complete_message",
			input: "feat(api)!: add user endpoint\n\nAdds a new REST API endpoint for user management.\n\nThis is a breaking change because it modifies the API contract.\n\nFixes: #456\nSigned-off-by: Bob <bob@example.com>",
			want: conventional.Message{
				Type:     conventional.Feat,
				Scope:    "api",
				Subject:  "add user endpoint",
				Breaking: true,
				Body:     "Adds a new REST API endpoint for user management.\n\nThis is a breaking change because it modifies the API contract.",
				Trailers: []conventional.Trailer{
					{Key: "Fixes", Value: "#456"},
					{Key: "Signed-off-by", Value: "Bob <bob@example.com>"},
				},
			},
			wantErr: false,
		},
		{
			name:  "breaking_from_BREAKING_CHANGE_trailer_with_space",
			input: "feat: add new API\n\nBREAKING CHANGE: removes old endpoint",
			want: conventional.Message{
				Type:     conventional.Feat,
				Subject:  "add new API",
				Breaking: true,
				Body:     "",
				Trailers: nil, // BREAKING CHANGE with space is not added to Trailers
			},
			wantErr: false,
		},
		{
			name:  "breaking_from_BREAKING_CHANGE_trailer_with_hyphen",
			input: "feat: add new API\n\nBREAKING-CHANGE: removes old endpoint",
			want: conventional.Message{
				Type:     conventional.Feat,
				Subject:  "add new API",
				Breaking: true,
				Body:     "",
				Trailers: []conventional.Trailer{
					{Key: "BREAKING-CHANGE", Value: "removes old endpoint"},
				},
			},
			wantErr: false,
		},
		{
			name:  "breaking_from_both_marker_and_trailer",
			input: "feat!: add new API\n\nBREAKING CHANGE: removes old endpoint",
			want: conventional.Message{
				Type:     conventional.Feat,
				Subject:  "add new API",
				Breaking: true,
				Body:     "",
				Trailers: nil,
			},
			wantErr: false,
		},
		{
			name:    "empty_message",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid_header_no_colon",
			input:   "feat add feature",
			wantErr: true,
		},
		{
			name:    "invalid_type",
			input:   "invalid: add feature",
			wantErr: true,
		},
		{
			name:    "uppercase_type",
			input:   "FEAT: add feature",
			wantErr: true,
		},
		{
			name:    "no_subject",
			input:   "feat:",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := conventional.ParseMessage(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if !got.Type.Equal(tt.want.Type) {
					t.Errorf("Type = %v, want %v", got.Type, tt.want.Type)
				}
				if !got.Scope.Equal(tt.want.Scope) {
					t.Errorf("Scope = %q, want %q", got.Scope, tt.want.Scope)
				}
				if !got.Subject.Equal(tt.want.Subject) {
					t.Errorf("Subject = %q, want %q", got.Subject, tt.want.Subject)
				}
				if got.Breaking != tt.want.Breaking {
					t.Errorf("Breaking = %v, want %v", got.Breaking, tt.want.Breaking)
				}
				if !got.Body.Equal(tt.want.Body) {
					t.Errorf("Body = %q, want %q", got.Body, tt.want.Body)
				}
				if len(got.Trailers) != len(tt.want.Trailers) {
					t.Errorf("len(Trailers) = %d, want %d", len(got.Trailers), len(tt.want.Trailers))
				} else {
					for i := range got.Trailers {
						if !got.Trailers[i].Equal(tt.want.Trailers[i]) {
							t.Errorf("Trailers[%d] = %v, want %v", i, got.Trailers[i], tt.want.Trailers[i])
						}
					}
				}
			}
		})
	}
}

func TestMessage_String(t *testing.T) {
	tests := []struct {
		name string
		msg  conventional.Message
		want string
	}{
		{
			name: "simple",
			msg: conventional.Message{
				Type:    conventional.Feat,
				Subject: "add feature",
			},
			want: "feat: add feature",
		},
		{
			name: "with_scope",
			msg: conventional.Message{
				Type:    conventional.Fix,
				Scope:   "api",
				Subject: "fix bug",
			},
			want: "fix(api): fix bug",
		},
		{
			name: "breaking",
			msg: conventional.Message{
				Type:     conventional.Feat,
				Subject:  "breaking change",
				Breaking: true,
			},
			want: "feat!: breaking change",
		},
		{
			name: "with_body",
			msg: conventional.Message{
				Type:    conventional.Docs,
				Subject: "update docs",
				Body:    "Added examples",
			},
			want: "docs: update docs\n\nAdded examples",
		},
		{
			name: "with_trailers",
			msg: conventional.Message{
				Type:    conventional.Fix,
				Subject: "fix issue",
				Trailers: []conventional.Trailer{
					{Key: "Fixes", Value: "#123"},
				},
			},
			want: "fix: fix issue\n\nFixes: #123",
		},
		{
			name: "complete",
			msg: conventional.Message{
				Type:     conventional.Feat,
				Scope:    "core",
				Subject:  "add feature",
				Breaking: true,
				Body:     "Feature description",
				Trailers: []conventional.Trailer{
					{Key: "Signed-off-by", Value: "Alice"},
				},
			},
			want: "feat(core)!: add feature\n\nFeature description\n\nSigned-off-by: Alice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.msg.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMessage_Validate(t *testing.T) {
	tests := []struct {
		name    string
		msg     conventional.Message
		wantErr bool
	}{
		{
			name: "valid_simple",
			msg: conventional.Message{
				Type:    conventional.Feat,
				Subject: "add feature",
			},
			wantErr: false,
		},
		{
			name: "valid_complete",
			msg: conventional.Message{
				Type:     conventional.Fix,
				Scope:    "api",
				Subject:  "fix bug",
				Breaking: true,
				Body:     "Bug description",
				Trailers: []conventional.Trailer{
					{Key: "Fixes", Value: "#123"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing_subject",
			msg: conventional.Message{
				Type: conventional.Feat,
			},
			wantErr: true,
		},
		{
			name: "invalid_scope",
			msg: conventional.Message{
				Type:    conventional.Feat,
				Scope:   "INVALID",
				Subject: "add feature",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMessage_JSON_RoundTrip(t *testing.T) {
	msg := conventional.Message{
		Type:     conventional.Feat,
		Scope:    "api",
		Subject:  "add endpoint",
		Breaking: true,
		Body:     "Adds new endpoint",
		Trailers: []conventional.Trailer{
			{Key: "Fixes", Value: "#123"},
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal() failed: %v", err)
	}

	var decoded conventional.Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() failed: %v", err)
	}

	if !decoded.Equal(msg) {
		t.Errorf("Round-trip failed: got %+v, want %+v", decoded, msg)
	}
}

func TestMessage_YAML_RoundTrip(t *testing.T) {
	msg := conventional.Message{
		Type:     conventional.Fix,
		Scope:    "core",
		Subject:  "fix bug",
		Breaking: false,
		Body:     "Bug fix description",
		Trailers: []conventional.Trailer{
			{Key: "Reviewed-by", Value: "Bob"},
		},
	}

	data, err := yaml.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal() failed: %v", err)
	}

	var decoded conventional.Message
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() failed: %v", err)
	}

	if !decoded.Equal(msg) {
		t.Errorf("Round-trip failed: got %+v, want %+v", decoded, msg)
	}
}

func TestMessage_ParseAndFormat_RoundTrip(t *testing.T) {
	tests := []string{
		"feat: add feature",
		"fix(api): resolve issue",
		"feat!: breaking change",
		"docs: update README\n\nAdded examples",
		"fix: bug\n\nFixes: #123\nReviewed-by: Alice",
	}

	for i, input := range tests {
		t.Run(fmt.Sprintf("test_%d", i), func(t *testing.T) {
			msg, err := conventional.ParseMessage(input)
			if err != nil {
				t.Fatalf("ParseMessage() failed: %v", err)
			}

			output := msg.String()

			// Parse again
			msg2, err := conventional.ParseMessage(output)
			if err != nil {
				t.Fatalf("ParseMessage() second parse failed: %v", err)
			}

			if !msg.Equal(msg2) {
				t.Errorf("Round-trip failed:\noriginal: %+v\nreparsed: %+v", msg, msg2)
			}
		})
	}
}

func TestMessage_Redacted(t *testing.T) {
	msg := conventional.Message{
		Type:    conventional.Feat,
		Scope:   "api",
		Subject: "add endpoint",
		Body:    "This is a long body with sensitive information",
		Trailers: []conventional.Trailer{
			{Key: "Signed-off-by", Value: "secret@example.com"},
		},
	}

	redacted := msg.Redacted()
	expected := "feat(api): add endpoint"

	if redacted != expected {
		t.Errorf("Redacted() = %q, want %q", redacted, expected)
	}

	// Ensure body and trailers are not in redacted output
	if strings.Contains(redacted, "long body") {
		t.Error("Redacted() contains body content")
	}
	if strings.Contains(redacted, "secret") {
		t.Error("Redacted() contains trailer content")
	}
}
