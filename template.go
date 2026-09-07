package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// templateDir specifies where templates are stored.
var templateDir string

// templateNames returns the names of saved templates in lexical order.
// If [templateDir] does not exist, it returns an empty list.
func templateNames() ([]string, error) {
	ents, err := os.ReadDir(templateDir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	names := make([]string, len(ents))
	for i, ent := range ents {
		names[i] = ent.Name()
	}
	return names, nil
}

// templateByName returns file information for the named template
// without following symbolic links.
func templateByName(name string) (fs.FileInfo, error) {
	path, err := templatePath(name)
	if err != nil {
		return nil, err
	}

	fi, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%s: no such template", name)
	}
	return fi, err
}

// addTemplate adds src as a template named name.
// If name is empty, the base name of src is used.
func addTemplate(src, name string) error {
	if name == "" {
		abs, err := filepath.Abs(src)
		if err != nil {
			return err
		}
		name = filepath.Base(abs)
	}

	dst, err := templatePath(name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(templateDir, 0o700); err != nil {
		return err
	}
	return copyPath(src, dst)
}

func removeTemplate(name string) error {
	if _, err := templateByName(name); err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(templateDir, name))
}

func createFromTemplate(name, dst string) error {
	if _, err := templateByName(name); err != nil {
		return err
	}
	return copyPath(filepath.Join(templateDir, name), dst)
}

// copyPath copies a regular file, directory, or symbolic link from src to dst.
// It returns an error if dst already exists and attempts to remove an
// incomplete dst if copying fails.
func copyPath(src, dst string) error {
	fi, err := os.Lstat(src)
	if err != nil {
		return err
	}

	mode := fi.Mode()
	switch mode & os.ModeType {
	case os.ModeDir:
		// A lexical check should be enough for normal use. Resolving
		// symlinks would add complexity for little practical benefit.
		srcPath, _ := filepath.Abs(src)
		dstPath, _ := filepath.Abs(dst)
		if rel, err := filepath.Rel(srcPath, dstPath); err == nil && filepath.IsLocal(rel) {
			return fmt.Errorf("%s: destination is inside source", dst)
		}

		root, err := os.OpenRoot(src)
		if err != nil {
			return err
		}
		defer root.Close()

		// Create dst first so [os.CopyFS] cannot merge into an existing directory.
		if err := os.Mkdir(dst, 0o777); err != nil {
			return err
		}
		if err := os.CopyFS(dst, root.FS()); err != nil {
			os.RemoveAll(dst)
			return err
		}
		return nil
	case os.ModeSymlink:
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		return os.Symlink(target, dst)
	case 0:
		r, err := os.Open(src)
		if err != nil {
			return err
		}
		defer r.Close()
		// Preserve executable bits from src to match [os.CopyFS].
		w, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o666|mode&0o777)
		if err != nil {
			return err
		}
		if _, err := io.Copy(w, r); err != nil {
			w.Close()
			os.Remove(dst)
			return err
		}
		if err := w.Close(); err != nil {
			os.Remove(dst)
			return err
		}
		return nil
	default:
		return fmt.Errorf("%s: unsupported file type", src)
	}
}

// templatePath returns the path for name inside [templateDir].
// Name must be a single local path element other than ".".
func templatePath(name string) (string, error) {
	if name == "." || !filepath.IsLocal(name) || filepath.Base(name) != name {
		return "", fmt.Errorf("%s: invalid template name", name)
	}
	return filepath.Join(templateDir, name), nil
}
