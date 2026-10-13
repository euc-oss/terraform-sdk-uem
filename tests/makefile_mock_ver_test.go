package tests

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

// The fixture corpus that `make mock-populate` (and so `make test`) stages is
// chosen by MOCK_VER. A VER on the command line still selects it, as
// `make mock-populate VER=<ver>` always did; a VER that is only in the
// environment must not, so a stray exported VER cannot flip the corpus under
// the tests.

var mockPopulateBranch = regexp.MustCompile(`git fetch -q origin (\S+)`)

// stagedCorpusBranch runs `make -n <target>` in the repository root with the
// given environment additions and command-line arguments and returns the
// orphan branch the recipe would fetch; ok is false when there is no make or
// no such target (the assembled public package has a different Makefile).
func stagedCorpusBranch(t *testing.T, target string, env []string, args ...string) (branch string, ok bool) {
	t.Helper()
	if _, err := exec.LookPath("make"); err != nil {
		return "", false
	}
	cmd := exec.Command("make", append([]string{"-n", target}, args...)...)
	cmd.Dir = ".."
	cmd.Env = append(hermeticMakeEnv(os.Environ()), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", false
	}
	m := mockPopulateBranch.FindSubmatch(out)
	if m == nil {
		return "", false
	}
	return string(m[1]), true
}

// hermeticMakeEnv drops the corpus selectors and make's own propagation
// variables. When the suite itself runs under `make test VER=<ver>`, GNU make
// passes that command-line VER to child makes through MAKEFLAGS/MAKEOVERRIDES,
// where it would still count as a command-line assignment.
func hermeticMakeEnv(environ []string) []string {
	var clean []string
	for _, e := range environ {
		switch {
		case strings.HasPrefix(e, "VER="), strings.HasPrefix(e, "MOCK_VER="), strings.HasPrefix(e, "MOCK_ORPHAN_BRANCH="),
			strings.HasPrefix(e, "MAKEFLAGS="), strings.HasPrefix(e, "MAKEOVERRIDES="),
			strings.HasPrefix(e, "MFLAGS="), strings.HasPrefix(e, "GNUMAKEFLAGS="), strings.HasPrefix(e, "MAKELEVEL="):
		default:
			clean = append(clean, e)
		}
	}
	return clean
}

func TestHermeticMakeEnvDropsMakePropagation(t *testing.T) {
	got := hermeticMakeEnv([]string{
		"PATH=/bin", "VER=26.2.0.0", "MAKEFLAGS= -- VER=26.2.0.0", "MAKEOVERRIDES=${-*-command-variables-*-}",
		"MFLAGS=-s", "GNUMAKEFLAGS=-s", "MAKELEVEL=1", "MOCK_VER=main", "MOCK_ORPHAN_BRANCH=x/y", "HOME=/h",
	})
	if strings.Join(got, " ") != "PATH=/bin HOME=/h" {
		t.Errorf("hermeticMakeEnv kept %q, want only PATH and HOME", got)
	}
}

func TestMakefileMockVerSelectsTheCorpus(t *testing.T) {
	const pinned = "chore/vendored-swagger-uem-26.4.0.0"
	if _, ok := stagedCorpusBranch(t, "mock-populate", nil); !ok {
		t.Skip("no make, or this Makefile has no mock-populate target (the assembled public package)")
	}
	for _, c := range []struct {
		name   string
		target string
		env    []string
		args   []string
		want   string
	}{
		{"default", "mock-populate", nil, nil, pinned},
		{"stray environment VER does not flip the corpus", "mock-populate", []string{"VER=26.2.0.0"}, nil, pinned},
		{"stray environment VER does not flip make test", "test", []string{"VER=26.2.0.0"}, nil, pinned},
		{"command-line VER selects the corpus", "mock-populate", nil, []string{"VER=26.2.0.0"}, "chore/vendored-swagger-uem-26.2.0.0"},
		{"command-line VER selects the corpus for make test", "test", nil, []string{"VER=25.9.0.0"}, "chore/vendored-swagger-uem-25.9.0.0"},
		{"MOCK_VER on the command line", "mock-populate", nil, []string{"MOCK_VER=25.9.0.0"}, "chore/vendored-swagger-uem-25.9.0.0"},
		{"MOCK_VER in the environment", "mock-populate", []string{"MOCK_VER=25.9.0.0"}, nil, "chore/vendored-swagger-uem-25.9.0.0"},
		{"MOCK_VER beats a command-line VER", "mock-populate", nil, []string{"VER=26.2.0.0", "MOCK_VER=main"}, "chore/vendored-swagger-uem-main"},
		{"empty MOCK_VER selects main", "mock-populate", nil, []string{"MOCK_VER="}, "chore/vendored-swagger-uem-main"},
		{"an explicit branch wins", "mock-populate", []string{"VER=26.2.0.0"}, []string{"MOCK_ORPHAN_BRANCH=x/y"}, "x/y"},
		{"an explicit branch in the environment wins", "mock-populate", []string{"MOCK_ORPHAN_BRANCH=x/y"}, nil, "x/y"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, ok := stagedCorpusBranch(t, c.target, c.env, c.args...)
			if !ok {
				t.Fatalf("make -n %s printed no corpus fetch", c.target)
			}
			if got != c.want {
				t.Errorf("staged corpus branch = %q, want %q", got, c.want)
			}
		})
	}
}
