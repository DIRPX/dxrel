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

package git_test

import (
	"encoding/json"
	"strings"
	"testing"

	"dirpx.dev/dxrel/dxcore/model/git"
	"gopkg.in/yaml.v3"
)

func TestNewWorktreeStatus(t *testing.T) {
	tests := []struct {
		name         string
		hasUnstaged  bool
		hasStaged    bool
		hasUntracked bool
	}{
		{
			name:         "clean",
			hasUnstaged:  false,
			hasStaged:    false,
			hasUntracked: false,
		},
		{
			name:         "only_unstaged",
			hasUnstaged:  true,
			hasStaged:    false,
			hasUntracked: false,
		},
		{
			name:         "only_staged",
			hasUnstaged:  false,
			hasStaged:    true,
			hasUntracked: false,
		},
		{
			name:         "only_untracked",
			hasUnstaged:  false,
			hasStaged:    false,
			hasUntracked: true,
		},
		{
			name:         "all_dirty",
			hasUnstaged:  true,
			hasStaged:    true,
			hasUntracked: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := git.NewWorktreeStatus(tt.hasUnstaged, tt.hasStaged, tt.hasUntracked)
			if status.HasUnstaged != tt.hasUnstaged {
				t.Errorf("HasUnstaged = %v, want %v", status.HasUnstaged, tt.hasUnstaged)
			}
			if status.HasStaged != tt.hasStaged {
				t.Errorf("HasStaged = %v, want %v", status.HasStaged, tt.hasStaged)
			}
			if status.HasUntracked != tt.hasUntracked {
				t.Errorf("HasUntracked = %v, want %v", status.HasUntracked, tt.hasUntracked)
			}
		})
	}
}

