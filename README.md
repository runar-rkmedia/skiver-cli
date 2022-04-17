<!-- This file is generated. -->
# Skiver-cli

CLI accompanying [Skiver](https://githbub.com/runar-rkmedia/skiver), a translation-management solution.

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
      --config string                 config file (default is $HOME/skiver/skiver-cli.yaml)
  -h, --help                          help for Skiver-CLI
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
