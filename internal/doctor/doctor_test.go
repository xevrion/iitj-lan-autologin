package doctor

import "testing"

func TestFindRecentProblem(t *testing.T) {
	t.Parallel()

	lines := []string{
		"[2026-03-23 12:00:00] already authenticated",
		"[2026-03-23 12:05:00] warning: retrying",
		"[2026-03-23 12:10:00] error: trigger check failed",
	}

	got := findRecentProblem(lines)
	want := "[2026-03-23 12:10:00] error: trigger check failed"
	if got != want {
		t.Fatalf("findRecentProblem() = %q, want %q", got, want)
	}
}

func TestFindRecentProblemNoFailure(t *testing.T) {
	t.Parallel()

	lines := []string{
		"already authenticated",
		"login successful",
	}
	if got := findRecentProblem(lines); got != "" {
		t.Fatalf("findRecentProblem() = %q, want empty", got)
	}
}
