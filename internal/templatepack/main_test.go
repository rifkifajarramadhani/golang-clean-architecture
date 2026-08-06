package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestBuildUsesManifestAndOnlyTrackedFiles(t *testing.T) {
	root := newTestRepository(t)
	outputDir := filepath.Join(root, "dist")

	archive, err := build(root, outputDir, "")
	if err != nil {
		t.Fatal(err)
	}

	if filepath.Base(archive) != archivePrefix+"0.2.0.tar.gz" {
		t.Fatalf("archive = %q", archive)
	}

	first, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}

	tracked := filepath.Join(root, "tracked.txt")
	if err := os.Chtimes(tracked, time.Now().Add(-time.Hour), time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	if _, err := build(root, outputDir, ""); err != nil {
		t.Fatal(err)
	}

	second, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(first, second) {
		t.Fatal("repeated builds produced different archives")
	}

	want := []string{".craft/manifest.json", ".gitignore", "CONTEXT.md", "tracked.txt"}
	if got := archiveNames(t, archive); !slices.Equal(got, want) {
		t.Fatalf("archive entries = %q, want %q", got, want)
	}

	verifyChecksum(t, archive)
}

func TestBuildPreservesExplicitVersionOverride(t *testing.T) {
	root := newTestRepository(t)

	archive, err := build(root, filepath.Join(root, "release"), "9.8.7")
	if err != nil {
		t.Fatal(err)
	}

	if filepath.Base(archive) != archivePrefix+"9.8.7.tar.gz" {
		t.Fatalf("archive = %q", archive)
	}
}

func TestValidateVersionRejectsNonSemanticVersion(t *testing.T) {
	for _, version := range []string{"", "1", "1.2", "v1.2.3", "1.2.x"} {
		if err := validateVersion(version); err == nil {
			t.Fatalf("validateVersion(%q) succeeded", version)
		}
	}
}

func newTestRepository(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".craft"), 0o755); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		".craft/manifest.json": "{\"template_version\":\"0.2.0\"}\n",
		".gitignore":           "*.log\ndist/\nrelease/\n",
		"CONTEXT.md":           "# Context\n",
		"tracked.txt":          "tracked\n",
		"untracked.txt":        "untracked\n",
		"credentials.log":      "ignored\n",
	}
	for name, contents := range files {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	runGit(t, root, "init", "--quiet")
	runGit(t, root, "add", ".craft/manifest.json", ".gitignore", "CONTEXT.md", "tracked.txt")

	return root
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()

	command := exec.Command("git", args...)
	command.Dir = root

	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, output)
	}
}

func archiveNames(t *testing.T, filename string) []string {
	t.Helper()

	input, err := os.Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := input.Close(); err != nil {
			t.Error(err)
		}
	}()

	compressed, err := gzip.NewReader(input)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := compressed.Close(); err != nil {
			t.Error(err)
		}
	}()

	reader := tar.NewReader(compressed)

	var names []string

	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			t.Fatal(err)
		}

		names = append(names, header.Name)
	}

	return names
}

func verifyChecksum(t *testing.T, archive string) {
	t.Helper()

	data, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}

	checksum, err := os.ReadFile(archive + ".sha256")
	if err != nil {
		t.Fatal(err)
	}

	sum := sha256.Sum256(data)

	want := hex.EncodeToString(sum[:]) + "  " + filepath.Base(archive) + "\n"
	if string(checksum) != want {
		t.Fatalf("checksum = %q, want %q", checksum, want)
	}
}