func TestWorktreeStatus_Clean(t *testing.T) {
	tests := []struct {
		name   string
		status git.WorktreeStatus
		want   bool
	}{
		{
			name:   "clean_zero_value",
			status: git.WorktreeStatus{},
			want:   true,
		},
		{
			name:   "clean_explicit",
			status: git.WorktreeStatus{HasUnstaged: false, HasStaged: false, HasUntracked: false},
			want:   true,
		},
		{
			name:   "dirty_unstaged",
			status: git.WorktreeStatus{HasUnstaged: true},
			want:   false,
		},
		{
			name:   "dirty_staged",
			status: git.WorktreeStatus{HasStaged: true},
			want:   false,
		},
		{
			name:   "dirty_untracked",
			status: git.WorktreeStatus{HasUntracked: true},
			want:   false,
		},
		{
			name:   "dirty_all",
			status: git.WorktreeStatus{HasUnstaged: true, HasStaged: true, HasUntracked: true},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.Clean(); got != tt.want {
				t.Errorf("Clean() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWorktreeStatus_String(t *testing.T) {
	tests := []struct {
		name     string
		status   git.WorktreeStatus
		expected string
	}{
		{
			name:     "clean",
			status:   git.WorktreeStatus{},
			expected: "clean",
		},
		{
			name:     "unstaged_only",
			status:   git.WorktreeStatus{HasUnstaged: true},
			expected: "unstaged",
		},
		{
			name:     "staged_only",
			status:   git.WorktreeStatus{HasStaged: true},
			expected: "staged",
		},
		{
			name:     "untracked_only",
			status:   git.WorktreeStatus{HasUntracked: true},
			expected: "untracked",
		},
		{
			name:     "unstaged_and_staged",
			status:   git.WorktreeStatus{HasUnstaged: true, HasStaged: true},
			expected: "unstaged, staged",
		},
		{
			name:     "unstaged_and_untracked",
			status:   git.WorktreeStatus{HasUnstaged: true, HasUntracked: true},
			expected: "unstaged, untracked",
		},
		{
			name:     "staged_and_untracked",
			status:   git.WorktreeStatus{HasStaged: true, HasUntracked: true},
			expected: "staged, untracked",
		},
		{
			name:     "all_dirty",
			status:   git.WorktreeStatus{HasUnstaged: true, HasStaged: true, HasUntracked: true},
			expected: "unstaged, staged, untracked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestWorktreeStatus_Redacted(t *testing.T) {
	// Redacted should be identical to String for WorktreeStatus
	tests := []struct {
		name   string
		status git.WorktreeStatus
	}{
		{
			name:   "clean",
			status: git.WorktreeStatus{},
		},
		{
			name:   "dirty",
			status: git.WorktreeStatus{HasUnstaged: true, HasStaged: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.Redacted(); got != tt.status.String() {
				t.Errorf("Redacted() = %q, want %q (should match String())", got, tt.status.String())
			}
		})
	}
}

func TestWorktreeStatus_TypeName(t *testing.T) {
	var status git.WorktreeStatus
	if got := status.TypeName(); got != "WorktreeStatus" {
		t.Errorf("TypeName() = %v, want WorktreeStatus", got)
	}
}

func TestWorktreeStatus_IsZero(t *testing.T) {
	tests := []struct {
		name   string
		status git.WorktreeStatus
		want   bool
	}{
		{
			name:   "zero_value",
			status: git.WorktreeStatus{},
			want:   true,
		},
		{
			name:   "explicit_clean",
			status: git.WorktreeStatus{HasUnstaged: false, HasStaged: false, HasUntracked: false},
			want:   true,
		},
		{
			name:   "has_unstaged",
			status: git.WorktreeStatus{HasUnstaged: true},
			want:   false,
		},
		{
			name:   "has_staged",
			status: git.WorktreeStatus{HasStaged: true},
			want:   false,
		},
		{
			name:   "has_untracked",
			status: git.WorktreeStatus{HasUntracked: true},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsZero(); got != tt.want {
				t.Errorf("IsZero() = %v, want %v", got, tt.want)
			}
			// IsZero should match Clean for WorktreeStatus
			if got := tt.status.IsZero(); got != tt.status.Clean() {
				t.Errorf("IsZero() = %v, but Clean() = %v (should match)", got, tt.status.Clean())
			}
		})
	}
}

func TestWorktreeStatus_Equal(t *testing.T) {
	tests := []struct {
		name    string
		status1 git.WorktreeStatus
		status2 any
		want    bool
	}{
		{
			name:    "equal_clean",
			status1: git.WorktreeStatus{},
			status2: git.WorktreeStatus{},
			want:    true,
		},
		{
			name:    "equal_dirty",
			status1: git.WorktreeStatus{HasUnstaged: true, HasStaged: true},
			status2: git.WorktreeStatus{HasUnstaged: true, HasStaged: true},
			want:    true,
		},
		{
			name:    "different_unstaged",
			status1: git.WorktreeStatus{HasUnstaged: true},
			status2: git.WorktreeStatus{HasUnstaged: false},
			want:    false,
		},
		{
			name:    "different_staged",
			status1: git.WorktreeStatus{HasStaged: true},
			status2: git.WorktreeStatus{HasStaged: false},
			want:    false,
		},
		{
			name:    "different_untracked",
			status1: git.WorktreeStatus{HasUntracked: true},
			status2: git.WorktreeStatus{HasUntracked: false},
			want:    false,
		},
		{
			name:    "pointer_equal",
			status1: git.WorktreeStatus{HasUnstaged: true},
			status2: &git.WorktreeStatus{HasUnstaged: true},
			want:    true,
		},
		{
			name:    "nil_pointer",
			status1: git.WorktreeStatus{},
			status2: (*git.WorktreeStatus)(nil),
			want:    false,
		},
		{
			name:    "different_type",
			status1: git.WorktreeStatus{},
			status2: "not a status",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status1.Equal(tt.status2); got != tt.want {
				t.Errorf("Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWorktreeStatus_Validate(t *testing.T) {
	// All states are valid for WorktreeStatus
	tests := []struct {
		name   string
		status git.WorktreeStatus
	}{
		{
			name:   "clean",
			status: git.WorktreeStatus{},
		},
		{
			name:   "dirty_unstaged",
			status: git.WorktreeStatus{HasUnstaged: true},
		},
		{
			name:   "dirty_staged",
			status: git.WorktreeStatus{HasStaged: true},
		},
		{
			name:   "dirty_untracked",
			status: git.WorktreeStatus{HasUntracked: true},
		},
		{
			name:   "all_dirty",
			status: git.WorktreeStatus{HasUnstaged: true, HasStaged: true, HasUntracked: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.status.Validate(); err != nil {
				t.Errorf("Validate() unexpected error = %v (all states should be valid)", err)
			}
		})
	}
}

func TestWorktreeStatus_JSON_RoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		status git.WorktreeStatus
	}{
		{
			name:   "clean",
			status: git.WorktreeStatus{},
		},
		{
			name:   "dirty_all",
			status: git.WorktreeStatus{HasUnstaged: true, HasStaged: true, HasUntracked: true},
		},
		{
			name:   "dirty_partial",
			status: git.WorktreeStatus{HasUnstaged: true, HasStaged: false, HasUntracked: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.status)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			var decoded git.WorktreeStatus
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			if !decoded.Equal(tt.status) {
				t.Errorf("Round trip failed:\ngot  %+v\nwant %+v", decoded, tt.status)
			}
		})
	}
}

func TestWorktreeStatus_JSON_Content(t *testing.T) {
	status := git.WorktreeStatus{HasUnstaged: true, HasStaged: false, HasUntracked: true}
	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Verify JSON contains expected fields
	jsonStr := string(data)
	if !strings.Contains(jsonStr, `"has_unstaged":true`) {
		t.Errorf("JSON missing has_unstaged field: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"has_staged":false`) {
		t.Errorf("JSON missing has_staged field: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"has_untracked":true`) {
		t.Errorf("JSON missing has_untracked field: %s", jsonStr)
	}
}

func TestWorktreeStatus_YAML_RoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		status git.WorktreeStatus
	}{
		{
			name:   "clean",
			status: git.WorktreeStatus{},
		},
		{
			name:   "dirty_all",
			status: git.WorktreeStatus{HasUnstaged: true, HasStaged: true, HasUntracked: true},
		},
		{
			name:   "dirty_partial",
			status: git.WorktreeStatus{HasUnstaged: true, HasStaged: false, HasUntracked: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := yaml.Marshal(tt.status)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			var decoded git.WorktreeStatus
			if err := yaml.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			if !decoded.Equal(tt.status) {
				t.Errorf("Round trip failed:\ngot  %+v\nwant %+v", decoded, tt.status)
			}
		})
	}
}

func TestWorktreeStatus_CommonScenarios(t *testing.T) {
	scenarios := []struct {
		name        string
		status      git.WorktreeStatus
		shouldClean bool
		description string
	}{
		{
			name:        "ready_for_release",
			status:      git.WorktreeStatus{},
			shouldClean: true,
			description: "Clean working tree ready for release",
		},
		{
			name:        "forgot_to_add",
			status:      git.WorktreeStatus{HasUnstaged: true},
			shouldClean: false,
			description: "Modified files not staged",
		},
		{
			name:        "forgot_to_commit",
			status:      git.WorktreeStatus{HasStaged: true},
			shouldClean: false,
			description: "Staged changes not committed",
		},
		{
			name:        "new_files_present",
			status:      git.WorktreeStatus{HasUntracked: true},
			shouldClean: false,
			description: "Untracked files in working directory",
		},
		{
			name:        "work_in_progress",
			status:      git.WorktreeStatus{HasUnstaged: true, HasStaged: true, HasUntracked: true},
			shouldClean: false,
			description: "Active development with all types of changes",
		},
		{
			name:        "staging_in_progress",
			status:      git.WorktreeStatus{HasUnstaged: true, HasStaged: true},
			shouldClean: false,
			description: "Partially staged changes",
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			if got := sc.status.Clean(); got != sc.shouldClean {
				t.Errorf("Clean() = %v, want %v for %s", got, sc.shouldClean, sc.description)
			}

			// Verify String() provides useful information
			str := sc.status.String()
			if str == "" {
				t.Error("String() returned empty string")
			}
			if sc.shouldClean && str != "clean" {
				t.Errorf("String() = %q for clean status, want \"clean\"", str)
			}
		})
	}
}
