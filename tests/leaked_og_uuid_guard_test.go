package tests

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// leakedOGUUIDSHA256 is the sha256 of the lower-cased dash-form UUID of a
// real tenant organization group that was once captured un-anonymized. The
// guard carries only the hash, so this file never contains the value itself.
var leakedOGUUIDSHA256 = map[string]bool{
	"90cd82d2e5f5ce88e288e50bdbf629b86a19e39091711f1fdec30cd090bbd3e5": true,
}

// leakedOGUUIDExemptFiles are tracked files that name the organization group
// on purpose: they document or drive live captures and never ship. Each is
// proven absent from the assembled public packages by
// TestLeakedOGUUIDExemptFilesAreNotShipped.
var leakedOGUUIDExemptFiles = map[string]string{
	".claude/skills/capture-mock-data/SKILL.md":  "live-capture instructions; not a sync carry-over path",
	".cursor/skills/capture-mock-data/SKILL.md":  "live-capture instructions; not a sync carry-over path",
	".cursor/skills/capture-mock-data/README.md": "live-capture instructions; not a sync carry-over path",
	"scripts/capture-mam-apps-v2.sh":             "live-capture script; not a sync carry-over path",
	"scripts/capture-smartgroups.sh":             "live-capture script; not a sync carry-over path",
}

// leakedUUIDHits reports every dash-form UUID in data whose lower-cased
// sha256 is in hashes.
func leakedUUIDHits(label string, data []byte, hashes map[string]bool) []string {
	var hits []string
	for i, line := range strings.Split(string(data), "\n") {
		for _, m := range dashFormUUIDPattern.FindAllString(line, -1) {
			sum := sha256.Sum256([]byte(strings.ToLower(m)))
			if hashes[hex.EncodeToString(sum[:])] {
				hits = append(hits, fmt.Sprintf("%s:%d", label, i+1))
			}
		}
	}
	return hits
}

// TestVendoredCorpusHasNoLeakedOGUUID fails if any vendored fixture, in any
// field (body, headers, metadata or request), carries the leaked UUID.
func TestVendoredCorpusHasNoLeakedOGUUID(t *testing.T) {
	var hits []string
	for _, e := range walkVendoredFixtureCorpus(t) {
		data, err := os.ReadFile(e.fullPath)
		if err != nil {
			t.Fatalf("reading %s: %v", e.fullPath, err)
		}
		hits = append(hits, leakedUUIDHits(e.relPath, data, leakedOGUUIDSHA256)...)
	}
	if len(hits) > 0 {
		sort.Strings(hits)
		t.Fatalf("%d vendored fixture line(s) carry the leaked organization group UUID; re-anonymize them:\n%s", len(hits), joinLines(hits))
	}
}

// TestTrackedFilesHaveNoLeakedOGUUID fails if any git-tracked file outside
// leakedOGUUIDExemptFiles carries the leaked UUID. Skips outside a git
// checkout (the public package).
func TestTrackedFilesHaveNoLeakedOGUUID(t *testing.T) {
	out, err := exec.Command("git", "-C", "..", "ls-files", "-z").Output()
	if err != nil {
		t.Skipf("git ls-files unavailable (%v); this guard needs the private checkout", err)
	}
	var hits []string
	for _, rel := range strings.Split(strings.TrimRight(string(out), "\x00"), "\x00") {
		if rel == "" || leakedOGUUIDExemptFiles[rel] != "" {
			continue
		}
		data, err := os.ReadFile(filepath.Join("..", rel))
		if err != nil {
			continue // deleted in the working tree
		}
		hits = append(hits, leakedUUIDHits(rel, data, leakedOGUUIDSHA256)...)
	}
	if len(hits) > 0 {
		sort.Strings(hits)
		t.Fatalf("%d tracked line(s) carry the leaked organization group UUID:\n%s", len(hits), joinLines(hits))
	}
}

