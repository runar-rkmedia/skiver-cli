<!-- This file is generated. -->
# Skiver-cli

CLI accompanying [Skiver](https://github.com/runar-rkmedia/skiver), a translation-management solution.

## Installation

### Windows ([Scoop](https://scoop.sh/))

```
scoop bucket add rkmedia github.com/runar-rkmedia/scoops.git
scoop install rkmedia/skiver
```

### Linux, macOS

An install-script is available:

```shell-script
curl https://raw.githubusercontent.com/runar-rkmedia/skiver-cli/main/install.sh | sh
```

### Compile
```
go install github.com/runar-rkmedia/skiver-cli@latest
mv "$(which skiver-cli)" "$(which skiver-cli | sed 's/skiver-cli$/skiver/')"
```

### Manual from release

The [release-binaries are available for manual download](https://github.com/runar-rkmedia/skiver-cli/releases/latest/).


1. Download the release
2. Unarchive it
3. make it executable with `chmod +x <skiver.cli>`
4. Move it into your PATH

### Usage

```
Interactions with skiver, a developer-focused translation-service

Usage:
  Skiver-CLI [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  config      Show information about the current configuration, or create a new one
  generate    Generate files for the project
  help        Help about any command
  import      Import from local i18n-file
  inject      Inject comments into source-code for locale-usage, with rich descriptions
  unused      Find unused translations

Flags:
      --color string                  Force set color output. one of 'auto', 'always', 'none'.
      --config string                 config file (default is $HOME/skiver/skiver-cli.yaml)
  -h, --help                          help for Skiver-CLI
      --highlight-style string        Highlighting-style to use. See https://github.com/alecthomas/chroma/tree/master/styles for valid styles
      --ignore-filter strings         Ignore-filter for files
  -l, --locale string                 Locale to use
      --log-format string             Format to log as (default "human")
      --log-level string              Level for logging. (default "info")
      --prettier-d-slim-path string   Path-override for prettier_d_slim, which should be faster than regular prettier (default "prettier_d_slim")
      --prettier-path string          Path-override for prettier (default "prettier")
  -p, --project string                Project-id/ShortName
  -t, --token string                  Token used for authentication
  -u, --uri string                    Endpoint for skiver
      --with-prettier                 Where available, will attempt to run prettier, or prettier_d if available

Use "Skiver-CLI [command] --help" for more information about a command.

```

### Configuration

Configuration-files can be used in various formats, including `yml`, `toml` and `json`.

A benefit of using `toml` here is that it can be generated with comments included.

The file should be named `skiver-cli.toml` (change the extension to fit your preferred format).

The can store the file in your home-directory/config-directory, or in the current path, which is what you want in most cases when working with projects.

An initial file can be generated with:

```shell-script
skiver config new
```

#### Current Configuration-file (toml)

```toml
# Force set color output. one of 'auto', 'always', 'none'.
color = ""
# Highlighting-style to use. See https://github.com/alecthomas/chroma/tree/master/styles for valid styles
highlight_style = ""
# Ignore-filter for files
ignore_filter = []
# Locale to use
locale = ""
# Format to log as
log_format = ""
# Level for logging.
log_level = ""
# Path-override for prettier_d_slim, which should be faster than regular prettier
prettier_d_slim_path = ""
# Path-override for prettier
prettier_path = ""
# Project-id/ShortName
project = ""
# Token used for authentication
token = ""
# Endpoint for skiver
uri = ""
# Where available, will attempt to run prettier, or prettier_d if available
with_prettier = false

# Configuration
[config]
  format = ""

# Generate files from project etc.
[generate]
  # Generate files from export. Common formats are: i18n,tKeys.
  format = ""
  # Ouput file to write to
  path = ""

# Import from file
[import]
  # Source-file for import
  source = ""

# Inject helper-comments into source-files
[inject]
  # Directory for source-code
  dir = ""
  # Enable dry-run
  dry_run = false
  # Command to run on file after replacement, like prettier
  on_replace = ""
  # Type of injection. Can be either 'comment', or 'tKeys'
  type = ""

# Find unused translation-keys
[unused]
  # Directory for source-code
  dir = ""
  # Source-file to check-against. If ommitted, the upstream project is used as source
  source = ""

```



