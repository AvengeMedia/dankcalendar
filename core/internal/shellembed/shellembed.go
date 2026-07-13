// Package shellembed carries the quickshell UI inside the dcal binary and
// materializes it at runtime, since quickshell needs a real filesystem
// path. The tree is extracted read-only into the runtime dir keyed by
// content hash and verified on every resolution, so local edits never
// stick — customization goes through -c / DANKCAL_SHELL_DIR instead.
package shellembed

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
)

const (
	distRoot   = "dist"
	shellEntry = "shell.qml"
	hashLen    = 16
)

// Available reports whether this binary was built with the embedded UI
// (the withshell build tag).
func Available() bool {
	info, err := fs.Stat(distFS, path.Join(distRoot, shellEntry))
	return err == nil && !info.IsDir()
}

// Extract materializes the embedded UI under baseDir, keyed by a content
// hash, and returns the directory containing shell.qml. An existing
// extraction is re-verified against the embedded content and replaced if
// it was modified.
func Extract(baseDir string) (string, error) {
	sub, err := fs.Sub(distFS, distRoot)
	if err != nil {
		return "", err
	}
	return extract(sub, baseDir)
}

// Prune removes previously extracted UI revisions under baseDir, keeping
// keep. Callers must ensure no running instance still uses the others.
func Prune(baseDir, keep string) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		p := filepath.Join(baseDir, entry.Name())
		if !entry.IsDir() || p == keep {
			continue
		}
		forceRemoveAll(p)
	}
}

func extract(fsys fs.FS, baseDir string) (string, error) {
	hash, err := hashFS(fsys)
	if err != nil {
		return "", fmt.Errorf("hash embedded UI: %w", err)
	}

	target := filepath.Join(baseDir, hash)
	if verify(target, hash) {
		return target, nil
	}
	forceRemoveAll(target)

	if err := os.MkdirAll(baseDir, 0o700); err != nil {
		return "", err
	}

	tmp, err := os.MkdirTemp(baseDir, ".extract-")
	if err != nil {
		return "", err
	}
	defer forceRemoveAll(tmp)

	if err := os.CopyFS(tmp, fsys); err != nil {
		return "", fmt.Errorf("extract embedded UI: %w", err)
	}
	if err := makeReadOnly(tmp); err != nil {
		return "", fmt.Errorf("extract embedded UI: %w", err)
	}

	if err := os.Rename(tmp, target); err != nil {
		// A concurrent invocation may have extracted the same revision first.
		if verify(target, hash) {
			return target, nil
		}
		return "", err
	}
	return target, nil
}

// verify reports whether dir holds an intact extraction of content hash.
func verify(dir, hash string) bool {
	if _, err := os.Stat(dir); err != nil {
		return false
	}
	got, err := hashFS(os.DirFS(dir))
	return err == nil && got == hash
}

func hashFS(fsys fs.FS) (string, error) {
	h := sha256.New()
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		f, err := fsys.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()
		h.Write([]byte(p))
		h.Write([]byte{0})
		_, err = io.Copy(h, f)
		return err
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil))[:hashLen], nil
}

func makeReadOnly(root string) error {
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.Chmod(p, 0o555)
		}
		return os.Chmod(p, 0o444)
	})
}

// forceRemoveAll removes a read-only extraction, restoring dir write
// permission first so unlinking succeeds.
func forceRemoveAll(root string) {
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() {
			_ = os.Chmod(p, 0o755)
		}
		return nil
	})
	os.RemoveAll(root)
}
