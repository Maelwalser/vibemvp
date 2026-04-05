package skills

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Extract unpacks files from srcFS (rooted at srcDir) into destDir.
// Files already present in destDir are left untouched so local customisations
// are preserved. The destination directory is created with mode 0755 if it
// does not exist.
func Extract(destDir string, srcFS fs.FS, srcDir string) error {
	sub, err := fs.Sub(srcFS, srcDir)
	if err != nil {
		return fmt.Errorf("sub FS %s: %w", srcDir, err)
	}

	return fs.WalkDir(sub, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		dest := filepath.Join(destDir, filepath.FromSlash(path))

		if d.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}

		// Skip files that already exist — preserve user customisations.
		if _, statErr := os.Stat(dest); statErr == nil {
			return nil
		}

		data, err := fs.ReadFile(sub, path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}

		return os.WriteFile(dest, data, 0o644)
	})
}
