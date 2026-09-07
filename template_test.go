package main

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestTemplatePath(t *testing.T) {
	templateDir = t.TempDir()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "ValidName",
			input: "template",
			want:  filepath.Join(templateDir, "template"),
		},
		{
			name:    "Dot",
			input:   ".",
			wantErr: true,
		},
		{
			name:    "ParentDirectory",
			input:   "..",
			wantErr: true,
		},
		{
			name:    "NestedPath",
			input:   filepath.Join("dir", "template"),
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := templatePath(test.input)
			if (err != nil) != test.wantErr {
				t.Errorf("templatePath(%q) error = %v, want error presence = %t", test.input, err, test.wantErr)
				return
			}
			if err != nil {
				return
			}
			if got != test.want {
				t.Errorf("templatePath(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestTemplateNames(t *testing.T) {
	templateDir = filepath.Join(t.TempDir(), "templates")

	names, err := templateNames()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 0 {
		t.Errorf("templateNames() = %v, want no templates", names)
	}

	for _, name := range []string{"c", "a", "b"} {
		writeFile(t, filepath.Join(templateDir, name), "")
	}

	names, err = templateNames()
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"a", "b", "c"}
	if !slices.Equal(names, want) {
		t.Errorf("templateNames() = %v, want %v", names, want)
	}
}

func TestAddTemplate(t *testing.T) {
	root := t.TempDir()
	templateDir = filepath.Join(root, "templates")

	src := filepath.Join(root, "source")
	writeFile(t, src, "hello")

	if err := addTemplate(src, ""); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(templateDir, "source")); got != "hello" {
		t.Errorf("default template contents = %q, want %q", got, "hello")
	}

	if err := addTemplate(src, "template"); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(templateDir, "template")); got != "hello" {
		t.Errorf("named template contents = %q, want %q", got, "hello")
	}

	project := filepath.Join(root, "project")
	writeFile(t, filepath.Join(project, "main.go"), "package main\n")
	t.Chdir(project)

	if err := addTemplate(".", ""); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(templateDir, "project", "main.go")); got != "package main\n" {
		t.Errorf("directory template contents = %q, want %q", got, "package main\n")
	}
}

func TestCopyPath(t *testing.T) {
	const src, dst = "src", "dst"

	t.Run("RegularFile", func(t *testing.T) {
		chdirTemp(t)
		writeFile(t, src, "hello")

		if err := copyPath(src, dst); err != nil {
			t.Fatal(err)
		}
		if got := readFile(t, dst); got != "hello" {
			t.Errorf("ReadFile(%q) = %q, want %q", dst, got, "hello")
		}
	})

	t.Run("Directory", func(t *testing.T) {
		chdirTemp(t)
		writeFile(t, "src/a.txt", "a")
		writeFile(t, "src/dir/b.txt", "b")

		if err := copyPath(src, dst); err != nil {
			t.Fatal(err)
		}
		if got := readFile(t, "dst/a.txt"); got != "a" {
			t.Errorf("ReadFile(%q) = %q, want %q", "dst/a.txt", got, "a")
		}
		if got := readFile(t, "dst/dir/b.txt"); got != "b" {
			t.Errorf("ReadFile(%q) = %q, want %q", "dst/dir/b.txt", got, "b")
		}
	})

	t.Run("Symlink", func(t *testing.T) {
		chdirTemp(t)

		if err := os.Symlink("target", src); err != nil {
			t.Skip(err)
		}

		if err := copyPath(src, dst); err != nil {
			t.Fatal(err)
		}

		got, err := os.Readlink(dst)
		if err != nil {
			t.Fatal(err)
		}
		if got != "target" {
			t.Errorf("Readlink(%q) = %q, want %q", dst, got, "target")
		}
	})

	t.Run("ExistingFile", func(t *testing.T) {
		chdirTemp(t)
		writeFile(t, src, "new")
		writeFile(t, dst, "old")

		err := copyPath(src, dst)
		if !errors.Is(err, os.ErrExist) {
			t.Errorf("copyPath(%q, %q) error = %v, want %v", src, dst, err, os.ErrExist)
		}
		if got := readFile(t, dst); got != "old" {
			t.Errorf("ReadFile(%q) = %q, want %q", dst, got, "old")
		}
	})

	t.Run("ExistingDirectory", func(t *testing.T) {
		chdirTemp(t)
		writeFile(t, "src/new", "new")
		writeFile(t, "dst/old", "old")

		err := copyPath(src, dst)
		if !errors.Is(err, os.ErrExist) {
			t.Errorf("copyPath(%q, %q) error = %v, want %v", src, dst, err, os.ErrExist)
		}
		if got := readFile(t, "dst/old"); got != "old" {
			t.Errorf("ReadFile(%q) = %q, want %q", "dst/old", got, "old")
		}
		if _, err := os.Lstat(filepath.Join("dst", "new")); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("Lstat(%q) error = %v, want %v", "dst/new", err, os.ErrNotExist)
		}
	})
}

func chdirTemp(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
}

func writeFile(t *testing.T, name, contents string) {
	t.Helper()

	name = filepath.FromSlash(name)
	if err := os.MkdirAll(filepath.Dir(name), 0o777); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(contents), 0o666); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, name string) string {
	t.Helper()

	b, err := os.ReadFile(filepath.FromSlash(name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
