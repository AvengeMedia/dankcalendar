// Package shellembed carries the quickshell UI inside the dcal binary and
// materializes it at runtime via dankgo/shellapp/shellfs, since quickshell
// needs a real filesystem path. Customization goes through -c /
// DANKCAL_SHELL_DIR instead of editing the extraction.
package shellembed

import (
	"io/fs"
	"path"

	"github.com/AvengeMedia/dankgo/shellapp/shellfs"
)

const (
	distRoot   = "dist"
	shellEntry = "shell.qml"
)

// Available reports whether this binary was built with the embedded UI
// (the withshell build tag).
func Available() bool {
	info, err := fs.Stat(distFS, path.Join(distRoot, shellEntry))
	return err == nil && !info.IsDir()
}

func Extract(baseDir string) (string, error) {
	sub, err := fs.Sub(distFS, distRoot)
	if err != nil {
		return "", err
	}
	return shellfs.Extract(sub, baseDir)
}

func Prune(baseDir, keep string) {
	shellfs.Prune(baseDir, keep)
}
