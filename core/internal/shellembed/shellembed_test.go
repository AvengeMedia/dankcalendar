package shellembed

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func testFS() fstest.MapFS {
	return fstest.MapFS{
		"shell.qml":            &fstest.MapFile{Data: []byte("import QtQuick\n")},
		"Common/Theme.qml":     &fstest.MapFile{Data: []byte("// theme\n")},
		"translations/en.json": &fstest.MapFile{Data: []byte("{}\n")},
	}
}

// tempBase returns a temp dir whose read-only extractions are made
// writable again before t.TempDir's own cleanup.
func tempBase(t *testing.T) string {
	base := t.TempDir()
	t.Cleanup(func() {
		filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
			if err == nil && d.IsDir() {
				os.Chmod(p, 0o755)
			}
			return nil
		})
	})
	return base
}

func TestExtractIsIdempotent(t *testing.T) {
	base := tempBase(t)

	first, err := extract(testFS(), base)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if _, err := os.Stat(filepath.Join(first, "shell.qml")); err != nil {
		t.Fatalf("shell.qml missing after extract: %v", err)
	}
	if _, err := os.Stat(filepath.Join(first, "translations", "en.json")); err != nil {
		t.Fatalf("nested file missing after extract: %v", err)
	}

	second, err := extract(testFS(), base)
	if err != nil {
		t.Fatalf("re-extract: %v", err)
	}
	if second != first {
		t.Fatalf("re-extract returned %q, want %q", second, first)
	}
}

func TestExtractIsReadOnly(t *testing.T) {
	base := tempBase(t)

	dir, err := extract(testFS(), base)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "shell.qml"), []byte("hacked"), 0o644); err == nil {
		t.Fatal("expected write to extracted file to fail")
	}
	if err := os.WriteFile(filepath.Join(dir, "new.qml"), []byte("x"), 0o644); err == nil {
		t.Fatal("expected file creation in extracted dir to fail")
	}
}

func TestExtractHealsTamperedTree(t *testing.T) {
	base := tempBase(t)

	dir, err := extract(testFS(), base)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	tampered := filepath.Join(dir, "shell.qml")
	if err := os.Chmod(tampered, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tampered, []byte("hacked\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	healed, err := extract(testFS(), base)
	if err != nil {
		t.Fatalf("re-extract after tamper: %v", err)
	}
	if healed != dir {
		t.Fatalf("healed dir %q, want %q", healed, dir)
	}

	data, err := os.ReadFile(tampered)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "import QtQuick\n" {
		t.Fatalf("tampered file not restored, got %q", data)
	}
}

func TestExtractChangesDirWithContent(t *testing.T) {
	base := tempBase(t)

	first, err := extract(testFS(), base)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	changed := testFS()
	changed["shell.qml"] = &fstest.MapFile{Data: []byte("import QtQuick // v2\n")}
	second, err := extract(changed, base)
	if err != nil {
		t.Fatalf("extract changed: %v", err)
	}
	if second == first {
		t.Fatal("expected a new dir for changed content")
	}
}

func TestPruneKeepsCurrent(t *testing.T) {
	base := tempBase(t)

	old, err := extract(testFS(), base)
	if err != nil {
		t.Fatalf("extract old: %v", err)
	}

	changed := testFS()
	changed["shell.qml"] = &fstest.MapFile{Data: []byte("import QtQuick // v2\n")}
	current, err := extract(changed, base)
	if err != nil {
		t.Fatalf("extract current: %v", err)
	}

	Prune(base, current)

	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatalf("stale read-only dir not pruned: %v", err)
	}
	if _, err := os.Stat(filepath.Join(current, "shell.qml")); err != nil {
		t.Fatalf("current dir pruned: %v", err)
	}
}

func TestAvailableFalseWithoutTag(t *testing.T) {
	if Available() {
		t.Fatal("Available() should be false in untagged test builds")
	}
}
