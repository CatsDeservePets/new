package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOutputPath(t *testing.T) {
	chdirTemp(t)
	writeFile(t, "file", "")
	if err := os.Mkdir("dir", 0o777); err != nil {
		t.Fatal(err)
	}

	const template = "template"
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name:   "EmptyOutput",
			output: "",
			want:   template,
		},
		{
			name:   "TrailingSeparator",
			output: "missing" + string(os.PathSeparator),
			want:   filepath.Join("missing", template),
		},
		{
			name:   "ExistingDirectory",
			output: "dir",
			want:   filepath.Join("dir", template),
		},
		{
			name:   "ExistingFile",
			output: "file",
			want:   "file",
		},
		{
			name:   "MissingFile",
			output: "missing",
			want:   "missing",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := outputPath(template, test.output); got != test.want {
				t.Errorf("outputPath(%q, %q) = %q, want %q", template, test.output, got, test.want)
			}
		})
	}
}

func TestConfigDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	got, err := configDir()
	if err != nil {
		t.Fatal(err)
	}
	if got != dir {
		t.Errorf("configDir() = %q, want %q", got, dir)
	}

	t.Setenv("XDG_CONFIG_HOME", "relative")

	if _, err := configDir(); err == nil {
		t.Error("configDir() with relative $XDG_CONFIG_HOME returned nil error")
	}
}
