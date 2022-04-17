<!-- {{.Header}} -->
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
{{.Usage}}
```
