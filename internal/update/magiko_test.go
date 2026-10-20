//go:build magiko && !windows

package update

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMagikoCLILeavesMagpieUntouched(t *testing.T) {
	dir := t.TempDir()
	original := filepath.Join(dir, "magpie")
	if err := os.WriteFile(original, []byte("upstream command"), 0700); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(t.TempDir(), "magiko")
	if err := os.WriteFile(exe, []byte("fork executable"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := cliPlace(exe, dir); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(original)
	if err != nil || string(got) != "upstream command" {
		t.Fatal("overwrote the upstream CLI")
	}
	target, err := os.Readlink(filepath.Join(dir, "magiko"))
	if err != nil || target != exe {
		t.Fatalf("fork link: %s %v", target, err)
	}
	if got := findCLI([]string{dir}); got != filepath.Join(dir, "magiko") {
		t.Fatalf("picked upstream executable: %s", got)
	}
}
