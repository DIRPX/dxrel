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
	"time"

	"dirpx.dev/dxrel/dxcore/model/git"
	"gopkg.in/yaml.v3"
)

func TestNewCommit(t *testing.T) {
	hash := git.Hash("a1b2c3d4e5f67890abcdef1234567890abcdef12")
	parent := git.Hash("1234567890abcdef1234567890abcdef12345678")
	author := git.Signature{Name: "Jane", Email: "jane@example.com", When: time.Now()}
	committer := git.Signature{Name: "John", Email: "john@example.com", When: time.Now()}
	changes := []git.FileChange{{Path: "src/file.go", Kind: git.FileChangeAdded}}

	tests := []struct {
		name      string
		hash      git.Hash
		parents   []git.Hash
		author    git.Signature
		committer git.Signature
		message   string
		summary   string
		changes   []git.FileChange
		wantErr   bool
	}{
		{
			name:      "valid_normal_commit",
			hash:      hash,
			parents:   []git.Hash{parent},
			author:    author,
			committer: committer,
			message:   "feat: add feature\n\nDetailed description.",
			summary:   "feat: add feature",
			changes:   changes,
			wantErr:   false,
		},
		{
			name:      "valid_auto_summary",
			hash:      hash,
			parents:   []git.Hash{parent},
			author:    author,
			committer: committer,
			message:   "fix: bug fix\n\nBody text.",
			summary:   "", // Will be auto-extracted
			changes:   changes,
			wantErr:   false,
		},
		{
			name:      "valid_initial_commit",
			hash:      hash,
			parents:   []git.Hash{}, // No parents
			author:    author,
			committer: committer,
			message:   "Initial commit",
			summary:   "Initial commit",
			changes:   changes,
			wantErr:   false,
		},
		{
			name:      "valid_merge_commit",
			hash:      hash,
			parents:   []git.Hash{parent, hash},
			author:    author,
			committer: committer,
			message:   "Merge branch 'feature' into main",
			summary:   "Merge branch 'feature' into main",
			changes:   []git.FileChange{},
			wantErr:   false,
		},
		{
			name:      "invalid_empty_hash",
			hash:      "",
			parents:   []git.Hash{parent},
			author:    author,
			committer: committer,
			message:   "feat: add",
			summary:   "feat: add",
			changes:   changes,
			wantErr:   true,
		},
		{
			name:      "invalid_empty_message",
			hash:      hash,
			parents:   []git.Hash{parent},
			author:    author,
			committer: committer,
			message:   "",
			summary:   "",
			changes:   changes,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := git.NewCommit(tt.hash, tt.parents, tt.author, tt.committer, tt.message, tt.summary, tt.changes)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewCommit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.Hash != tt.hash {
				t.Errorf("NewCommit().Hash = %v, want %v", got.Hash, tt.hash)
			}
		})
	}
}

func TestCommit_String(t *testing.T) {
	hash := git.Hash("a1b2c3d4e5f67890abcdef1234567890abcdef12")
	parent := git.Hash("1234567890abcdef1234567890abcdef12345678")
	author := git.Signature{Name: "Jane", Email: "jane@example.com", When: time.Now()}
	committer := git.Signature{Name: "John", Email: "john@example.com", When: time.Now()}

	commit := git.Commit{
		Hash:      hash,
		Parents:   []git.Hash{parent},
		Author:    author,
		Committer: committer,
		Message:   "feat: add feature",
		Summary:   "feat: add feature",
	}

	str := commit.String()
	if !strings.Contains(str, "feat: add feature") {
		t.Errorf("Commit.String() doesn't contain summary: %s", str)
	}
	if !strings.Contains(str, "Parents:1") {
		t.Errorf("Commit.String() doesn't contain parent count: %s", str)
	}
}

func TestCommit_TypeName(t *testing.T) {
	var commit git.Commit
	if got := commit.TypeName(); got != "Commit" {
		t.Errorf("Commit.TypeName() = %v, want Commit", got)
	}
}