// TestLeakedOGUUIDExemptFilesAreNotShipped proves the exemption cannot hide
// a shipped leak: no exempt file is under a sync carry-over path or the
// overlay root in either manifest, and, when assembled packages are staged,
// none is present in them.
func TestLeakedOGUUIDExemptFilesAreNotShipped(t *testing.T) {
	checked := 0
	for _, mf := range []string{"../tools/sync/manifest.yaml", "../tools/sync/manifest-python.yaml"} {
		data, err := os.ReadFile(mf)
		if err != nil {
			continue
		}
		var raw struct {
			CarryOver   []yaml.Node `yaml:"carry_over"`
			OverlayRoot string      `yaml:"overlay_root"`
		}
		if err := yaml.Unmarshal(data, &raw); err != nil {
			t.Fatalf("parsing %s: %v", mf, err)
		}
		var paths []string
		for _, n := range raw.CarryOver {
			var p string
			if n.Kind == yaml.ScalarNode {
				p = n.Value
			} else {
				var e struct {
					Src string `yaml:"src"`
				}
				if err := n.Decode(&e); err != nil {
					t.Fatalf("parsing a carry_over entry in %s: %v", mf, err)
				}
				p = e.Src
			}
			if p == "" {
				t.Fatalf("%s: a carry_over entry has no recognizable path; update this guard's parser", mf)
			}
			paths = append(paths, p)
		}
		if raw.OverlayRoot != "" {
			paths = append(paths, raw.OverlayRoot)
		}
		for exempt := range leakedOGUUIDExemptFiles {
			for _, p := range paths {
				if exempt == p || strings.HasPrefix(exempt, strings.TrimSuffix(p, "/")+"/") {
					t.Errorf("%s: exempt file %s is under shipped path %q", mf, exempt, p)
				}
			}
		}
		checked++
	}
	if checked == 0 {
		t.Skip("tools/sync manifests are not present (public package)")
	}
	for _, env := range []string{"ASSEMBLED_PUBLIC_GO_DIR", "ASSEMBLED_PUBLIC_PY_DIR"} {
		dir := strings.TrimSpace(os.Getenv(env))
		if dir == "" {
			continue
		}
		for exempt := range leakedOGUUIDExemptFiles {
			if _, err := os.Stat(filepath.Join(dir, exempt)); err == nil {
				t.Errorf("exempt file %s is present in the assembled package at %s", exempt, env)
			}
		}
	}
}

// TestLeakedUUIDHits_PlantedControls proves the hash match fires on a
// planted UUID given its hash (in either case), and ignores other UUIDs.
func TestLeakedUUIDHits_PlantedControls(t *testing.T) {
	planted := strings.Join([]string{"9f3c2a71", "4b0e", "4d6a", "8c1f", "2e7b9d05a346"}, "-")
	sum := sha256.Sum256([]byte(planted))
	hashes := map[string]bool{hex.EncodeToString(sum[:]): true}
	other := strings.Join([]string{"00000000", "0000", "0000", "0000", "000000000000"}, "-")

	if hits := leakedUUIDHits("x", []byte(`{"a": "`+planted+`"}`), hashes); len(hits) != 1 {
		t.Fatalf("planted UUID: got %v, want one hit", hits)
	}
	if hits := leakedUUIDHits("x", []byte("/groups/"+strings.ToUpper(planted)+"/b"), hashes); len(hits) != 1 {
		t.Fatalf("upper-case planted UUID inside a URL: got %v, want one hit", hits)
	}
	if hits := leakedUUIDHits("x", []byte(other), hashes); len(hits) != 0 {
		t.Fatalf("unrelated UUID: got %v, want none", hits)
	}
}

// leakedOGUUIDExtraHashesEnv lets a caller add sha256 hashes to the deny list for
// one run (comma-separated). The make public-* targets never set it; the negative
// control of the assembled-package guard does, to plant a value without the real
// one ever appearing in a file.
const leakedOGUUIDExtraHashesEnv = "LEAKED_OG_UUID_SHA256_EXTRA"

