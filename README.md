# new

`new` is a simple tool for creating files and directories from templates.
Think of it as a command-line version of the `New` context menu.

## Installation

```shell
go install github.com/CatsDeservePets/new@latest
```

## Usage

```
usage: new [-o path] [template]
       new command [argument ...]

new creates files and directories from templates.
If template is omitted, new prompts you to choose one.

Options:
  -o path
    	write output to path

Commands:
  add     add a template
  list    list templates
  remove  remove a template
  show    show template information

Use 'new command -h' for more information about a command.
Templates are stored in $XDG_CONFIG_HOME/new/templates.
```

<details>

<summary>Command-specific usage</summary>

```
usage: new add [-n name] path

Add the file or directory at path as a template.

Options:
  -n name
    	set the template name
```

```
usage: new list

List the names of saved templates.
```

```
usage: new remove template

Remove the named template.
```

```
usage: new show template

Show information about the named template.
```

</details>

## Examples

Choose a template by name:

```shell
$ new main.go
```

Choose a template from the built-in menu:

```shell
$ new
Select a template:
  1. .hushlogin
  2. Dockerfile
  3. README.md
  4. beemovie.txt
  5. fork-bomb.sh
  6. main.go
  7. node_modules
  8. untitled.txt

Selection (q to cancel): 7
```

Choose a template using `fzf`:

```shell
$ new "$(new list | fzf)"
```

Specify the output path:

```shell
$ new -o app.go main.go
$ new -o cmd/example/ main.go
```

Add templates:

```shell
$ new add Dockerfile
$ new add -n project .
```