func TestCommit_IsZero(t *testing.T) {
	tests := []struct {
		name   string
		commit git.Commit
		want   bool
	}{
		{
			name:   "zero_value",
			commit: git.Commit{},
			want:   true,
		},
		{
			name: "non_zero",
			commit: git.Commit{
				Hash:    "a1b2c3d4e5f67890abcdef1234567890abcdef12",
				Message: "test",
				Summary: "test",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.commit.IsZero(); got != tt.want {
				t.Errorf("Commit.IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommit_Equal(t *testing.T) {
	hash1 := git.Hash("a1b2c3d4e5f67890abcdef1234567890abcdef12")
	hash2 := git.Hash("1234567890abcdef1234567890abcdef12345678")
	parent := git.Hash("fedcba0987654321fedcba0987654321fedcba09")
	author := git.Signature{Name: "Jane", Email: "jane@example.com", When: time.Now()}
	committer := git.Signature{Name: "John", Email: "john@example.com", When: time.Now()}

	tests := []struct {
		name    string
		commit1 git.Commit
		commit2 git.Commit
		want    bool
	}{
		{
			name: "equal_commits",
			commit1: git.Commit{
				Hash:      hash1,
				Parents:   []git.Hash{parent},
				Author:    author,
				Committer: committer,
				Message:   "test",
				Summary:   "test",
			},
			commit2: git.Commit{
				Hash:      hash1,
				Parents:   []git.Hash{parent},
				Author:    author,
				Committer: committer,
				Message:   "test",
				Summary:   "test",
			},
			want: true,
		},
		{
			name: "different_hash",
			commit1: git.Commit{
				Hash:    hash1,
				Message: "test",
				Summary: "test",
			},
			commit2: git.Commit{
				Hash:    hash2,
				Message: "test",
				Summary: "test",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.commit1.Equal(tt.commit2); got != tt.want {
				t.Errorf("Commit.Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommit_Validate(t *testing.T) {
	hash := git.Hash("a1b2c3d4e5f67890abcdef1234567890abcdef12")
	parent := git.Hash("1234567890abcdef1234567890abcdef12345678")
	author := git.Signature{Name: "Jane", Email: "jane@example.com", When: time.Now()}
	committer := git.Signature{Name: "John", Email: "john@example.com", When: time.Now()}
	changes := []git.FileChange{{Path: "src/file.go", Kind: git.FileChangeAdded}}

	tests := []struct {
		name    string
		commit  git.Commit
		wantErr bool
	}{
		{
			name: "valid_normal_commit",
			commit: git.Commit{
				Hash:      hash,
				Parents:   []git.Hash{parent},
				Author:    author,
				Committer: committer,
				Message:   "feat: add feature\n\nDetailed description.",
				Summary:   "feat: add feature",
				Changes:   changes,
			},
			wantErr: false,
		},
		{
			name: "valid_initial_commit",
			commit: git.Commit{
				Hash:      hash,
				Parents:   []git.Hash{},
				Author:    author,
				Committer: committer,
				Message:   "Initial commit",
				Summary:   "Initial commit",
				Changes:   changes,
			},
			wantErr: false,
		},
		{
			name: "invalid_empty_hash",
			commit: git.Commit{
				Hash:      "",
				Parents:   []git.Hash{parent},
				Author:    author,
				Committer: committer,
				Message:   "test",
				Summary:   "test",
			},
			wantErr: true,
		},
		{
			name: "invalid_empty_message",
			commit: git.Commit{
				Hash:      hash,
				Parents:   []git.Hash{parent},
				Author:    author,
				Committer: committer,
				Message:   "",
				Summary:   "test",
			},
			wantErr: true,
		},
		{
			name: "invalid_empty_summary",
			commit: git.Commit{
				Hash:      hash,
				Parents:   []git.Hash{parent},
				Author:    author,
				Committer: committer,
				Message:   "test",
				Summary:   "",
			},
			wantErr: true,
		},
		{
			name: "invalid_summary_mismatch",
			commit: git.Commit{
				Hash:      hash,
				Parents:   []git.Hash{parent},
				Author:    author,
				Committer: committer,
				Message:   "feat: add feature",
				Summary:   "wrong summary",
			},
			wantErr: true,
		},
		{
			name: "invalid_summary_with_newline",
			commit: git.Commit{
				Hash:      hash,
				Parents:   []git.Hash{parent},
				Author:    author,
				Committer: committer,
				Message:   "test\nmore",
				Summary:   "test\nmore",
			},
			wantErr: true,
		},
		{
			name: "invalid_message_crlf",
			commit: git.Commit{
				Hash:      hash,
				Parents:   []git.Hash{parent},
				Author:    author,
				Committer: committer,
				Message:   "test\r\nmore",
				Summary:   "test",
			},
			wantErr: true,
		},
		{
			name: "invalid_zero_author",
			commit: git.Commit{
				Hash:      hash,
				Parents:   []git.Hash{parent},
				Author:    git.Signature{},
				Committer: committer,
				Message:   "test",
				Summary:   "test",
			},
			wantErr: true,
		},
		{
			name: "invalid_zero_committer",
			commit: git.Commit{
				Hash:      hash,
				Parents:   []git.Hash{parent},
				Author:    author,
				Committer: git.Signature{},
				Message:   "test",
				Summary:   "test",
			},
			wantErr: true,
		},
		{
			name: "invalid_too_many_parents",
			commit: git.Commit{
				Hash:      hash,
				Parents:   make([]git.Hash, 65),
				Author:    author,
				Committer: committer,
				Message:   "test",
				Summary:   "test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.commit.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Commit.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCommit_JSON_RoundTrip(t *testing.T) {
	hash := git.Hash("a1b2c3d4e5f67890abcdef1234567890abcdef12")
	parent := git.Hash("1234567890abcdef1234567890abcdef12345678")
	author := git.Signature{Name: "Jane", Email: "jane@example.com", When: time.Now().UTC()}
	committer := git.Signature{Name: "John", Email: "john@example.com", When: time.Now().UTC()}
	changes := []git.FileChange{{Path: "src/file.go", Kind: git.FileChangeAdded}}

	commit := git.Commit{
		Hash:      hash,
		Parents:   []git.Hash{parent},
		Author:    author,
		Committer: committer,
		Message:   "feat: add feature\n\nDetailed description.",
		Summary:   "feat: add feature",
		Changes:   changes,
	}

	data, err := json.Marshal(commit)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded git.Commit
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if !decoded.Equal(commit) {
		t.Errorf("Round trip failed:\ngot  %+v\nwant %+v", decoded, commit)
	}
}

func TestCommit_YAML_RoundTrip(t *testing.T) {
	hash := git.Hash("a1b2c3d4e5f67890abcdef1234567890abcdef12")
	parent := git.Hash("1234567890abcdef1234567890abcdef12345678")
	author := git.Signature{Name: "Jane", Email: "jane@example.com", When: time.Now().UTC()}
	committer := git.Signature{Name: "John", Email: "john@example.com", When: time.Now().UTC()}

	commit := git.Commit{
		Hash:      hash,
		Parents:   []git.Hash{parent},
		Author:    author,
		Committer: committer,
		Message:   "feat: add feature",
		Summary:   "feat: add feature",
		Changes:   []git.FileChange{},
	}

	data, err := yaml.Marshal(commit)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded git.Commit
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if !decoded.Equal(commit) {
		t.Errorf("Round trip failed:\ngot  %+v\nwant %+v", decoded, commit)
	}
}

func TestCommit_CommonScenarios(t *testing.T) {
	hash := git.Hash("a1b2c3d4e5f67890abcdef1234567890abcdef12")
	parent := git.Hash("1234567890abcdef1234567890abcdef12345678")
	author := git.Signature{Name: "Jane Doe", Email: "jane@example.com", When: time.Now()}
	committer := git.Signature{Name: "Jane Doe", Email: "jane@example.com", When: time.Now()}

	scenarios := []struct {
		name    string
		commit  git.Commit
		valid   bool
	}{
		{
			name: "conventional_commit_feat",
			commit: git.Commit{
				Hash:      hash,
				Parents:   []git.Hash{parent},
				Author:    author,
				Committer: committer,
				Message:   "feat(api): add user authentication\n\nImplements OAuth2 flow.\n\nFixes #123",
				Summary:   "feat(api): add user authentication",
				Changes:   []git.FileChange{{Path: "api/auth.go", Kind: git.FileChangeAdded}},
			},
			valid: true,
		},
		{
			name: "conventional_commit_fix",
			commit: git.Commit{
				Hash:      hash,
				Parents:   []git.Hash{parent},
				Author:    author,
				Committer: committer,
				Message:   "fix: correct typo in README",
				Summary:   "fix: correct typo in README",
				Changes:   []git.FileChange{{Path: "README.md", Kind: git.FileChangeModified}},
			},
			valid: true,
		},
		{
			name: "merge_commit",
			commit: git.Commit{
				Hash:      hash,
				Parents:   []git.Hash{parent, hash},
				Author:    author,
				Committer: committer,
				Message:   "Merge pull request #42 from feature-branch\n\nAdd new feature",
				Summary:   "Merge pull request #42 from feature-branch",
				Changes:   []git.FileChange{},
			},
			valid: true,
		},
		{
			name: "initial_commit",
			commit: git.Commit{
				Hash:      hash,
				Parents:   []git.Hash{},
				Author:    author,
				Committer: committer,
				Message:   "Initial commit",
				Summary:   "Initial commit",
				Changes:   []git.FileChange{{Path: "README.md", Kind: git.FileChangeAdded}},
			},
			valid: true,
		},
		{
			name: "long_commit_message",
			commit: git.Commit{
				Hash:      hash,
				Parents:   []git.Hash{parent},
				Author:    author,
				Committer: committer,
				Message:   "feat: add feature\n\n" + strings.Repeat("More details.\n", 100),
				Summary:   "feat: add feature",
				Changes:   []git.FileChange{{Path: "src/feature.go", Kind: git.FileChangeAdded}},
			},
			valid: true,
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			err := sc.commit.Validate()
			if sc.valid && err != nil {
				t.Errorf("Expected valid commit, got error: %v", err)
			}
			if !sc.valid && err == nil {
				t.Errorf("Expected invalid commit, got nil error")
			}
		})
	}
}
