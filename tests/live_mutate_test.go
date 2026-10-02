package tests

import (
	"os"
	"testing"
)

// liveMutateEnvVar is the runtime opt-in that gates the full-live-with-
// OG-containment tier of tests: TEST_MODE=live alone stays read-only-safe (skipUnlessLive's
// existing gate covers get/search/lifecycle-on-disposable-resources tests
// already committed), and a SECOND, explicit opt-in is required before a
// test is allowed to POST/PUT a brand-new mutating payload against a real
// tenant. This keeps a plain `TEST_MODE=live` run from ever creating
// smartgroups/profiles nobody asked for.
//
// This helper lives in a _test.go file (not a plain live_mutate.go) because
// it necessarily depends on two test-only symbols -- getEnvOrDefault
// (profile_crud_test.go) and skipUnlessLive (live_resource_lifecycle_test.go)
// -- and a plain .go file referencing _test.go-only symbols fails `go build
// ./...` at the repo root (verified: moving this same code into a non-test
// file breaks the top-level build with "undefined: getEnvOrDefault").
const liveMutateEnvVar = "LIVE_MUTATE"

// liveMutateEnabled reports whether LIVE_MUTATE=1 is set. Extracted as a
// pure function (no *testing.T dependency) so the opt-in logic itself is
// unit-testable without fighting t.Skip's control flow -- see
// TestLiveMutateEnabled below.
func liveMutateEnabled() bool {
	return getEnvOrDefault(liveMutateEnvVar, "") == "1"
}

// skipUnlessLiveMutate is the shared gate for every hand-written live-mutate
// test added under Task C: it first applies skipUnlessLive's existing
// TEST_MODE=live + -short gate (tests/live_resource_lifecycle_test.go), then
// additionally skips (never fails) unless LIVE_MUTATE=1 is also set. A test
// gated by this function therefore only executes its body when a caller has
// opted into BOTH TEST_MODE=live AND LIVE_MUTATE=1 -- normal CI (`go test
// -short` or a plain `TEST_MODE=live` run) always skips it.
func skipUnlessLiveMutate(t *testing.T) {
	t.Helper()
	skipUnlessLive(t)
	if !liveMutateEnabled() {
		t.Skip("LIVE_MUTATE=1 not set; skipping live-mutate test")
	}
}

// TestLiveMutateEnabled exercises the pure LIVE_MUTATE=1 opt-in check
// directly (rather than skipUnlessLiveMutate itself, which calls t.Skip and
// so can't be asserted against from within the same test body). Covers: the
// env var unset, set to something other than "1", and set to exactly "1".
func TestLiveMutateEnabled(t *testing.T) {
	const envVar = "LIVE_MUTATE"
	original, hadOriginal := os.LookupEnv(envVar)
	t.Cleanup(func() {
		if hadOriginal {
			_ = os.Setenv(envVar, original)
		} else {
			_ = os.Unsetenv(envVar)
		}
	})

	cases := []struct {
		name  string
		value string
		unset bool
		want  bool
	}{
		{name: "unset", unset: true, want: false},
		{name: "empty", value: "", want: false},
		{name: "true_word_not_recognized", value: "true", want: false},
		{name: "zero_not_recognized", value: "0", want: false},
		{name: "one_enables", value: "1", want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.unset {
				_ = os.Unsetenv(envVar)
			} else {
				_ = os.Setenv(envVar, tc.value)
			}
			if got := liveMutateEnabled(); got != tc.want {
				t.Errorf("liveMutateEnabled() = %v, want %v (LIVE_MUTATE=%q)", got, tc.want, tc.value)
			}
		})
	}
}

// TestSkipUnlessLiveMutate_SkipsWithoutOptIn documents (via a sub-test that
// asserts it skips) that skipUnlessLiveMutate never runs its caller's body
// unless both TEST_MODE=live and LIVE_MUTATE=1 are set -- neither is set in
// a normal `go test` invocation, so this sub-test always skips itself, and
// that skip IS the assertion (if skipUnlessLiveMutate ever stopped calling
// t.Skip() here, this sub-test would instead fall through and fail below).
func TestSkipUnlessLiveMutate_SkipsWithoutOptIn(t *testing.T) {
	for _, v := range []string{"TEST_MODE", "LIVE_MUTATE"} {
		original, hadOriginal := os.LookupEnv(v)
		_ = os.Unsetenv(v)
		t.Cleanup(func(v, original string, hadOriginal bool) func() {
			return func() {
				if hadOriginal {
					_ = os.Setenv(v, original)
				} else {
					_ = os.Unsetenv(v)
				}
			}
		}(v, original, hadOriginal))
	}

	t.Run("gated", func(t *testing.T) {
		skipUnlessLiveMutate(t)
		t.Fatal("skipUnlessLiveMutate did not skip with TEST_MODE/LIVE_MUTATE unset")
	})
}
