package errors

import (
	"errors"
	"strings"
	"testing"
)

func TestHintBuilder(t *testing.T) {
	t.Run("single hint", func(t *testing.T) {
		builder := NewHintBuilder()
		builder.Add("First hint")

		result := builder.Build()
		if result != "First hint" {
			t.Errorf("Build() = %q, want %q", result, "First hint")
		}
	})

	t.Run("multiple hints", func(t *testing.T) {
		builder := NewHintBuilder()
		builder.Add("First hint")
		builder.Add("Second hint")

		result := builder.Build()
		expected := "First hint\nSecond hint"
		if result != expected {
			t.Errorf("Build() = %q, want %q", result, expected)
		}
	})

	t.Run("empty builder", func(t *testing.T) {
		builder := NewHintBuilder()
		result := builder.Build()
		if result != "" {
			t.Errorf("Build() = %q, want empty string", result)
		}
	})

	t.Run("conditional hints", func(t *testing.T) {
		builder := NewHintBuilder()
		builder.AddIf(true, "Included hint")
		builder.AddIf(false, "Excluded hint")

		result := builder.Build()
		if result != "Included hint" {
			t.Errorf("Build() = %q, want %q", result, "Included hint")
		}
	})

	t.Run("formatted hints", func(t *testing.T) {
		builder := NewHintBuilder()
		builder.AddFormatted("Polecat '%s' not found", "Toast")

		result := builder.Build()
		expected := "Polecat 'Toast' not found"
		if result != expected {
			t.Errorf("Build() = %q, want %q", result, expected)
		}
	})

	t.Run("chaining", func(t *testing.T) {
		result := NewHintBuilder().
			Add("First").
			AddIf(true, "Second").
			AddFormatted("Third: %s", "test").
			Build()

		expected := "First\nSecond\nThird: test"
		if result != expected {
			t.Errorf("Build() = %q, want %q", result, expected)
		}
	})
}

func TestWithPolecatNotFoundHint(t *testing.T) {
	hint := WithPolecatNotFoundHint("Toast")
	if !strings.Contains(hint, "Toast") {
		t.Errorf("hint should contain polecat name 'Toast'")
	}
	if !strings.Contains(hint, "gt polecat list") {
		t.Errorf("hint should mention 'gt polecat list' command")
	}
}

func TestWithRigNotFoundHint(t *testing.T) {
	hint := WithRigNotFoundHint("myrig")
	if !strings.Contains(hint, "myrig") {
		t.Errorf("hint should contain rig name 'myrig'")
	}
	if !strings.Contains(hint, "gt rig list") {
		t.Errorf("hint should mention 'gt rig list' command")
	}
}

func TestWithIssueNotFoundHint(t *testing.T) {
	hint := WithIssueNotFoundHint("gt-123")
	if !strings.Contains(hint, "gt-123") {
		t.Errorf("hint should contain issue ID 'gt-123'")
	}
	if !strings.Contains(hint, "bd list") {
		t.Errorf("hint should mention 'bd list' command")
	}
}

func TestWithMRNotFoundHint(t *testing.T) {
	hint := WithMRNotFoundHint("mr-456")
	if !strings.Contains(hint, "mr-456") {
		t.Errorf("hint should contain MR ID 'mr-456'")
	}
	if !strings.Contains(hint, "gt mq list") {
		t.Errorf("hint should mention 'gt mq list' command")
	}
}

func TestWithGitBranchHint(t *testing.T) {
	hint := WithGitBranchHint("feature/new-feature")
	if !strings.Contains(hint, "feature/new-feature") {
		t.Errorf("hint should contain branch name")
	}
	if !strings.Contains(hint, "git branch") {
		t.Errorf("hint should mention 'git branch' command")
	}
}

func TestWithCommandNotFoundHint(t *testing.T) {
	tests := []struct {
		name        string
		cmd         string
		expectedStr string
	}{
		{
			name:        "beads command",
			cmd:         "bd",
			expectedStr: "brew install beads",
		},
		{
			name:        "tmux command",
			cmd:         "tmux",
			expectedStr: "brew install tmux",
		},
		{
			name:        "git command",
			cmd:         "git",
			expectedStr: "brew install git",
		},
		{
			name:        "unknown command",
			cmd:         "unknown",
			expectedStr: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hint := WithCommandNotFoundHint(tt.cmd)
			if !strings.Contains(hint, tt.expectedStr) {
				t.Errorf("hint should contain %q, got %q", tt.expectedStr, hint)
			}
		})
	}
}

