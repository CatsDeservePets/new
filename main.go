package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("new: ")

	dir, err := configDir()
	if err != nil {
		log.Fatalf("determining config dir: %v", err)
	}
	templateDir = filepath.Join(dir, "new", "templates")

	args := os.Args[1:]
	if len(args) > 0 {
		// If the first argument is a command, run it with the rest.
		switch args[0] {
		case "add":
			runAdd(args[1:])
			return
		case "list":
			runList(args[1:])
			return
		case "remove":
			runRemove(args[1:])
			return
		case "show":
			runShow(args[1:])
			return
		}
	}
	runNew(args)
}

// runNew creates a file or directory from a template.
// If no template is specified, it prompts for one.
func runNew(args []string) {
	var f flag.FlagSet
	var output string

	f.StringVar(&output, "o", "", "write output to `path`")
	f.Usage = func() {
		fmt.Fprint(f.Output(), `usage: new [-o path] [template]
       new command [argument ...]

new creates files and directories from templates.
If template is omitted, new prompts you to choose one.

Options:
`)
		f.PrintDefaults()
		fmt.Fprintf(f.Output(), `
Commands:
  add     add a template
  list    list templates
  remove  remove a template
  show    show template information

Use 'new command -h' for more information about a command.
Templates are stored in %s.
`, templateDir)
	}
	parseFlags(&f, args)
	if f.NArg() > 1 {
		usageError(&f, "too many arguments")
	}

	name := f.Arg(0)
	if name == "" {
		names, err := templateNames()
		if err != nil {
			log.Fatalln(err)
		}
		// An empty list is valid for [runList], but there's nothing to select here.
		if len(names) == 0 {
			log.Fatalln("no templates found; see 'new add -h'")
		}

		// TODO: Only show the menu when running interactively?
		name, err = showMenu("Select a template:", names)
		if err == errCanceled {
			return
		}
		if err != nil {
			log.Fatalln(err)
		}
	}

	if err := createFromTemplate(name, outputPath(name, output)); err != nil {
		log.Fatalln(err)
	}
}

func runAdd(args []string) {
	var f flag.FlagSet
	var name string

	f.StringVar(&name, "n", "", "set the template `name`")
	f.Usage = func() {
		fmt.Fprint(f.Output(), `usage: new add [-n name] path

Add the file or directory at path as a template.

Options:
`)
		f.PrintDefaults()
	}
	parseFlags(&f, args)
	if f.NArg() == 0 {
		usageError(&f, "missing path")
	}
	if f.NArg() > 1 {
		usageError(&f, "too many arguments")
	}

	if err := addTemplate(f.Arg(0), name); err != nil {
		log.Fatalln(err)
	}
}

func runList(args []string) {
	var f flag.FlagSet

	f.Usage = func() {
		fmt.Fprintln(f.Output(), `usage: new list

List the names of saved templates.`)
	}
	parseFlags(&f, args)
	if f.NArg() > 0 {
		usageError(&f, "too many arguments")
	}

	names, err := templateNames()
	if err != nil {
		log.Fatalln(err)
	}
	for _, name := range names {
		fmt.Println(name)
	}
}

func runRemove(args []string) {
	var f flag.FlagSet

	f.Usage = func() {
		fmt.Fprintln(f.Output(), `usage: new remove template

Remove the named template.`)
	}
	parseFlags(&f, args)
	if f.NArg() == 0 {
		usageError(&f, "missing template")
	}
	if f.NArg() > 1 {
		usageError(&f, "too many arguments")
	}

	if err := removeTemplate(f.Arg(0)); err != nil {
		log.Fatalln(err)
	}
}

func runShow(args []string) {
	var f flag.FlagSet

	f.Usage = func() {
		fmt.Fprintln(f.Output(), `usage: new show template

Show information about the named template.`)
	}
	parseFlags(&f, args)
	if f.NArg() == 0 {
		usageError(&f, "missing template")
	}
	if f.NArg() > 1 {
		usageError(&f, "too many arguments")
	}

	fi, err := templateByName(f.Arg(0))
	if err != nil {
		log.Fatalln(err)
	}
	size := fmt.Sprintf("%d bytes", fi.Size())
	path := filepath.Join(templateDir, fi.Name())
	var kind string
	switch fi.Mode() & os.ModeType {
	case os.ModeDir:
		kind = "directory"
		size = "-"
		path += string(os.PathSeparator)
	case os.ModeSymlink:
		kind = "symlink"
		target, err := os.Readlink(path)
		if err != nil {
			log.Fatalln(err)
		}
		path += " -> " + target
	case 0:
		kind = "regular file"
	default:
		kind = "unsupported"
	}

	fmt.Printf("name: %s\n", fi.Name())
	fmt.Printf("path: %s\n", path)
	fmt.Printf("size: %s\n", size)
	fmt.Printf("kind: %s\n", kind)
}

// parseFlags is a wrapper for [flag.FlagSet.Parse] that handles
// usage output and exit status for help and parse errors.
func parseFlags(f *flag.FlagSet, args []string) {
	usage := f.Usage
	f.Usage = func() {} // Suppress automatic usage.
	err := f.Parse(args)
	f.Usage = usage

	if err == flag.ErrHelp {
		f.SetOutput(os.Stdout)
		f.Usage()
		os.Exit(0)
	}
	if err != nil {
		f.Usage()
		os.Exit(2)
	}
}

func usageError(f *flag.FlagSet, s string) {
	log.Println(s)
	f.Usage()
	os.Exit(2)
}

var errCanceled = errors.New("selection canceled")

// showMenu prompts the user to select an item by number or name.
// It returns [errCanceled] if the selection is canceled.
func showMenu(prompt string, items []string) (string, error) {
	fmt.Println(prompt)
	for i, v := range items {
		fmt.Printf("  %d. %s\n", i+1, v)
	}
	fmt.Print("\nSelection (q to cancel): ")

	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		inp := strings.TrimSpace(s.Text())
		// "q" and numeric input take precedence over matching template names.
		if inp == "" || inp == "q" {
			return "", errCanceled
		}
		n, err := strconv.Atoi(inp)
		if err != nil {
			n = slices.Index(items, inp) + 1
		}
		if n < 1 || n > len(items) {
			fmt.Fprint(os.Stderr, "Invalid selection, try again: ")
			continue
		}
		return items[n-1], nil
	}
	if err := s.Err(); err != nil {
		return "", err
	}
	return "", errCanceled
}

// outputPath returns the output path for name. If output is an existing
// directory or ends in a path separator, name is appended to it.
func outputPath(name, output string) string {
	if output == "" {
		return name
	}
	// A trailing separator means output is a directory path.
	// Missing directories are not created.
	if os.IsPathSeparator(output[len(output)-1]) {
		return filepath.Join(output, name)
	}
	if fi, err := os.Stat(output); err == nil && fi.IsDir() {
		return filepath.Join(output, name)
	}
	return output
}

// configDir is like [os.UserConfigDir], but looks for $XDG_CONFIG_HOME on all
// platforms rather than just Unix.
func configDir() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		if !filepath.IsAbs(dir) {
			return "", errors.New("path in $XDG_CONFIG_HOME is relative")
		}
		return dir, nil
	}
	return os.UserConfigDir()
}