// assembledLeakedUUIDHits scans EVERY file under dir (any extension, including
// READMEs and scripts, skipping only .git) for a deny-listed dash-form UUID. A
// walk or read error is returned, never skipped: a guard must not fail open.
func assembledLeakedUUIDHits(dir string, hashes map[string]bool) ([]string, error) {
	var hits []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		hits = append(hits, leakedUUIDHits(filepath.ToSlash(rel), data, hashes)...)
		return nil
	})
	sort.Strings(hits)
	return hits, err
}

// TestAssembledPackagesHaveNoLeakedOGUUID is the deny-list guard for the SHIPPED
// trees: it scans every file of the assembled Go and Python packages
// (ASSEMBLED_PUBLIC_GO_DIR / ASSEMBLED_PUBLIC_PY_DIR), where the vendored corpus,
// its README and the overlay all land, not only the fixture JSON that
// TestVendoredCorpusHasNoLeakedOGUUID reads. The make public-* targets run it on
// the assembled tree before any branch or commit is created.
func TestAssembledPackagesHaveNoLeakedOGUUID(t *testing.T) {
	goDir, pyDir := assembledPackageDirs(t)
	hashes := map[string]bool{}
	for h := range leakedOGUUIDSHA256 {
		hashes[h] = true
	}
	for _, h := range strings.Split(os.Getenv(leakedOGUUIDExtraHashesEnv), ",") {
		if h = strings.TrimSpace(h); h != "" {
			hashes[h] = true
		}
	}
	var hits []string
	for label, dir := range map[string]string{"go": goDir, "python": pyDir} {
		if dir == "" {
			continue
		}
		got, err := assembledLeakedUUIDHits(dir, hashes)
		if err != nil {
			t.Fatalf("scanning the assembled %s package %s: %v", label, dir, err)
		}
		for _, h := range got {
			hits = append(hits, label+": "+h)
		}
	}
	if len(hits) > 0 {
		sort.Strings(hits)
		t.Fatalf("%d line(s) of the assembled package(s) carry the leaked organization group UUID (an old or unreviewed corpus?):\n%s", len(hits), joinLines(hits))
	}
}

// TestAssembledLeakedUUIDScan_PlantedControls proves the scan fires on a planted
// deny-listed value in any file of an assembled tree (including a README, where the
// real leak was), stays quiet on a clean tree, and fails on an unreadable one.
func TestAssembledLeakedUUIDScan_PlantedControls(t *testing.T) {
	planted := strings.Join([]string{"9f3c2a71", "4b0e", "4d6a", "8c1f", "2e7b9d05a346"}, "-")
	sum := sha256.Sum256([]byte(planted))
	hashes := map[string]bool{hex.EncodeToString(sum[:]): true}

	dir := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("testdata/mock-responses/a.json", `{"ok": true}`)
	write("testdata/mock-responses/README.md", "line one\nthe group "+strings.ToUpper(planted)+" here\n")
	write(".git/ignored", planted)

	hits, err := assembledLeakedUUIDHits(dir, hashes)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(hits) != 1 || hits[0] != "testdata/mock-responses/README.md:2" {
		t.Fatalf("hits = %v, want exactly the planted README line", hits)
	}

	clean := t.TempDir()
	if err := os.WriteFile(filepath.Join(clean, "a.txt"), []byte("nothing here\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if hits, err := assembledLeakedUUIDHits(clean, hashes); err != nil || len(hits) != 0 {
		t.Fatalf("clean tree: hits=%v err=%v", hits, err)
	}
	if _, err := assembledLeakedUUIDHits(filepath.Join(clean, "missing"), hashes); err == nil {
		t.Fatal("a missing directory was scanned without error (the guard would fail open)")
	}
}