func TestSuggestRetry(t *testing.T) {
	t.Run("with reason", func(t *testing.T) {
		hint := SuggestRetry("git push", "network timeout")
		if !strings.Contains(hint, "git push") {
			t.Errorf("hint should contain operation name")
		}
		if !strings.Contains(hint, "network timeout") {
			t.Errorf("hint should contain reason")
		}
		if !strings.Contains(hint, "retried") && !strings.Contains(strings.ToLower(hint), "retry") {
			t.Errorf("hint should suggest retry, got: %s", hint)
		}
	})

	t.Run("without reason", func(t *testing.T) {
		hint := SuggestRetry("database query", "")
		if !strings.Contains(hint, "database query") {
			t.Errorf("hint should contain operation name")
		}
		if !strings.Contains(hint, "retried") && !strings.Contains(strings.ToLower(hint), "retry") {
			t.Errorf("hint should suggest retry, got: %s", hint)
		}
	})
}

func TestSuggestManualIntervention(t *testing.T) {
	steps := "1. Run git status\n2. Resolve conflicts\n3. Run git commit"
	hint := SuggestManualIntervention("merge", steps)

	if !strings.Contains(hint, "merge") {
		t.Errorf("hint should contain operation name")
	}
	if !strings.Contains(hint, steps) {
		t.Errorf("hint should contain steps")
	}
}

func TestSuggestCheckCommand(t *testing.T) {
	hint := SuggestCheckCommand("Check session status", "gt status")
	if !strings.Contains(hint, "Check session status") {
		t.Errorf("hint should contain description")
	}
	if !strings.Contains(hint, "gt status") {
		t.Errorf("hint should contain command")
	}
}

func TestSuggestDocumentation(t *testing.T) {
	t.Run("with URL", func(t *testing.T) {
		hint := SuggestDocumentation("polecats", "https://example.com/docs/polecats")
		if !strings.Contains(hint, "polecats") {
			t.Errorf("hint should contain topic")
		}
		if !strings.Contains(hint, "https://example.com/docs/polecats") {
			t.Errorf("hint should contain URL")
		}
	})

	t.Run("without URL", func(t *testing.T) {
		hint := SuggestDocumentation("workflows", "")
		if !strings.Contains(hint, "workflows") {
			t.Errorf("hint should contain topic")
		}
		if !strings.Contains(hint, "documentation") {
			t.Errorf("hint should mention documentation")
		}
	})
}

func TestEnrichErrorWithHint(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		expectedStr string
	}{
		{
			name:        "nil error",
			err:         nil,
			expectedStr: "",
		},
		{
			name:        "beads not found",
			err:         errors.New("executable file not found in $PATH: bd"),
			expectedStr: "beads",
		},
		{
			name:        "tmux not found",
			err:         errors.New("command not found: tmux"),
			expectedStr: "tmux",
		},
		{
			name:        "git push failed",
			err:         errors.New("git push failed: connection refused"),
			expectedStr: "push",
		},
		{
			name:        "git pull failed",
			err:         errors.New("git pull error: network timeout"),
			expectedStr: "pull",
		},
		{
			name:        "git conflict",
			err:         errors.New("merge conflict in file.go"),
			expectedStr: "Resolve merge conflicts",
		},
		{
			name:        "connection refused",
			err:         errors.New("connection refused"),
			expectedStr: "network",
		},
		{
			name:        "timeout",
			err:         errors.New("operation timeout"),
			expectedStr: "timed out",
		},
		{
			name:        "file not found",
			err:         errors.New("no such file or directory: /path/to/file"),
			expectedStr: "file",
		},
		{
			name:        "permission denied",
			err:         errors.New("permission denied: /path/to/file"),
			expectedStr: "permission",
		},
		{
			name:        "already has hint",
			err:         User("test", "error").WithHint("existing hint"),
			expectedStr: "existing hint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enriched := EnrichErrorWithHint(tt.err)

			if tt.err == nil {
				if enriched != nil {
					t.Errorf("EnrichErrorWithHint() should return nil for nil error")
				}
				return
			}

			hint := GetHint(enriched)
			if tt.expectedStr != "" && !strings.Contains(hint, tt.expectedStr) {
				t.Errorf("enriched error hint should contain %q, got %q", tt.expectedStr, hint)
			}
		})
	}
}

func TestEnrichErrorWithHint_PreservesExistingHint(t *testing.T) {
	originalHint := "original hint"
	err := User("test", "error").WithHint(originalHint)

	enriched := EnrichErrorWithHint(err)

	hint := GetHint(enriched)
	if hint != originalHint {
		t.Errorf("EnrichErrorWithHint() should preserve existing hint, got %q, want %q", hint, originalHint)
	}
}

func TestEnrichErrorWithHint_RecognizesPatterns(t *testing.T) {
	// Test that enrichment correctly categorizes errors
	gitPushErr := errors.New("git push origin main failed")
	enriched := EnrichErrorWithHint(gitPushErr)

	if !IsTransient(enriched) {
		t.Error("Git push error should be categorized as transient")
	}

	conflictErr := errors.New("git merge conflict detected")
	enriched = EnrichErrorWithHint(conflictErr)

	if GetCategory(enriched) != CategoryUser {
		t.Errorf("Merge conflict should be categorized as user error, got: %v", GetCategory(enriched))
	}
}
