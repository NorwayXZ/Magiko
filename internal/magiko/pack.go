package magiko

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/yetone/magpie/internal/appdir"
)

// These MIT skills were extracted from LeilaCrak. They are data, not
// startup instructions; neither they nor their helper scripts run on import.
//
//go:embed skills
var pack embed.FS
var packMu sync.Mutex

var skillNames = []string{"protocol-reverse-engineering", "reverse-engineering-api"}

func Skills() []string { return append([]string(nil), skillNames...) }

// StageSkills extracts an immutable, versioned local source. Existing files
// must match: never overwrite a user's edits or follow a substituted symlink.
func StageSkills() (string, error) {
	packMu.Lock()
	defer packMu.Unlock()
	if err := os.MkdirAll(appdir.Config(), 0700); err != nil {
		return "", err
	}
	root := filepath.Join(appdir.Config(), "magiko-pack-v1")
	err := fs.WalkDir(pack, "skills", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel("skills", path)
		if err != nil {
			return err
		}
		dst := filepath.Join(root, rel)
		if d.IsDir() {
			if err := os.Mkdir(dst, 0700); err != nil && !os.IsExist(err) {
				return err
			}
			info, err := os.Lstat(dst)
			if err != nil {
				return err
			}
			if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("skill source is not a regular directory: %s", dst)
			}
			return nil
		}
		b, err := pack.ReadFile(path)
		if err != nil {
			return err
		}
		f, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if os.IsExist(err) {
			info, e := os.Lstat(dst)
			if e != nil {
				return e
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("skill source is not a regular file: %s", dst)
			}
			old, e := os.ReadFile(dst)
			if e != nil {
				return e
			}
			if !bytes.Equal(old, b) {
				return fmt.Errorf("skill source was edited; left unchanged: %s", dst)
			}
			return nil
		}
		if err != nil {
			return err
		}
		_, writeErr := f.Write(b)
		closeErr := f.Close()
		if writeErr != nil {
			os.Remove(dst)
			return writeErr
		}
		if closeErr != nil {
			return closeErr
		}
		return nil
	})
	return root, err
}
