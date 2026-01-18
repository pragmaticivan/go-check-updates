# go-check-updates (gcu)

`gcu` is a Go module upgrade tool inspired by `npm-check-updates`. It helps you identify available updates for your Go dependencies and upgrade them easily.

## Features

-   **List Updates**: Check for newer versions of your direct and indirect dependencies.
-   **Interactive Mode**: Select which packages to upgrade using a terminal UI.
-   **Upgrade**: Update all packages to their latest versions.
-   **Filter**: Filter packages by name.
-   **Cooldown**: Ignore versions published within the last N days.
-   **Formats**: Customize output for scripting or readability (`--format lines`, `--format group`, `--format time`).
-   **Transitive (optional)**: Include transitive dependency updates with `--all`.

## Installation

```bash
go install github.com/pragmaticivan/go-check-updates/cmd/gcu@latest
```

Or build from source:

```bash
git clone https://github.com/pragmaticivan/go-check-updates.git
cd go-check-updates
go build -o gcu ./cmd/gcu
```

## Usage

### Check for updates (Dry Run)

Run `gcu` in the root of your project:

```bash
$ gcu
Checking for updates...

Available updates:
 github.com/charmbracelet/bubbletea  v0.23.1  →  v0.23.2
 github.com/spf13/cobra              v1.6.0   →  v1.6.1

Run with -u to upgrade, or -i for interactive mode.
```

### Upgrade all packages

Use the `-u` flag to upgrade all found updates:

```bash
$ gcu -u
Checking for updates...
Upgrading...
Done.
```

### Interactive Mode

Use the `-i` flag to select packages to upgrade:

```bash
$ gcu -i
```

You will see a checklist:
```
Which packages would you like to update?

  [x] github.com/charmbracelet/bubbletea v0.23.1 -> v0.23.2
> [ ] github.com/spf13/cobra             v1.6.0  -> v1.6.1

Press <space> to select, <enter> to update, <q> to quit.
```

### Filtering

Check only specific packages:

```bash
$ gcu -f charm
Checking for updates...

Available updates:
 github.com/charmbracelet/bubbletea  v0.23.1  →  v0.23.2
```

### Include transitive dependencies

By default, `gcu` shows updates for modules explicitly listed in `go.mod` (split into direct vs `// indirect`).
To include transitive dependency updates too:

```bash
$ gcu --all
```

### Cooldown

Ignore versions published in the last N days:

```bash
$ gcu --cooldown 30
```

### Output formatting

Supported values: `group`, `lines`, `time` (comma-separated).

- Pipe-friendly lines (prints only `module@version`):

```bash
$ gcu --format lines
github.com/foo/bar@v1.2.3
```

- Group by update type (major/minor/patch):

```bash
$ gcu --format group
```

- Show publish time (date + days ago):

```bash
$ gcu --format time
```

- Combine formats:

```bash
$ gcu --format group,time
```

## How it works

1.  `gcu` runs `go list -m -u -json all` to detect available updates.
2.  If `-u` is passed, it runs `go get [package]@[new_version]` for each update.
3.  Finally, it runs `go mod tidy` to ensure `go.mod` and `go.sum` are clean.
