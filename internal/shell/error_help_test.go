package shell

import (
	"testing"
)

func TestShouldOfferErrorHelp(t *testing.T) {
	tests := []struct {
		name          string
		autoErrorHelp bool
		result        execResult
		want          bool
	}{
		{
			name:          "disabled via config",
			autoErrorHelp: false,
			result:        execResult{exitCode: 2, stderr: "error: something broke"},
			want:          false,
		},
		{
			name:          "exit code 0 — success",
			autoErrorHelp: true,
			result:        execResult{exitCode: 0, stderr: ""},
			want:          false,
		},
		{
			name:          "exit code 130 — Ctrl-C",
			autoErrorHelp: true,
			result:        execResult{exitCode: 130, stderr: ""},
			want:          false,
		},
		{
			name:          "exit code 127 — command not found handled separately",
			autoErrorHelp: true,
			result:        execResult{exitCode: 127, stderr: "command not found"},
			want:          false,
		},
		{
			name:          "exit code 1 with empty stderr — grep no-match",
			autoErrorHelp: true,
			result:        execResult{exitCode: 1, stderr: ""},
			want:          false,
		},
		{
			name:          "exit code 1 with whitespace-only stderr",
			autoErrorHelp: true,
			result:        execResult{exitCode: 1, stderr: "  \n  "},
			want:          false,
		},
		{
			name:          "exit code 2 with empty stderr — no useful context",
			autoErrorHelp: true,
			result:        execResult{exitCode: 2, stderr: ""},
			want:          false,
		},
		{
			name:          "exit code 1 with stderr — real error",
			autoErrorHelp: true,
			result:        execResult{exitCode: 1, stderr: "error: file not found"},
			want:          true,
		},
		{
			name:          "exit code 2 with stderr — real error",
			autoErrorHelp: true,
			result:        execResult{exitCode: 2, stderr: "fatal: not a git repository"},
			want:          true,
		},
		{
			name:          "high exit code with stderr",
			autoErrorHelp: true,
			result:        execResult{exitCode: 126, stderr: "permission denied"},
			want:          true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &Shell{autoErrorHelp: tc.autoErrorHelp}
			got := s.shouldOfferErrorHelp(tc.result)
			if got != tc.want {
				t.Errorf("shouldOfferErrorHelp() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFormatErrorQuery(t *testing.T) {
	query := formatErrorQuery("go build ./...", 2, "  syntax error: unexpected }\n")

	if query == "" {
		t.Fatal("formatErrorQuery returned empty string")
	}

	// Should contain the command.
	if got := query; !contains(got, "`go build ./...`") {
		t.Errorf("query should contain the command, got: %s", got)
	}

	// Should contain the exit code.
	if !contains(query, "exit code 2") {
		t.Errorf("query should contain exit code, got: %s", query)
	}

	// Should contain the trimmed stderr.
	if !contains(query, "syntax error: unexpected }") {
		t.Errorf("query should contain stderr, got: %s", query)
	}

	// Should not have leading/trailing whitespace in stderr block.
	if contains(query, "  syntax error") {
		t.Errorf("query should trim stderr whitespace, got: %s", query)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
